package messagebus_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/zircuit-labs/zkr-go-common/config"
	"github.com/zircuit-labs/zkr-go-common/log"
	"github.com/zircuit-labs/zkr-go-common/messagebus"
)

type TestMessage struct {
	Content string `json:"content"`
}

type MockHandler struct {
	logger   *slog.Logger
	received chan TestMessage
}

func (h *MockHandler) HandleMessage(ctx context.Context, data TestMessage, subject string, metadata jetstream.MsgMetadata) error {
	h.received <- data
	h.logger.Debug("Received message",
		slog.String("subject", subject),
		slog.Any("data", data),
		slog.Any("metadata", metadata),
	)
	return nil
}

type NatsConsumerSuite struct {
	suite.Suite
	ctx         context.Context
	cancel      context.CancelFunc
	container   testcontainers.Container
	nc          *nats.Conn
	js          jetstream.JetStream
	consumerCfg *config.Configuration
	handler     *MockHandler
	natsURL     string
}

func (suite *NatsConsumerSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithTimeout(context.Background(), 1*time.Minute)

	req := testcontainers.ContainerRequest{
		Image:        "nats:latest",
		ExposedPorts: []string{"4222/tcp"},
		Cmd:          []string{"-js"},
		WaitingFor:   wait.ForListeningPort("4222/tcp"),
		HostConfigModifier: func(hc *container.HostConfig) {
			hc.PortBindings = map[nat.Port][]nat.PortBinding{
				"4222/tcp": {{HostIP: "0.0.0.0", HostPort: "4222"}},
			}
		},
	}
	suite.natsURL = "nats://127.0.0.1:4222"

	container, err := testcontainers.GenericContainer(suite.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	suite.Require().NoError(err)
	suite.container = container

	// Retry connection since NATS might not be fully ready even after port is available
	// (especially when running with -race flag which adds overhead)
	var nc *nats.Conn
	for i := 0; i < 10; i++ {
		nc, err = nats.Connect(suite.natsURL, nats.ReconnectWait(1*time.Second), nats.MaxReconnects(10))
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	suite.Require().NoError(err)

	suite.nc = nc

	js, err := jetstream.New(suite.nc)
	suite.Require().NoError(err)
	suite.js = js

	_, err = suite.js.CreateStream(suite.ctx, jetstream.StreamConfig{
		Name:     "TEST_STREAM",
		Subjects: []string{"test.subject"},
	})
	suite.Require().NoError(err)

	consumerCfg, err := config.NewConfigurationFromMap(
		map[string]any{
			"subject":      "test.subject",
			"stream":       "TEST_STREAM",
			"durablequeue": "test-consumer",
		},
	)
	suite.Require().NoError(err)

	suite.consumerCfg = consumerCfg

	suite.handler = &MockHandler{
		received: make(chan TestMessage, 10),
		logger:   log.NewTestLogger(suite.T()),
	}
}

func (suite *NatsConsumerSuite) TearDownSuite() {
	suite.cancel()
	suite.nc.Close()
	_ = suite.container.Terminate(context.Background())
}

func (suite *NatsConsumerSuite) TestMessageConsumption() {
	consumer, err := messagebus.NewNatsStreamConsumer[TestMessage](
		suite.consumerCfg,
		"",
		suite.handler,
	)
	suite.Require().NoError(err)

	ctx, cancel := context.WithCancel(suite.ctx)
	defer cancel()
	//nolint:errcheck // ok
	go consumer.Run(ctx)

	testMsg := TestMessage{"Hello Suite!"}
	data, _ := json.Marshal(testMsg)
	_, err = suite.js.Publish(ctx, "test.subject", data)
	suite.Require().NoError(err)

	select {
	case msg := <-suite.handler.received:
		suite.Equal("Hello Suite!", msg.Content)
	case <-time.After(5 * time.Second):
		suite.Fail("Did not receive message")
	}
}

func (suite *NatsConsumerSuite) TestReconnectLogic() {
	consumer, err := messagebus.NewNatsStreamConsumer[TestMessage](
		suite.consumerCfg,
		"",
		suite.handler,
		messagebus.WithLogger(log.NewTestLogger(suite.T())),
	)
	suite.Require().NoError(err)

	ctx, cancel := context.WithCancel(suite.ctx)
	defer cancel()
	//nolint:errcheck // ok
	go consumer.Run(ctx)

	msg1 := TestMessage{"First Message"}
	data1, _ := json.Marshal(msg1)
	_, err = suite.js.Publish(suite.ctx, "test.subject", data1)
	suite.Require().NoError(err)

	select {
	case received := <-suite.handler.received:
		suite.Equal(msg1.Content, received.Content)
	case <-time.After(3 * time.Second):
		suite.Fail("Initial message not received")
	}

	//nolint:errcheck // ok
	suite.container.Stop(suite.ctx, nil)
	//nolint:errcheck // ok
	suite.container.Start(suite.ctx)

	suite.nc.Close()
	suite.nc, err = nats.Connect(suite.natsURL)
	suite.Require().NoError(err)
	suite.js, err = jetstream.New(suite.nc)
	suite.Require().NoError(err)

	msg2 := TestMessage{"Second Message after Reconnect"}
	data2, _ := json.Marshal(msg2)
	_, err = suite.js.Publish(suite.ctx, "test.subject", data2)
	suite.Require().NoError(err)

	select {
	case received := <-suite.handler.received:
		suite.Equal(msg2.Content, received.Content)
	case <-time.After(3 * time.Second):
		suite.Fail("Message after reconnect not received")
	}
}

//nolint:paralleltest // should not run in parallel, since the tests are related
func TestNatsConsumerSuite_Docker(t *testing.T) {
	suite.Run(t, new(NatsConsumerSuite))
}

func TestCalculateNakDelay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		numDelivered uint64
		expected     time.Duration
	}{
		{
			name:         "First attempt",
			numDelivered: 1,
			expected:     200 * time.Millisecond, // 100ms << 1
		},
		{
			name:         "Second attempt",
			numDelivered: 2,
			expected:     400 * time.Millisecond, // 100ms << 2
		},
		{
			name:         "Third attempt",
			numDelivered: 3,
			expected:     800 * time.Millisecond, // 100ms << 3
		},
		{
			name:         "Fourth attempt",
			numDelivered: 4,
			expected:     1600 * time.Millisecond, // 100ms << 4 = 1.6s
		},
		{
			name:         "Ninth attempt (still under max)",
			numDelivered: 9,
			expected:     51200 * time.Millisecond, // 100ms << 9 = 51.2s
		},
		{
			name:         "Tenth attempt (at boundary, hits max)",
			numDelivered: 10,
			expected:     time.Minute, // Should hit max (1 minute)
		},
		{
			name:         "Eleventh attempt (over boundary)",
			numDelivered: 11,
			expected:     time.Minute, // Should return max delay
		},
		{
			name:         "High attempt count",
			numDelivered: 50,
			expected:     time.Minute, // Should return max delay
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			meta := &jetstream.MsgMetadata{
				NumDelivered: tt.numDelivered,
			}

			result := messagebus.CalculateNakDelay(meta)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateNakDelayExponentialProgression(t *testing.T) {
	t.Parallel()

	// Test that delays are actually exponential (each delay doubles)
	baseDelay := 100 * time.Millisecond
	maxDelay := time.Minute

	for i := uint64(1); i <= 8; i++ {
		meta := &jetstream.MsgMetadata{NumDelivered: i}
		delay := messagebus.CalculateNakDelay(meta)

		expectedDelay := baseDelay << i // 2^i * baseDelay

		// Verify it's exponential up to the max
		if expectedDelay < maxDelay {
			assert.Equal(t, expectedDelay, delay, "Delay should be exponential for attempt %d", i)
		} else {
			assert.Equal(t, maxDelay, delay, "Delay should cap at maxDelay for attempt %d", i)
		}
	}
}
