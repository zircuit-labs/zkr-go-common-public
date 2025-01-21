package log

import (
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/neilotoole/slogt"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
	slogcommon "github.com/samber/slog-common"
	slogzerolog "github.com/samber/slog-zerolog/v2"
	"github.com/zircuit-labs/zkr-go-common/version"
	"github.com/zircuit-labs/zkr-go-common/xerrors"
	"github.com/zircuit-labs/zkr-go-common/xerrors/errclass"
	"github.com/zircuit-labs/zkr-go-common/xerrors/errcontext"
	"github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
)

const (
	ErrorKey      = "error"
	SourceKey     = "source"
	StackTraceKey = "stacktrace"
	ErrClassKey   = "class"
)

type Identity struct {
	ServiceName string
	InstanceID  string
}

func (i Identity) String() string {
	return fmt.Sprintf("%s-%s", i.ServiceName, i.InstanceID)
}

var (
	logLevel = &slog.LevelVar{}
	identity = Identity{
		ServiceName: "unknown",
		InstanceID:  xid.New().String(),
	}
)

func WhoAmI() Identity {
	return identity
}

func SetLogLevel(level string) error {
	if level != "" {
		return logLevel.UnmarshalText([]byte(level))
	}
	return nil
}

// ErrAttr is a helper for logging error values.
func ErrAttr(err error) slog.Attr {
	return slog.Any(ErrorKey, err)
}

// NewTestLogger creates a new logger for testing.
// NOTE: Since this logger uses the testing t.Log method,
// it will only log when the test fails. Additionally,
// it will cause a panic if the logger is called after the
// test has completed. This can be helpful for finding race conditions.
func NewTestLogger(t *testing.T) *slog.Logger {
	t.Helper()
	log := slogt.New(t, slogt.JSON()).With(slog.String("test", t.Name()))
	return log
}

// NewLogger creates a new slog logger backed by zerolog with some standard defaults.
func NewLogger(serviceName string) *slog.Logger {
	// ms granularity should be sufficient
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	identity.ServiceName = serviceName

	zlogger := zerolog.
		New(os.Stdout).With().                      // log to stdout not stderr
		Timestamp().                                // include timestamp
		Str("service", identity.ServiceName).       // include the service name
		Str("instance", identity.InstanceID).       // include unique id for instance
		Str("git_commit", version.Info.GitCommit).  // include hash of last git commit
		Time("git_commit_time", version.Info.Date). // include timestamp of last git commit
		Str("git_branch", version.Info.GitBranch).  // include name of the branch when built
		Str("version", version.Info.Version).       // include version information
		Str("version_meta", version.Info.Meta).     // include version metadata
		Logger()

	logger := slog.New(slogzerolog.Option{
		Converter: CustomSlogConverter,
		Level:     logLevel,
		Logger:    &zlogger,
	}.NewZerologHandler())

	return logger
}

// CustomSlogConverter is a copy of slogcommon.DefaultConverter, except that the replaceError function has been swapped with our own.
func CustomSlogConverter(addSource bool, replaceAttr func(groups []string, a slog.Attr) slog.Attr, loggerAttr []slog.Attr, groups []string, record *slog.Record) map[string]any {
	// aggregate all attributes
	attrs := slogcommon.AppendRecordAttrsToAttrs(loggerAttr, groups, record)

	// replace error(s)
	attrs = replaceError(attrs)
	if addSource {
		attrs = append(attrs, slogcommon.Source(SourceKey, record))
	}
	attrs = slogcommon.ReplaceAttrs(replaceAttr, []string{}, attrs...)

	// handler formatter
	output := slogcommon.AttrsToMap(attrs...)

	return output
}

/*
replaceError looks for an "error" attribute, and if found, replaces it with the following:
if the error is not joined:

	{
		"error": err.Error()
		"error_context":
			"error": err.Error(),
			"stacktrace": <the error stacktrace if it exists>,
			"class": <the error class if it exists>,
			"key": <value>, // for each key/value in the error context
		},
	}

if the error is joined:

	{
		"error": [err.Error(), err.Error(), ...]
		"error_context": [
			"error_0": {
				"error": err.Error(),
				"stacktrace": <the error stacktrace if it exists>,
				"class": <the error class if it exists>,
				"key": <value>, // for each key/value in the error context
			},
			"error_1": {
				"error": err.Error(),
				"stacktrace": <the error stacktrace if it exists>,
				"class": <the error class if it exists>,
				"key": <value>, // for each key/value in the error context
			},
			...
		]
	}

	See the tests for detailed examples.
*/
func replaceError(attrs []slog.Attr) []slog.Attr {
	var groupedAttrs [][]any
	replaceAttr := func(groups []string, a slog.Attr) slog.Attr {
		if len(groups) > 1 {
			return a
		}

		if a.Key != ErrorKey {
			return a
		}

		err, ok := a.Value.Any().(error)
		if !ok || err == nil {
			return a
		}

		joinedErrs := xerrors.Unjoin(err)
		groupedAttrs = make([][]any, len(joinedErrs))
		errorStrings := make([]string, 0, len(joinedErrs))
		for i, joinedErr := range joinedErrs {

			groupedAttrs[i] = append(groupedAttrs[i], slog.String(ErrorKey, joinedErr.Error()))
			errorStrings = append(errorStrings, joinedErr.Error())

			// add a stacktrace if found
			if trace := stacktrace.StackTraceMarshaler(joinedErr); trace != nil {
				groupedAttrs[i] = append(groupedAttrs[i], slog.Any(StackTraceKey, trace))
			}

			// add an error class if found
			if class := errclass.GetClass(joinedErr); class != errclass.Unknown {
				groupedAttrs[i] = append(groupedAttrs[i], slog.String(ErrClassKey, class.String()))
			}

			// add any additional context
			for _, attr := range errcontext.Get(joinedErr) {
				groupedAttrs[i] = append(groupedAttrs[i], attr)
			}
		}

		if len(joinedErrs) == 1 {
			return slog.String(ErrorKey, err.Error())
		}

		return slog.Any(a.Key, errorStrings)
	}
	results := append(attrs, slogcommon.ReplaceAttrs(replaceAttr, []string{}, attrs...)...)

	if len(groupedAttrs) == 0 {
		return results
	}

	if len(groupedAttrs) == 1 {
		if len(groupedAttrs[0]) > 1 {
			results = append(results, slog.Group("error_context", groupedAttrs[0]...))
		}
		return results
	}

	groups := make([]slog.Attr, len(groupedAttrs))
	for i, group := range groupedAttrs {
		groupName := fmt.Sprintf("error_%d", i)
		groups[i] = slog.Group(groupName, group...)
	}

	if len(groupedAttrs) > 0 {
		results = append(results, slog.Any("error_context", groups))
	}

	return results
}
