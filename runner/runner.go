package runner

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/DataDog/dd-trace-go/v2/profiler"

	"github.com/zircuit-labs/zkr-go-common/calm"
	"github.com/zircuit-labs/zkr-go-common/config"
	"github.com/zircuit-labs/zkr-go-common/log"
	"github.com/zircuit-labs/zkr-go-common/log/identity"
	"github.com/zircuit-labs/zkr-go-common/messagebus"
	"github.com/zircuit-labs/zkr-go-common/singleton"
	"github.com/zircuit-labs/zkr-go-common/task"
	"github.com/zircuit-labs/zkr-go-common/task/ossignal"
	"github.com/zircuit-labs/zkr-go-common/version"
	"github.com/zircuit-labs/zkr-go-common/xerrors/errclass"
	"github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
)

const (
	exitError = 1
	exitPanic = 2 // go standard exit code on panic
	cfgPath   = "runner"
)

type runnerConfig struct {
	LogLevel string
}

type options struct {
	singleton       bool
	useProvidedName bool
}

type Option func(options *options)

func AsSingleton() Option {
	return func(options *options) {
		options.singleton = true
	}
}

func UseProvidedName() Option {
	return func(options *options) {
		options.useProvidedName = true
	}
}

// Runner limits task manager interface.
type Runner interface {
	Run(tasks ...task.Task)
	RunTerminable(tasks ...task.Task)
	Cleanup(f func())
	Context() context.Context
}

// Runnable is a func that takes arguments provided by Run.
type Runnable func(cfg *config.Configuration, tm Runner, logger *slog.Logger) error

// Run abstracts away common boilerplate from `main()` for standardized services.
func Run(serviceName string, f fs.FS, run Runnable, opts ...Option) {
	options := options{}
	for _, opt := range opts {
		opt(&options)
	}

	// Get the service name from the environment.
	name, ok := os.LookupEnv("DD_SERVICE")
	// If does not exist or option set, use provide name instead.
	if !ok || options.useProvidedName {
		name = serviceName
	}
	identity.SetServiceName(name)
	n, id := identity.WhoAmI()

	// create logger
	logger, err := log.NewLogger(
		log.WithServiceName(n),
		log.WithInstanceID(id),
		log.WithVersion(&version.Info),
	)
	if err != nil {
		fmt.Printf("failed to create logger: %s\n", err)
		os.Exit(exitError) //revive:disable:deep-exit // intentional
	}

	// execute the core run logic protected from direct panics.
	// NOTE: goroutines spawned by `run` must be themselves protected.
	err = calm.Unpanic(func() error {
		return protectedRun(f, run, logger, options)
	})

	switch errclass.GetClass(err) {
	case errclass.Nil:
		logger.Info("service exited normally")
	case errclass.Panic:
		logger.Error("service failed with panic", log.ErrAttr(err))
		os.Exit(exitPanic) //revive:disable:deep-exit // intentional
	default:
		logger.Error("service failed with error", log.ErrAttr(err))
		os.Exit(exitError) //revive:disable:deep-exit // intentional
	}
}

func protectedRun(f fs.FS, run Runnable, logger *slog.Logger, opts options) error {
	name, id := identity.WhoAmI()
	// start the DataDog profiler and tracer if the env var is set
	if _, ok := os.LookupEnv("DD_APM_ENABLED"); ok {
		err := profiler.Start(
			profiler.WithService(name),
			profiler.WithVersion(version.Info.Version),
			profiler.WithTags(
				fmt.Sprintf("instance:%s", id),
				fmt.Sprintf("git_commit:%s", version.Info.GitCommit),
			),
			profiler.WithProfileTypes(
				profiler.CPUProfile,
				profiler.HeapProfile,
				// The profiles below increase overhead, and could be disabled if needed.
				profiler.BlockProfile,
				profiler.MutexProfile,
				profiler.GoroutineProfile,
			),
		)
		if err != nil {
			logger.Error("failed to start datadog profiler", log.ErrAttr(err))
			return stacktrace.Wrap(err)
		}
		defer profiler.Stop()

		err = tracer.Start(
			tracer.WithService(name),
			tracer.WithServiceVersion(version.Info.Version),
		)
		if err != nil {
			logger.Error("failed to start datadog tracer", log.ErrAttr(err))
			return stacktrace.Wrap(err)
		}
		defer tracer.Stop()
	}

	// get config information
	cfg, err := config.NewConfiguration(f)
	if err != nil {
		return stacktrace.Wrap(err)
	}

	serverConfig := runnerConfig{}
	if err := cfg.Unmarshal(cfgPath, &serverConfig); err != nil {
		return stacktrace.Wrap(err)
	}

	if err := log.SetLogLevel(serverConfig.LogLevel); err != nil {
		logger.Error("failed to set log level", log.ErrAttr(err))
	}

	// create task manager
	tm := task.NewManager(task.WithLogger(logger))

	// start os signal task
	tm.Run(ossignal.NewTask(ossignal.WithLogger(logger)))

	// set up singleton state
	if opts.singleton {
		nc, err := messagebus.NewNatsConnection(cfg)
		if err != nil {
			return stacktrace.Wrap(err)
		}
		defer nc.Close()

		lockFactory, err := singleton.NewLockFactory[any](
			nc,
			id,
			singleton.WithLogger(logger),
		)
		if err != nil {
			return stacktrace.Wrap(err)
		}

		// Acquire lock before proceeding.
		// Use the same context as the task manager so that
		// the ossignal task can gracefully cancel this.
		lock, err := lockFactory.CreateLock(tm.Context(), name, nil)
		if err != nil {
			if errors.Is(err, tm.Context().Err()) {
				return nil
			}
			return stacktrace.Wrap(err)
		}
		// "Run" the lock to check for lock loss
		tm.Run(lock)
	}

	// execute the Runnable
	err = run(cfg, tm, logger)
	// if the Runnable fails, stop any running tasks and terminate now
	if err != nil {
		_ = tm.Stop() // ignore any error from Stop()
		return err
	}

	// otherwise wait for running tasks to complete
	return tm.Wait()
}
