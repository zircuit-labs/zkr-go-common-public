package messagebus_test

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zircuit-labs/zkr-go-common/config"
	"github.com/zircuit-labs/zkr-go-common/messagebus"
)

var (
	natsServer *messagebus.NatsEmbeddedServer

	// list of streams/subjects to create for tests
	streams = map[string][]string{
		"FOO":   {"foo"},
		"BAZ":   {"baz"},
		"QUX":   {"qux"},
		"WALDO": {"waldo", "waldo.>"},
		"CORGE": {"corge.>"},
	}
)

func getNatsConnection(t *testing.T) *nats.Conn {
	t.Helper()
	nc, err := natsServer.NewConnection()
	require.NoError(t, err)
	require.NotNil(t, natsServer)
	t.Cleanup(nc.Close)
	return nc
}

func getJetStream(t *testing.T, nc *nats.Conn) jetstream.JetStream {
	t.Helper()
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	return js
}

// TestMain runs a local NATS server for use with unit tests in this package.
// The server is started on a random available port
func TestMain(m *testing.M) {
	cfg, err := config.NewConfigurationFromMap(
		map[string]any{
			"servername": "unit_test_server",
		},
	)
	if err != nil {
		log.Fatalf("failed to parse server config: %v", err)
	}

	embeddedServer, err := messagebus.NewNatsEmbeddedServer(cfg, "")
	if err != nil {
		log.Fatalf("failed to start nats server: %v", err)
	}
	natsServer = embeddedServer

	// Setup JetStream connection.
	nc, err := natsServer.NewConnection()
	if err != nil {
		log.Fatalf("failed to get nats connection")
	}
	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatalf("failed to get jetstream connection")
	}

	for streamName, subjects := range streams {
		_, err = js.CreateStream(context.Background(), jetstream.StreamConfig{
			Name:        streamName,
			Compression: jetstream.S2Compression,
			Subjects:    subjects,
		})
		if err != nil {
			log.Fatalf("failed to create stream")
		}
	}

	// run the tests
	code := m.Run()

	for streamName := range streams {
		// don't check error (the nats server is limited to the test anyway)
		_ = js.DeleteStream(context.Background(), streamName)
	}

	natsServer.Close()
	os.Exit(code)
}

// TestNatsConnection ensures we are able to connect to the local NATS server
func TestNatsConnection(t *testing.T) {
	t.Parallel()
	nc := getNatsConnection(t)
	require.NotNil(t, nc)
	status := nc.Status()
	assert.Equal(t, nats.CONNECTED, status, "unexpected nats status %s", status.String())
}

// TestNatsConnectionWithConfigPath ensures we can connect to NATS using a custom config path
func TestNatsConnectionWithConfigPath(t *testing.T) {
	t.Parallel()

	cfg, err := config.NewConfigurationFromMap(
		map[string]any{
			"servername": "unit_test_server",
			"listenport": 4221,
		},
	)
	require.NoError(t, err)

	embeddedServer, err := messagebus.NewNatsEmbeddedServer(cfg, "")
	require.NoError(t, err)
	t.Cleanup(embeddedServer.Close)

	enc, err := embeddedServer.NewConnection()
	require.NoError(t, err)
	require.NotNil(t, enc)
	t.Cleanup(enc.Close)

	estatus := enc.Status()
	assert.Equal(t, nats.CONNECTED, estatus, "unexpected nats status %s", estatus.String())

	customNatsHost := "nats://localhost:4221"
	cfg, err = config.NewConfigurationFromMap(map[string]any{
		"custom_nats": map[string]any{
			"address": customNatsHost,
		},
	})
	require.NoError(t, err)

	// Use WithNATSConnectionConfigPath to specify the custom config path
	nc, err := messagebus.NewNatsConnection(cfg, messagebus.WithNATSConnectionConfigPath("custom_nats"))
	require.NoError(t, err)
	t.Cleanup(nc.Close)

	require.NotNil(t, nc)
	status := nc.Status()
	assert.Equal(t, nats.CONNECTED, status, "unexpected nats status %s", status.String())

	// Verify that the connection is using the custom config
	connURL := nc.ConnectedUrl()
	assert.Equal(t, customNatsHost, connURL, "unexpected connection URL")

	// Default config path should error
	nc, err = messagebus.NewNatsConnection(cfg)
	require.Error(t, err)
	require.Nil(t, nc)
}
