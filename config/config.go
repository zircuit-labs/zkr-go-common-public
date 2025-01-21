// Package config provides standardized runtime configuration.
package config

import (
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	koanffs "github.com/knadh/koanf/providers/fs"
	"github.com/zircuit-labs/zkr-go-common/xerrors/errclass"
	"github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
)

const (
	defaultEnv           = "default"
	defaultEnvPrefix     = "CFG_"
	defaultEnvSeparator  = "_"
	defaultConfSeparator = "."
	defaultSettingsPath  = "data/settings.toml"

	envVarName = "ENV"
)

type options struct {
	defaultEnv   string
	envPrefix    string
	filepath     string
	separator    string
	envSeparator string
}

// Option is an option func for NewConfiguration.
type Option func(options *options) error

// WithDefaultEnv sets the name of the default environment.
func WithDefaultEnv(env string) Option {
	return func(options *options) error {
		options.defaultEnv = env
		return nil
	}
}

// WithEnvPrefix sets the environment variable prefix.
func WithEnvPrefix(prefix string) Option {
	return func(options *options) error {
		options.envPrefix = prefix
		return nil
	}
}

// WithFilePath sets the path to the TOML file.
func WithFilePath(path string) Option {
	return func(options *options) error {
		options.filepath = path
		return nil
	}
}

// WithSeparator sets the path separator to be used internally.
func WithSeparator(separator string) Option {
	return func(options *options) error {
		options.separator = separator
		return nil
	}
}

// WithEnvSeparator sets the environment variable separator.
func WithEnvSeparator(separator string) Option {
	return func(options *options) error {
		options.envSeparator = separator
		return nil
	}
}

// Configuration is a wrapper for koanf to hide complexity.
type Configuration struct {
	k   *koanf.Koanf
	env string
}

// NewConfigurationFromMap allows for a direct flat map to be used to create configuration.
func NewConfigurationFromMap(cfg map[string]any) (*Configuration, error) {
	k := koanf.New(defaultConfSeparator)
	if err := k.Load(
		confmap.Provider(cfg, defaultConfSeparator),
		nil,
	); err != nil {
		return nil, errclass.WrapAs(stacktrace.Wrap(err), errclass.Persistent)
	}
	return &Configuration{k: k, env: defaultEnv}, nil
}

// NewConfiguration parses config from the given file system and environment variables.
func NewConfiguration(f fs.FS, opts ...Option) (*Configuration, error) {
	// Set up default parsing options
	options := options{
		defaultEnv:   defaultEnv,
		envPrefix:    defaultEnvPrefix,
		separator:    defaultConfSeparator,
		envSeparator: defaultEnvSeparator,
		filepath:     defaultSettingsPath,
	}

	// Apply provided options
	for _, opt := range opts {
		err := opt(&options)
		if err != nil {
			return nil, err
		}
	}

	// If no file system is provided, only use environment variables
	if f == nil {
		return envOnlyConfig(options)
	}

	// Load the full TOML config file, which should include the default env and all env overrides
	top := koanf.New(defaultConfSeparator)
	if err := top.Load(koanffs.Provider(f, options.filepath), toml.Parser()); err != nil {
		return nil, errclass.WrapAs(stacktrace.Wrap(err), errclass.Persistent)
	}

	// The default env settings must exist
	if !top.Exists(options.defaultEnv) {
		return nil, errclass.WrapAs(stacktrace.Wrap(
			fmt.Errorf("default environment '%s' not found", options.defaultEnv),
		), errclass.Persistent)
	}

	// The final settings are the merged result of the default env and others, so extract that first
	merged := koanf.New(defaultConfSeparator)
	defaultSettings, ok := top.Get(options.defaultEnv).(map[string]any)
	if !ok {
		return nil, errclass.WrapAs(stacktrace.Wrap(
			fmt.Errorf("failed to parse default env settings"),
		), errclass.Persistent)
	}
	if err := merged.Load(confmap.Provider(defaultSettings, options.separator), nil); err != nil {
		return nil, errclass.WrapAs(stacktrace.Wrap(err), errclass.Persistent)
	}

	// Determine if an override env was set
	envKey := fmt.Sprintf("%s%s", options.envPrefix, envVarName)
	environment := os.Getenv(envKey)

	if environment != "" {
		// Load and merge the override environment settings
		if !top.Exists(environment) {
			return nil, errclass.WrapAs(stacktrace.Wrap(
				fmt.Errorf("environment settings for '%s' not found", environment),
			), errclass.Persistent)
		}
		envSettings, ok := top.Get(environment).(map[string]any)
		if !ok {
			return nil, errclass.WrapAs(stacktrace.Wrap(
				fmt.Errorf("failed to parse env settings for '%s'", environment),
			), errclass.Persistent)
		}
		if err := merged.Load(confmap.Provider(envSettings, options.separator), nil); err != nil {
			return nil, errclass.WrapAs(stacktrace.Wrap(err), errclass.Persistent)
		}
	} else {
		// If it wasn't set, set it now to the default
		environment = options.defaultEnv
	}

	// Load and merge override settings from environment variables
	if err := merged.Load(
		env.Provider(options.envPrefix, options.separator, envToConfig(options)),
		nil,
	); err != nil {
		return nil, errclass.WrapAs(stacktrace.Wrap(err), errclass.Persistent)
	}

	return &Configuration{k: merged, env: environment}, nil
}

func envOnlyConfig(options options) (*Configuration, error) {
	// Determine environment if set
	envKey := fmt.Sprintf("%s%s", options.envPrefix, envVarName)
	environment := os.Getenv(envKey)
	if environment == "" {
		environment = options.defaultEnv
	}

	// Load settings from environment variables
	k := koanf.New(defaultConfSeparator)
	if err := k.Load(
		env.Provider(options.envPrefix, options.separator, envToConfig(options)),
		nil,
	); err != nil {
		return nil, errclass.WrapAs(stacktrace.Wrap(err), errclass.Persistent)
	}
	return &Configuration{k: k, env: environment}, nil
}

// Unmarshal sets values in struct `a` from the config rooted at `path`.
func (c Configuration) Unmarshal(path string, a any) error {
	return c.k.Unmarshal(path, a)
}

// Environment returns the value of the set environment
func (c Configuration) Environment() string {
	return c.env
}

// envToConfig is a factory to generate anonymous functions for transforming config keys.
// For example, env var `PREFIX_NESTED_VALUE_A` might be converted to `nested.value.a`
func envToConfig(options options) func(s string) string {
	return func(s string) string {
		return strings.ReplaceAll(
			strings.ToLower(
				strings.TrimPrefix(s, options.envPrefix),
			),
			options.envSeparator,
			options.separator,
		)
	}
}
