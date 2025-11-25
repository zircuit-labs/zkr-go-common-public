package testutils

import (
	"sync"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/require"

	"github.com/zircuit-labs/zkr-go-common/config"
	"github.com/zircuit-labs/zkr-go-common/messagebus"
)

var (
	sharedEmbeddedServer *SharedEmbeddedServer
	lock                 sync.Mutex
	userCount            uint
)

type SharedEmbeddedServer struct {
	server *messagebus.NatsEmbeddedServer
}

func (s SharedEmbeddedServer) Conn(t *testing.T) (*nats.Conn, jetstream.JetStream) {
	t.Helper()
	nc, err := s.server.NewConnection()
	require.NoError(t, err)

	js, err := jetstream.New(nc)
	require.NoError(t, err)
	return nc, js
}

func (s *SharedEmbeddedServer) Close() {
	lock.Lock()
	defer lock.Unlock()

	// No-op if there's no shared server
	if sharedEmbeddedServer == nil || sharedEmbeddedServer.server == nil {
		return
	}
	if userCount > 1 {
		userCount--
		return
	}
	userCount = 0
	// Close the shared instance (avoid closing via a copied receiver)
	sharedEmbeddedServer.server.Close()
	sharedEmbeddedServer = nil
}

func NewEmbeddedServer(t *testing.T) *SharedEmbeddedServer {
	t.Helper()

	lock.Lock()
	defer lock.Unlock()

	userCount++
	if sharedEmbeddedServer != nil {
		return sharedEmbeddedServer
	}

	cfg, err := config.NewConfigurationFromMap(
		map[string]any{
			"servername": "unit_test_server",
		},
	)
	require.NoError(t, err)

	s, err := messagebus.NewNatsEmbeddedServer(cfg, "")
	require.NoError(t, err)
	sharedEmbeddedServer = &SharedEmbeddedServer{server: s}
	return sharedEmbeddedServer
}
