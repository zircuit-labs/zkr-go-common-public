// WARNING: Do not use `t.Parallel()` for tests in this package
// since the tests rely on setting and unsetting of environment variables

package config_test

import (
	"embed"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zircuit-labs/zkr-go-common/config"
)

const (
	testPrefix = "ABCD_"
	testEnv    = "ABCD_ENV"
)

//go:embed test/*
var f embed.FS

type testConfig struct {
	A string
	B string
	C nestedConfig
}

type nestedConfig struct {
	W string
	X string
	Y string
	Z string
}

type mismatchedConfig struct {
	A int
}

type typeTestConfig struct {
	Title  string
	Period time.Duration
	Owner  struct {
		Name string
		DOB  time.Time
	}
	Database struct {
		Enabled     bool
		Ports       []int
		Data        [][]string
		TempTargets targets `koanf:"temp_targets"`
		String      string
	}
	Servers map[string]struct {
		IP   string
		Role string
	}
}

type targets struct {
	CPU  float32
	Case float32
}

// TestNoEnv directly loads the test/example.toml with no set env vars
// This should result in using the `default` section
func TestNoEnv(t *testing.T) { //nolint:paralleltest // uses env vars
	cfg, err := config.NewConfiguration(
		f,
		config.WithFilePath("test/example.toml"),
		config.WithEnvPrefix(testPrefix),
	)
	require.NoError(t, err)

	testStruct := testConfig{
		C: nestedConfig{
			X: "should be overwritten",
			Y: "yamaha",
		},
	}

	err = cfg.Unmarshal("", &testStruct)
	require.NoError(t, err)

	expected := testConfig{
		A: "alpha",
		B: "beta",
		C: nestedConfig{
			X: "x-ray",
			Y: "yamaha",
			Z: "zulu",
		},
	}

	// resulting testStruct should contain settings from default
	// as well as pre-existing values
	assert.Equal(t, expected, testStruct)
	assert.Equal(t, "default", cfg.Environment())
}

// TestHierarchy ensures that values are overridden correctly
// env vars > local section > default section > original struct
func TestHierarchy(t *testing.T) {
	t.Setenv(testEnv, "local")
	t.Setenv(fmt.Sprintf("%sB", testPrefix), "bravo")
	t.Setenv(fmt.Sprintf("%sC_W", testPrefix), "watermelon")

	cfg, err := config.NewConfiguration(
		f,
		config.WithFilePath("test/example.toml"),
		config.WithEnvPrefix(testPrefix),
	)
	require.NoError(t, err)

	testStruct := testConfig{
		C: nestedConfig{
			X: "should be overwritten",
			Y: "yamaha",
		},
	}

	err = cfg.Unmarshal("", &testStruct)
	require.NoError(t, err)

	expected := testConfig{
		A: "aardvark", // local > default
		B: "bravo",    // env > local > default
		C: nestedConfig{
			W: "watermelon", // env
			X: "x-ray",      // default > original struct
			Y: "yamaha",     // original struct
			Z: "zebra",      // local > default
		},
	}
	assert.Equal(t, expected, testStruct)
	assert.Equal(t, "local", cfg.Environment())

	// Also validate unmarshal of a sub-section
	var actualNested nestedConfig
	err = cfg.Unmarshal("c", &actualNested)
	require.NoError(t, err)
	expectedNested := nestedConfig{
		W: "watermelon",
		X: "x-ray",
		Y: "", // not set in original actualNested
		Z: "zebra",
	}
	assert.Equal(t, expectedNested, actualNested)
}

// TestMissingDefaultSection ensures an error is returned when
// the expected default section does not exist
func TestMissingDefaultSection(t *testing.T) {
	t.Parallel()
	_, err := config.NewConfiguration(
		f,
		config.WithFilePath("test/example.toml"),
		config.WithEnvPrefix(testPrefix),
		config.WithDefaultEnv("not default"), // use a different value
	)
	require.Error(t, err)
}

// TestMissingOverrideSection ensures that if there exists a named environment
// override in the env vars, that section must exist in the toml file
func TestMissingOverrideSection(t *testing.T) {
	t.Setenv(testEnv, "not-a-header")

	_, err := config.NewConfiguration(
		f,
		config.WithFilePath("test/example.toml"),
		config.WithEnvPrefix(testPrefix),
	)
	require.Error(t, err)
}

func TestMissingFile(t *testing.T) { //nolint:paralleltest // uses env vars
	_, err := config.NewConfiguration(
		f,
		config.WithFilePath("test/not-found.toml"),
		config.WithEnvPrefix(testPrefix),
	)
	require.Error(t, err)
}

func TestBadFileFormat(t *testing.T) { //nolint:paralleltest // uses env vars
	_, err := config.NewConfiguration(
		f,
		config.WithFilePath("test/not-toml.json"),
		config.WithEnvPrefix(testPrefix),
	)
	require.Error(t, err)
}

func TestMismatchedStruct(t *testing.T) {
	t.Parallel()
	cfg, err := config.NewConfiguration(
		f,
		config.WithFilePath("test/example.toml"),
		config.WithEnvPrefix(testPrefix),
	)
	require.NoError(t, err)

	var testStruct mismatchedConfig

	// This should fail since mismatchedConfig expects key A to be an int
	err = cfg.Unmarshal("", &testStruct)
	assert.Error(t, err)
}

// TestTypeConversions validates type conversion of supported TOML types
// as well as custom type `Secret` (which is just an alias for String)
func TestTypeConversions(t *testing.T) {
	t.Parallel()
	cfg, err := config.NewConfiguration(
		f,
		config.WithFilePath("test/types.toml"),
		config.WithEnvPrefix(testPrefix),
	)
	require.NoError(t, err)

	var testStruct typeTestConfig

	err = cfg.Unmarshal("", &testStruct)
	require.NoError(t, err)

	d, err := time.Parse(time.RFC3339, "1979-05-27T07:32:00-08:00")
	require.NoError(t, err)

	expected := typeTestConfig{
		Title:  "TOML Example",
		Period: time.Hour*2 + time.Minute*15,
		Owner: struct {
			Name string
			DOB  time.Time
		}{
			Name: "Tom Preston-Werner",
			DOB:  d,
		},
		Database: struct {
			Enabled bool
			Ports   []int
			Data    [][]string
			// The repetition of the type hinting here is required
			// only because `Database` is an anonymous struct
			TempTargets targets `koanf:"temp_targets"`
			String      string
		}{
			Enabled: true,
			Ports:   []int{8000, 8001, 8002},
			Data:    [][]string{{"delta", "phi"}, {"kappa"}},
			TempTargets: targets{
				CPU:  79.5,
				Case: 72.0,
			},
			String: "example",
		},
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
	assert.Equal(t, expected, testStruct)
}

func TestEnvOnly(t *testing.T) {
	t.Setenv(testEnv, "local")
	t.Setenv(fmt.Sprintf("%sB", testPrefix), "bravo")
	t.Setenv(fmt.Sprintf("%sC_W", testPrefix), "watermelon")

	defer func() {
		t.Setenv(fmt.Sprintf("%sB", testPrefix), "")
		t.Setenv(fmt.Sprintf("%sC_W", testPrefix), "")
	}()

	cfg, err := config.NewConfiguration(
		nil,
		config.WithEnvPrefix(testPrefix),
	)
	require.NoError(t, err)

	testStruct := testConfig{
		C: nestedConfig{
			Y: "yamaha",
		},
	}

	err = cfg.Unmarshal("", &testStruct)
	require.NoError(t, err)

	expected := testConfig{
		A: "",      // not set
		B: "bravo", // env
		C: nestedConfig{
			W: "watermelon", // env
			X: "",           // not set
			Y: "yamaha",     // original struct
			Z: "",           // not set
		},
	}
	assert.Equal(t, expected, testStruct)
	assert.Equal(t, "local", cfg.Environment())
}
