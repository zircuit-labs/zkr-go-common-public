// Package echotask wraps an http server using the echo framework as a task
package echotask

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	ddtrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/labstack/echo.v4"

	"github.com/zircuit-labs/zkr-go-common/calm/errgroup"
	"github.com/zircuit-labs/zkr-go-common/config"
	"github.com/zircuit-labs/zkr-go-common/http/echotask/cache"
	"github.com/zircuit-labs/zkr-go-common/http/echotask/healthcheck"
	"github.com/zircuit-labs/zkr-go-common/http/port"
	"github.com/zircuit-labs/zkr-go-common/log"
	"github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
)

// RouteRegistrant is able to register URI routes only.
type RouteRegistrant interface {
	CONNECT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	HEAD(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	OPTIONS(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	TRACE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
}

const (
	healthCheckRoute = "/healthcheck"
	metricsRoute     = "/metrics"
)

// RouteRegistration registers routes.
type RouteRegistration interface {
	RegisterRoutes(RouteRegistrant) error
}

type echoServerConfig struct {
	Port               int
	TLS                bool
	DisableCompression bool `koanf:"nogzip"`
	Prometheus         string
}

type options struct {
	name        string
	routes      []RouteRegistration
	middlewares []echo.MiddlewareFunc
	cleanup     func()
	healthcheck healthChecker
	logger      *slog.Logger
}

type healthChecker interface {
	HealthCheck(ctx context.Context) error
}

// Option is an option func for NewServer.
type Option func(options *options)

// WithLogger sets the logger to be used.
func WithLogger(logger *slog.Logger) Option {
	return func(options *options) {
		options.logger = logger
	}
}

// WithName sets the name of the task.
func WithName(name string) Option {
	return func(options *options) {
		options.name = name
	}
}

// WithRoutes adds routes to be served.
func WithRoutes(routes RouteRegistration) Option {
	return func(options *options) {
		options.routes = append(options.routes, routes)
	}
}

// WithHealthCheck adds a healthcheck route to be served.
func WithHealthCheck(checker healthChecker) Option {
	return func(options *options) {
		options.healthcheck = checker
	}
}

// WithCleanup sets a cleanup func to be called after server shutdown.
func WithCleanup(f func()) Option {
	return func(options *options) {
		options.cleanup = f
	}
}

// WithMemoryCache adds a memory-backed caching middleware with the specified duration to the server options.
func WithMemoryCache(maxItems int, ttl time.Duration) Option {
	return func(opts *options) {
		memoryCache := cache.NewMemory(maxItems, ttl)
		opts.middlewares = append(opts.middlewares, cache.ResponseCacheMiddleware(memoryCache))
	}
}

// Server is an HTTP(S) server using the echo framework.
type Server struct {
	e       *echo.Echo
	name    string
	port    int
	cleanup func()
	logger  *slog.Logger
}

// NewServer creates an HTTP(S) server using the echo framework that implements the Task interface.
func NewServer(cfg *config.Configuration, cfgPath string, opts ...Option) (*Server, error) {
	// Parse and validate server config
	serverConfig := echoServerConfig{}
	if err := cfg.Unmarshal(cfgPath, &serverConfig); err != nil {
		return nil, err
	}

	// Set up default options
	options := options{
		name:   "echo server",
		logger: log.NewNilLogger(),
	}

	// Apply provided options
	for _, opt := range opts {
		opt(&options)
	}

	// Determine appropriate port
	var p int
	var err error
	if serverConfig.Port != 0 {
		p = serverConfig.Port
	} else {
		p, err = port.AvailablePort()
		if err != nil {
			return nil, err
		}
	}

	if serverConfig.TLS {
		// TODO Support TLS
		panic("tls support not yet implemented")
	}

	// create the echo server
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	// include DataDog trace middleware if the env var is set
	if _, ok := os.LookupEnv("DD_APM_ENABLED"); ok {
		e.Use(ddtrace.Middleware(ddtrace.WithServiceName(log.WhoAmI().ServiceName)))
	}
	e.Use(middleware.CORS())
	e.Use(Recover(options.logger))
	e.Pre(middleware.RemoveTrailingSlash())

	// enable gzip compression
	if !serverConfig.DisableCompression {
		e.Use(middleware.Gzip())
	}

	// Apply middlewares
	for _, m := range options.middlewares {
		e.Use(m)
	}

	if serverConfig.Prometheus != "" {
		e.Use(echoprometheus.NewMiddlewareWithConfig(echoprometheus.MiddlewareConfig{
			Subsystem:                 serverConfig.Prometheus,
			DoNotUseRequestPathFor404: true,
		}))
		e.GET(metricsRoute, echoprometheus.NewHandler()) // register route for getting gathered metrics
	}

	// register routes
	for _, r := range options.routes {
		if err := r.RegisterRoutes(e); err != nil {
			return nil, err
		}
	}

	if options.healthcheck != nil {
		e.GET(healthCheckRoute, healthcheck.New(options.healthcheck).Handle)
	}

	return &Server{
		e:       e,
		port:    p,
		name:    options.name,
		cleanup: options.cleanup,
		logger:  options.logger,
	}, nil
}

// Run implements the Task interface.
func (t *Server) Run(ctx context.Context) error {
	if t.cleanup != nil {
		defer t.cleanup()
	}

	g := errgroup.New()

	// Start the server
	// This is a blocking call, so run it in a goroutine
	g.Go(func() error {
		err := t.e.Start(fmt.Sprintf(":%d", t.port))
		// ErrServerClosed is returned on graceful shutdown
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return stacktrace.Wrap(err)
	})

	// Wait for the Run context to complete
	// This is also blocking
	g.Go(func() error {
		<-ctx.Done()
		return t.e.Shutdown(context.Background())
	})

	return g.Wait()
}

// Name returns the name of this task.
func (t *Server) Name() string {
	return fmt.Sprintf("%s on :%d", t.name, t.port)
}
