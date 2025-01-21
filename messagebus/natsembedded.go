package messagebus

import (
	"context"
	"errors"
	"net/url"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"

	"github.com/zircuit-labs/zkr-go-common/config"
	"github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
)

var (
	ErrNotRunning = errors.New("embedded nats server is not running")
	ErrNotReady   = errors.New("embedded nats server is not ready for connections")
)

type natsEmbeddedServerConfig struct {
	ServerName            string `koanf:"servername"`            // Optionally provide a name for the embedded server
	ListenPort            int    `koanf:"listenport"`            // 0 = don't listen
	JetStreamDisabled     bool   `koanf:"jetstreamdisabled"`     // Disable JetStream functionality
	JetStreamDomain       string `koanf:"jetstreamdomain"`       // The remote will need this in order to use data from here
	LeafNodeURL           string `koanf:"leafnodeurl"`           // If connecting as a leaf-node. eg: "nats-leaf://nats"
	StoreDir              string `koanf:"storedir"`              // Directory in which to store JetStream data
	RemoteCredentialsPath string `koanf:"remotecredentialspath"` // Use this for .creds files
	EnableLogging         bool   `koanf:"enablelogging"`         // Enable logging
}

// NatsEmbeddedServer is an embedded NATS server with a connection that can be shared for local use.
type NatsEmbeddedServer struct {
	ns        *server.Server
	inProcess bool
}

// NewNatsEmbeddedServer creates a NatsEmbeddedServer parsing a limited config set for server options.
// If more options are required, it is probably better to use the nats.io code directly.
func NewNatsEmbeddedServer(cfg *config.Configuration, cfgPath string) (*NatsEmbeddedServer, error) {
	natsConfig := natsEmbeddedServerConfig{}

	if err := cfg.Unmarshal(cfgPath, &natsConfig); err != nil {
		return nil, stacktrace.Wrap(err)
	}

	serverOpts := &server.Options{
		ServerName:      natsConfig.ServerName,
		DontListen:      (natsConfig.ListenPort == 0),
		Port:            natsConfig.ListenPort,
		JetStream:       !natsConfig.JetStreamDisabled,
		JetStreamDomain: natsConfig.JetStreamDomain,
		StoreDir:        natsConfig.StoreDir,

		// Logging options
		Debug: true,
		Trace: true,
	}

	if natsConfig.LeafNodeURL != "" {
		leafURL, err := url.Parse(natsConfig.LeafNodeURL)
		if err != nil {
			return nil, stacktrace.Wrap(err)
		}

		serverOpts.LeafNode = server.LeafNodeOpts{
			Remotes: []*server.RemoteLeafOpts{
				{
					URLs:        []*url.URL{leafURL},
					Credentials: natsConfig.RemoteCredentialsPath,
				},
			},
		}
	}

	ns, err := server.NewServer(serverOpts)
	if err != nil {
		return nil, stacktrace.Wrap(err)
	}

	// Enable logging if requested
	if natsConfig.EnableLogging {
		ns.ConfigureLogger()
	}

	// Start the server, and ensure it is ready
	go ns.Start()
	if !ns.ReadyForConnections(time.Second * 5) {
		ns.Shutdown()
		return nil, stacktrace.Wrap(ErrNotReady)
	}

	return &NatsEmbeddedServer{
		ns:        ns,
		inProcess: serverOpts.DontListen,
	}, nil
}

// Name returns the name of this task for the purposes of logging.
func (s NatsEmbeddedServer) Name() string {
	return "embedded_nats_server_" + s.ns.Name()
}

// Run blocks until the context is done, or the server stops running.
func (s *NatsEmbeddedServer) Run(ctx context.Context) error {
	defer s.Close()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if !s.ns.Running() {
				return ErrNotRunning
			}
		}
	}
}

// Connection returns a new connection to the embedded server.
// Callers are responsible for closing this connection when finished with it.
func (s NatsEmbeddedServer) NewConnection() (*nats.Conn, error) {
	// create a connection
	clientOpts := []nats.Option{}
	if s.inProcess {
		clientOpts = append(clientOpts, nats.InProcessServer(s.ns))
	}

	nc, err := nats.Connect(s.ns.ClientURL(), clientOpts...)
	if err != nil {
		return nil, stacktrace.Wrap(err)
	}

	return nc, nil
}

// Close will shut down the embedded nats server.
func (s *NatsEmbeddedServer) Close() {
	s.ns.Shutdown()
	s.ns.WaitForShutdown()
}
