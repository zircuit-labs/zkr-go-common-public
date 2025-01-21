package messagebus_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zircuit-labs/zkr-go-common/calm/errgroup"
	"github.com/zircuit-labs/zkr-go-common/config"
	"github.com/zircuit-labs/zkr-go-common/log"
	"github.com/zircuit-labs/zkr-go-common/messagebus"
)

var sampleMessages = []sampleMessage{
	{
		Message: "hello world!",
		Integer: 123,
		Nested: struct {
			FloatA float32 `json:"float_a"`
			FloatB float32 `json:"float_b"`
		}{
			FloatA: 1.23,
			FloatB: 4.56,
		},
		Boolean: true,
	},
	{
		Message: "once again, with feeling",
		Integer: 12345,
		Nested: struct {
			FloatA float32 `json:"float_a"`
			FloatB float32 `json:"float_b"`
		}{
			FloatA: 91.23,
			FloatB: 84.56,
		},
		Boolean: false,
	},
}

// TestNatsStreamAsync produces several messages to a durable queue then consumes all of them ensuring they match
func TestNatsStreamAsync(t *testing.T) {
	t.Parallel()
	nc := getNatsConnection(t)

	cfg, err := config.NewConfigurationFromMap(
		map[string]any{
			"subject": "foo",
			"stream":  "FOO",
			"durable": "bar",
		},
	)
	require.NoError(t, err)

	// No messages have been produced, so validate that GetLastMessage handles this
	_, _, err = messagebus.GetLastMessage[sampleMessage](cfg, "", messagebus.WithNATSConnection(nc))
	assert.ErrorIs(t, err, messagebus.ErrNoMessages)

	// produce several messages
	producer, err := messagebus.NewNatsStreamProducer[sampleMessage](cfg, "", messagebus.WithNATSConnection(nc))
	require.NoError(t, err)
	t.Cleanup(producer.Close)

	ctx := context.Background()
	for _, m := range sampleMessages {
		err := producer.Produce(ctx, m)
		assert.NoError(t, err)
	}

	// ensure that GetLastMessage correctly identifies the last message produced
	lastMessage, _, err := messagebus.GetLastMessage[sampleMessage](cfg, "", messagebus.WithNATSConnection(nc))
	assert.NoError(t, err)
	assert.Equal(t, sampleMessages[1], lastMessage)

	// Since these were produced to durable queues,
	// the messages are safely waiting for a consumer

	// consume messages
	handler := &streamConsumerHandler[sampleMessage]{
		Messages:         []sampleMessage{},
		Subjects:         []string{},
		ExpectedMessages: len(sampleMessages),
		Done:             make(chan struct{}),
	}
	consumer, err := messagebus.NewNatsStreamConsumer(cfg, "", handler, messagebus.WithNATSConnection(nc))
	assert.NoError(t, err)

	// run the consumer in the background
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	t.Cleanup(cancel)
	group, _ := errgroup.WithContext(ctx)
	group.Go(func() error {
		// If Run returns early, cancel the context
		err := consumer.Run(ctx)
		cancel()
		return err
	})

	// wait for all expected messages (or timeout)
	select {
	case <-handler.Done:
		cancel()
	case <-ctx.Done():
	}

	// wait for consumer to stop
	err = group.Wait()
	require.NoError(t, err)

	// received messages should be identical to those sent
	assert.Equal(t, sampleMessages, handler.Messages)
	// All messages should be on expected subject
	for _, s := range handler.Subjects {
		assert.Equal(t, "foo", s)
	}
}

func TestPublisherWithSubjectTransform(t *testing.T) {
	t.Parallel()
	nc := getNatsConnection(t)

	cfg, err := config.NewConfigurationFromMap(
		map[string]any{
			"subject": "waldo",
			"stream":  "WALDO",
		},
	)
	require.NoError(t, err)

	// No messages have been produced, so validate that GetLastMessage handles this
	_, meta, err := messagebus.GetLastMessage[sampleMessage](cfg, "", messagebus.WithNATSConnection(nc))
	assert.ErrorIs(t, err, messagebus.ErrNoMessages)
	assert.Nil(t, meta)

	// produce several messages
	producer, err := messagebus.NewNatsStreamProducer[sampleMessage](cfg, "", messagebus.WithNATSConnection(nc))
	require.NoError(t, err)
	t.Cleanup(producer.Close)

	// messages now should be produced on subjects that leverage the produced data contents
	producer.SetSubjectTransform(func(data sampleMessage, defaultSubject string) string {
		return fmt.Sprintf("%s.%d", defaultSubject, data.Integer)
	})

	ctx := context.Background()
	for _, m := range sampleMessages {
		err := producer.Produce(ctx, m)
		assert.NoError(t, err)
	}

	// consume messages
	handler := &streamConsumerHandler[sampleMessage]{
		Messages:         []sampleMessage{},
		Subjects:         []string{},
		ExpectedMessages: len(sampleMessages),
		Done:             make(chan struct{}),
	}

	// we will need to consume from "waldo.>" rather than just "waldo"
	cfgB, err := config.NewConfigurationFromMap(
		map[string]any{
			"subject": "waldo.>",
			"durable": "waldo",
			"stream":  "WALDO",
		},
	)
	require.NoError(t, err)

	consumer, err := messagebus.NewNatsStreamConsumer(cfgB, "", handler, messagebus.WithNATSConnection(nc))
	assert.NoError(t, err)

	// run the consumer in the background
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	t.Cleanup(cancel)
	group, _ := errgroup.WithContext(ctx)
	group.Go(func() error {
		// If Run returns early, cancel the context
		err := consumer.Run(ctx)
		cancel()
		return err
	})

	// wait for all expected messages (or timeout)
	select {
	case <-handler.Done:
		cancel()
	case <-ctx.Done():
	}

	// wait for consumer to stop
	err = group.Wait()
	require.NoError(t, err)

	// received messages should be identical to those sent
	assert.Equal(t, sampleMessages, handler.Messages)
	// Messages should be on expected subject
	assert.Equal(t, []string{"waldo.123", "waldo.12345"}, handler.Subjects)
}

func TestConsumerSubjectTransform(t *testing.T) {
	t.Parallel()
	logger := log.NewTestLogger(t)

	type message struct {
		Version string
		Value   int
	}

	messages := []message{
		{
			Version: "v1",
			Value:   1,
		},
		{
			Version: "v1",
			Value:   2,
		},
		{
			Version: "v2",
			Value:   1,
		},
		{
			Version: "v1",
			Value:   3,
		},
		{
			Version: "v2",
			Value:   2,
		},
		{
			Version: "v2",
			Value:   3,
		},
		{
			Version: "v2",
			Value:   4,
		},
	}

	nc := getNatsConnection(t)

	// set up a producer with a subject transform
	cfg, err := config.NewConfigurationFromMap(
		map[string]any{
			"subject": "corge",
			"stream":  "CORGE",
		},
	)
	require.NoError(t, err)
	producer, err := messagebus.NewNatsStreamProducer[message](cfg, "", messagebus.WithNATSConnection(nc), messagebus.WithLogger(logger))
	require.NoError(t, err)
	t.Cleanup(producer.Close)
	producer.SetSubjectTransform(func(data message, defaultSubject string) string {
		return fmt.Sprintf("%s.%s.%d", defaultSubject, data.Version, data.Value)
	})

	// produce the messages
	ctx := context.Background()
	for _, m := range messages {
		err := producer.Produce(ctx, m)
		require.NoError(t, err)
	}

	// produce several messages to multiple subjects
	// consume with transform one specific set,
	// then another
	// validate that the messages are correctly consumed
	// consume messages

	handlerv1 := &streamConsumerHandler[message]{
		Messages:         []message{},
		Subjects:         []string{},
		ExpectedMessages: 3,
		Done:             make(chan struct{}),
	}

	handlerv2 := &streamConsumerHandler[message]{
		Messages:         []message{},
		Subjects:         []string{},
		ExpectedMessages: 4,
		Done:             make(chan struct{}),
	}

	consumerCfg, err := config.NewConfigurationFromMap(
		map[string]any{
			"subject":      "corge.<version>.*",
			"stream":       "CORGE",
			"durablequeue": "grault",
		},
	)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	t.Cleanup(cancel)
	group, _ := errgroup.WithContext(ctx)

	consumerv1, err := messagebus.NewNatsStreamConsumer[message](
		consumerCfg,
		"",
		handlerv1,
		messagebus.WithNATSConnection(nc),
		messagebus.WithLogger(logger),
		messagebus.WithConsumerSubjectTransform(map[string]string{"<version>": "v1"}),
	)
	require.NoError(t, err)

	// run consumerv1 in the background
	group.Go(func() error {
		return consumerv1.Run(ctx)
	})

	// wait for all expected messages (or timeout)
	select {
	case <-handlerv1.Done:
	case <-ctx.Done():
	}

	consumerv2, err := messagebus.NewNatsStreamConsumer[message](
		consumerCfg,
		"",
		handlerv2,
		messagebus.WithNATSConnection(nc),
		messagebus.WithLogger(logger),
		messagebus.WithConsumerSubjectTransform(map[string]string{"<version>": "v2"}),
	)
	require.NoError(t, err)

	// run consumerv2 in the background
	group.Go(func() error {
		return consumerv2.Run(ctx)
	})

	// wait for all expected messages (or timeout)
	select {
	case <-handlerv2.Done:
		cancel() // test is over, don't wait for the timeout
	case <-ctx.Done():
	}

	// wait for both consumers to stop
	err = group.Wait()
	require.NoError(t, err)

	// each consumer should have received the correct messages
	assert.Equal(t, []message{messages[0], messages[1], messages[3]}, handlerv1.Messages)
	assert.Equal(t, []message{messages[2], messages[4], messages[5], messages[6]}, handlerv2.Messages)
}

func TestNatsGetLastMessage(t *testing.T) {
	t.Parallel()
	nc := getNatsConnection(t)

	cfg, err := config.NewConfigurationFromMap(
		map[string]any{
			"subject": "qux",
			"stream":  "QUX",
		},
	)
	require.NoError(t, err)

	// No messages have been produced, so validate that GetLastMessage handles this
	_, meta, err := messagebus.GetLastMessage[sampleMessage](cfg, "", messagebus.WithNATSConnection(nc))
	assert.ErrorIs(t, err, messagebus.ErrNoMessages)
	assert.Nil(t, meta)

	// Produce a message
	producer, err := messagebus.NewNatsStreamProducer[sampleMessage](cfg, "", messagebus.WithNATSConnection(nc))
	require.NoError(t, err)
	t.Cleanup(producer.Close)

	ctx := context.Background()
	err = producer.Produce(ctx, sampleMessages[0])
	assert.NoError(t, err)

	// ensure that GetLastMessage correctly identifies the last message produced
	// do this multiple times
	for range 5 {
		lastMessage, meta, err := messagebus.GetLastMessage[sampleMessage](cfg, "", messagebus.WithNATSConnection(nc))
		assert.NoError(t, err)
		assert.Equal(t, sampleMessages[0], lastMessage)
		assert.NotNil(t, meta)
		// This was the first message published to the stream
		assert.Equal(t, uint64(1), meta.Sequence.Stream)
	}

	// produce another message
	err = producer.Produce(ctx, sampleMessages[1])
	assert.NoError(t, err)

	// ensure that GetLastMessage correctly identifies the last message produced
	// do this multiple times
	for range 5 {
		lastMessage, meta, err := messagebus.GetLastMessage[sampleMessage](cfg, "", messagebus.WithNATSConnection(nc))
		assert.NoError(t, err)
		assert.Equal(t, sampleMessages[1], lastMessage)
		assert.NotNil(t, meta)
		// This was the second message published to the stream
		assert.Equal(t, uint64(2), meta.Sequence.Stream)
	}
}

type sampleMessage struct {
	Message string
	Integer int
	Nested  struct {
		FloatA float32 `json:"float_a"`
		FloatB float32 `json:"float_b"`
	}
	Boolean bool
}

var (
	encodedMessage = []byte(`{"message":"example message","integer":91,"nested":{"float_a": 9.87,"float_b": 6.54},"boolean":true}`)
	decodedMesage  = sampleMessage{
		Message: "example message",
		Integer: 91,
		Nested: struct {
			FloatA float32 `json:"float_a"`
			FloatB float32 `json:"float_b"`
		}{
			FloatA: 9.87,
			FloatB: 6.54,
		},
		Boolean: true,
	}
)

type streamConsumerHandler[T any] struct {
	Messages         []T
	Subjects         []string
	ExpectedMessages int
	Done             chan struct{}
}

func (s *streamConsumerHandler[T]) HandleMessage(_ context.Context, message T, subject string, _ jetstream.MsgMetadata) error {
	s.Messages = append(s.Messages, message)
	s.Subjects = append(s.Subjects, subject)
	if len(s.Messages) >= s.ExpectedMessages {
		close(s.Done)
	}
	return nil
}

// TestJSONDecoder demonstrates that the NatsStreamConsumer correctly decodes raw JSON by default.
func TestJSONDecoder(t *testing.T) {
	t.Parallel()
	nc := getNatsConnection(t)
	js := getJetStream(t, nc)

	// Push raw json directly into NATS
	_, err := js.Publish(context.Background(), "baz", encodedMessage)
	require.NoError(t, err)

	cfg, err := config.NewConfigurationFromMap(
		map[string]any{
			"subject": "baz",
			"stream":  "BAZ",
			"durable": "qux",
		},
	)
	require.NoError(t, err)

	// consume messages
	handler := &streamConsumerHandler[sampleMessage]{
		Messages:         []sampleMessage{},
		Subjects:         []string{},
		ExpectedMessages: 1,
		Done:             make(chan struct{}),
	}
	consumer, err := messagebus.NewNatsStreamConsumer[sampleMessage](cfg, "", handler, messagebus.WithNATSConnection(nc))
	assert.NoError(t, err)

	// run the consumer in the background
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	t.Cleanup(cancel)
	group, _ := errgroup.WithContext(ctx)
	group.Go(func() error {
		// If Run returns early, cancel the context
		err := consumer.Run(ctx)
		cancel()
		return err
	})

	// wait for all expected messages (or timeout)
	select {
	case <-handler.Done:
		cancel()
	case <-ctx.Done():
	}

	// wait for consumer to stop
	err = group.Wait()
	require.NoError(t, err)

	// received message should be correctly decoded
	assert.Equal(t, []sampleMessage{decodedMesage}, handler.Messages)
	// received message should be received on expected subject
	assert.Equal(t, []string{"baz"}, handler.Subjects)
}
