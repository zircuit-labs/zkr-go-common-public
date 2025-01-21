# Config

This package is meant to aid in streamlining runtime configuration.

It is assumed that services will be run in any one of several environments (eg. locally, staging, and production). Some configuration values are likely to remain constant across all or several environments, while others may change frequently. Some may contain sensitive information that should never be committed to a code repository.

## Hierarchy of Values

This package enables users to provide configuration stored in one of three places:

1. Directly in **code**. This should be for sane and defaults that are never expected to change - especially if this code it intended for use by anyone other than the authors

2. In a separate **TOML file**. This allows for clear separation between the code and the configuration values. Within this file exists a `[default]` section and then separate sections named for each environment in which the service needs to run (for example `[staging]` or `[production]`). Values in `[default]` take precedence over values set in (1) above, and values in the running environment (eg `[staging]`) take precedence over values set in `[default]`.

3. In **environment variables**. This should be used for secrets that cannot be committed to a code repository, or for values that may change more frequently than the code itself. These take precedence over values set in (1) and (2).

## Example Usage

Suppose a service contains two pieces of code that require some configuration. Let's call these `Alice` and `Bob`. Following best practices, each of these has a separate config struct.

### Config Struct

#### AliceConfig

Suppose `Alice` periodically does a GET from a particular endpoint using provided credentials. Config for `Alice` might look something like this:

```go
type AliceConfig struct {
    Host string
    Endpoint string
    Credentials credentials
    Period time.Duration `koanf:"frequency"`
}

type credentials struct {
    UserName string `koanf:"user_name"`
    Password string
}
```

Note the hinting using `koanf`. This is required when the name of the struct field doesn't match with the name of the field provided in the configuration itself.

#### BobConfig

Suppose `Bob` is some database wrapper, and needs config like this:

```go
type BobConfig struct {
    Epoch   time.Time
    Enabled bool
    Ports   []int
    Data    [][]string
    Servers map[string]struct {
        IP   string
        Role string
    }
}
```

### TOML File

By default, this file is expected to be in a file called `data/settings.toml`. This can be optionally changed, but is likely the easiest option. More on this later.

The file should contain all the configuration needed for the service, which in our case means config for both `Alice` and `Bob`. Supposing that the service is expected to be run locally and in production, the TOML file might look like this:

```toml
# default values are always used
[default]
[default.alice]
credentials.user_name = "alice"
# `credentials.password` is intentionally omitted,
# since it is sensitive information that will be
# provided via environment variable
frequency = "2h15m"

[default.bob]
epoch = 1979-05-27T07:32:00-08:00
enabled = true
ports = [ 8000, 8001, 8002 ]

[default.bob.servers]

[default.bob.servers.alpha]
ip = "10.0.0.1"
role = "frontend"

[default.bob.servers.beta]
ip = "10.0.0.2"
role = "backend"

# local values override default values when running in "local"
[local]
[local.alice]
host = "http://localhost:8080"
frequency = "30s"

[local.bob]
data = [ ["delta", "phi"], ["kappa"] ]
[local.bob.servers.alpha.ip] = "127.0.0.3"
[local.bob.servers.beta.ip] = "127.0.0.4"

# production values override default values when running in "production"
[production]
[production.alice]
host = "http://www.realwebsite.com"

[production.bob]
data = [ ["one", "two"], ["three"] ]
```

### Environment Variables

Since a running system could contain a great many environment variables, this package will limit itself to looking at those which begin with a known prefix. By default this is set to `CFG_` but can be overridden.

At least one environment variable is expected: the name of the environment in which the service is running. By default, the name of this variable is `CFG_ENV`.

In our example, `AliceConfig` expects a password for the credentials. In order to make this as easy as possible on users, simply set the environment variable to match the naming and path of the config field. In our case, it would be:

```sh
CFG_ALICE_CREDENTIALS_PASSWORD="super secret value"
```

### Code Usage

main.go

```go
// assume settings.toml is in data/ relative to main.go
//go:embed data/*
var f embed.FS

func main() {
    cfg, err := config.NewConfiguration(f)
    // check error
    alice, err := alice.NewAlice(cfg)
    // check error
    bob, err := bob.NewBob(cfg)
    // check error

    // do stuff with alice and bob
}
```

alice.go

```go
func NewAlice(cfg *config.Configuration) (*Alice, error) {
    aliceConfig := AliceConfig{
        Endpoint: "/some/endpoint",
    }
    if err := cfg.Unmarshal("alice", &aliceConfig); err != nil {
        return nil, err
    }

    // validate aliceConfig
    // use aliceConfig to return an Alice instance
}
```

bob.go

```go
func NewBob(cfg *config.Configuration) (*Bob, error) {
    var bobConfig BobConfig

    if err := cfg.Unmarshal("bob", &bobConfig); err != nil {
        return nil, err
    }

    // validate bobConfig
    // use bobConfig to return a Bob instance
}
```

### Result

#### Using `default` env

```sh
CFG_ENV unset
CFG_ALICE_CREDENTIALS_PASSWORD="super secret value"
```

Then

```go
aliceConfig == AliceConfig{
    Host: "",                             // never set
    Endpoint: "/some/endpoint",           // from struct creation
    Credentials: credentials {
        UserName: "alice",                // from toml/default
        Password: "super secret value",   // from env var
    },
    Period: time.Hour*2 + time.Minute*15, // from toml/default
}
```

and

```go
bobConfig == BobConfig{
    Epoch: t, // where t is time.Time of 1979-05-27T07:32:00-08:00 (RFC3339)
    Enabled: true,
    Ports: []int{8000, 8001, 8002},
    Data: nil,
    Servers: map[string]struct {
			IP   string
			Role string
		}{
			"alpha": {
				IP:   "10.0.0.1",
				Role: "frontend",
			},
			"beta": {
				IP:   "10.0.0.2",
				Role: "backend",
			},
		},
}
```

#### Using `local` env

```sh
CFG_ENV="local"
CFG_ALICE_CREDENTIALS_PASSWORD="super secret value"
```

Then

```go
aliceConfig == AliceConfig{
    Host: "http://localhost:8080",      // from toml/local
    Endpoint: "/some/endpoint",         // from struct creation
    Credentials: credentials {
        UserName: "alice",              // from toml/default
        Password: "super secret value", // from env var
    },
    Period: time.Second*30,             // from toml/local
}
```

and

```go
bobConfig == BobConfig{
    Epoch: t, // where t is time.Time of 1979-05-27T07:32:00-08:00 (RFC3339)
    Enabled: true,
    Ports: []int{8000, 8001, 8002},
    Data: [][]string{{"delta", "phi"}, {"kappa"}}, // from toml/local
    Servers: map[string]struct {
			IP   string
			Role string
		}{
			"alpha": {
				IP:   "127.0.0.3", // from toml/local
				Role: "frontend",  // from toml/default
			},
			"beta": {
				IP:   "127.0.0.3", // from toml/local
				Role: "backend",   // from toml/default
			},
		},
}
```

#### Using `production` env

```sh
CFG_ENV="production"
CFG_ALICE_CREDENTIALS_PASSWORD="super secret value"
```

Then

```go
aliceConfig == AliceConfig{
    Host: "http://www.realwebsite.com",   // from toml/production
    Endpoint: "/some/endpoint",           // from struct creation
    Credentials: credentials {
        UserName: "alice",                // from toml/default
        Password: "super secret value",   // from env var
    },
    Period: time.Hour*2 + time.Minute*15, // from toml/default
}
```

and

```go
bobConfig == BobConfig{
    Epoch: t, // where t is time.Time of 1979-05-27T07:32:00-08:00 (RFC3339)
    Enabled: true,
    Ports: []int{8000, 8001, 8002},
    Data: [][]string{{"one", "two"}, {"three"}}, // from toml/production
    Servers: map[string]struct {
			IP   string
			Role string
		}{
			"alpha": {
				IP:   "10.0.0.1",
				Role: "frontend",
			},
			"beta": {
				IP:   "10.0.0.2",
				Role: "backend",
			},
		},
}
```

## Alternative Config Method

In the event a config struct is needed without using a file or env var (as in unit testing for example), use `NewConfigurationFromMap(cfg map[string]any)` to create one using a flat map of string values.

For example:

```go
cfg, err := config.NewConfigurationFromMap(
	map[string]any{
		"nats.address":  "nats://localhost:4222",
		"subject":       "foo",
		"durablequeue": "bar",
	},
)
```

### Config item names

Do not include `_` in your config item names, as they will conflict with overriding them through environment variables. For example you will not be able to override the following using environment variables.

```toml
[default.block_serializer]
l2geth_http = "http://localhost:8545"
serializer_path = "./cmd/l2listener/data/block-serializer"
```
