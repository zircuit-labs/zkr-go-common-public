# Package `messagebus`

This package provides methods and tasks for inter-service communication using [NATS](https://nats.io/).

NATS uses subject-based addressing to determine which messages are sent where. Be sure to read about this before using NATS.

## JetStream (ie Async Durable Messaging)

Using JetStream to create streams, messages are stored within NATS and are not removed until conditions are met (those conditions being set when the stream is created). Producers and consumers are thus able to operate independently from each other.

Use `NatsStreamProducer` to produce to these streams, and `NatsStreamConsumer` to consume. The consumer's `DurableQueue` name is used to load-balance messages between instances that share the same name. Different names each get a copy of the same messages.

Due to the durable nature of these queues, failures from the handler are retryable in more graceful manner (allowing for at-least-once semantics). Retries are set up to be automated based on the returned error from the handler:
- nil means the message was handled correctly, and the message will not be retried
- an error wrapped as `Persistent` or `Panic` means the message can never be handled and will not be retried
- any other error will result in a NAK to NATS which will cause the message to be retried later

**NOTE:** The actual durability of messages on these streams is dependant entirely on how they have been set up, and is more of an infrastructure issue than one of code.

