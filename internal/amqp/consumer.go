package amqp

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/makasim/amqpextra/consumer"
	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"
	"k8s.io/utils/strings"
)

// StartConsumer consumes messages from an AMQP queue. The method is blocking
// and will always return with nil after calls to Close.
//
// The consumers' queues are declared on startup along with a binding to the
// exchange with specified routing key.
//
// Channel constumersStarted is closed when all consumers have been initialized.
func (w *Worker) StartConsumer(consumerConfigs []ConsumerConfig, consumersStarted chan struct{}) error {
	// stopped is used to track that all started Go routines are stopped. ready is
	// used to track that all consumers have been initialized and become ready.
	var stopped, ready sync.WaitGroup
	for _, consumerConfig := range consumerConfigs {
		stopped.Add(1)
		ready.Add(1)

		// ensure that queues and exchanges are declared and bound to each other up
		// front.
		err := w.initializeConsumer(consumerConfig)
		if err != nil {
			return errors.WithMessagef(err, "initialize consumer on exchange %v routing keys %v", consumerConfig.Exchange, consumerConfig.RoutingPatterns)
		}

		prefixedQueue := w.prefixed(consumerConfig.Queue)
		if consumerConfig.Prefetch > 0 {
			w.logger.Infof("[amqp] Setting prefetch to '%d' for queue '%s'", consumerConfig.Prefetch, prefixedQueue)
		}

		if consumerConfig.WorkerCount == 0 {
			consumerConfig.WorkerCount = 1
		}
		w.logger.Infof("[amqp] Setting workers to '%d' for queue '%s'", consumerConfig.WorkerCount, prefixedQueue)

		consumerState := make(chan consumer.State, 2)
		amqpConsumer, err := w.dialer.Consumer(
			consumer.WithQueue(prefixedQueue),
			consumer.WithNotify(consumerState),
			consumer.WithHandler(consumer.Wrap(handlerFunc(consumerConfig.Handle), loggerMiddleware(w.logger))),
			consumer.WithLogger(newLogger(w.logger, "consumer")),
			consumer.WithConsumeArgs("", false, false, false, false, nil),
			consumer.WithQos(consumerConfig.Prefetch, false),
			consumer.WithWorker(consumer.NewParallelWorker(consumerConfig.WorkerCount)),
		)
		if err != nil {
			return errors.WithMessagef(err, "instantiate consumer on exchange %v routing keys %v to queue %v", consumerConfig.Exchange, consumerConfig.RoutingPatterns, prefixedQueue)
		}

		// track when the consumer is ready
		go func() {
			waitForConsumerReady(consumerState)
			ready.Done()
		}()
		// shutdown is handled when the consumers are closed in worker.Close
		go func() {
			defer stopped.Done()
			<-amqpConsumer.NotifyClosed()
			w.logger.Infof("[amqp] Consumer on queue %v closed", prefixedQueue)
		}()
	}
	// when all consumers are ready, the ready WaitGroup is Done and we signal to
	// the caller by closing the consumersStarted channel.
	go func() {
		ready.Wait()
		w.logger.Debugf("[amqp] All consumers started")
		close(consumersStarted)
	}()
	stopped.Wait()
	return nil
}

// waitForConsumerReady waits until the consumer state becomes Ready. It stops
// only when the consumerState channel is closed.
func waitForConsumerReady(consumerState chan consumer.State) {
	for {
		select {
		case state := <-consumerState:
			if state.Ready != nil {
				return
			}
		}
	}
}

// initializeConsumer initializes a consumer by declaring exchange and queue and
// configuring a binding between them.
func (w *Worker) initializeConsumer(c ConsumerConfig) error {
	conn, err := w.dialer.Connection(context.Background())
	if err != nil {
		return err
	}

	prefixedExchange := w.prefixed(c.Exchange)
	prefixedQueue := w.prefixed(c.Queue)

	channel, err := conn.Channel()
	if err != nil {
		return errors.WithMessage(err, "create channel")
	}

	w.logger.Infof("[amqp] Declaring consumer exchange '%s'", prefixedExchange)
	err = channel.ExchangeDeclare(
		prefixedExchange,
		amqp.ExchangeTopic,
		true,
		false,
		false,
		false,
		nil)
	if err != nil {
		return errors.WithMessagef(err, "declare exchange '%s'", prefixedExchange)
	}

	w.logger.Infof("[amqp] Declaring queue '%s'", prefixedQueue)
	queueArgs := amqp.Table{
		"x-queue-type":             "quorum",
		"x-single-active-consumer": true,
	}
	_, err = channel.QueueDeclare(
		prefixedQueue,   // name
		c.DurableQueue,  // durable
		!c.DurableQueue, // delete when unused. We set this to the negation of DurableQueue: either our queues are durable or they live and die with the service creating them
		false,           // exclusive
		false,           // no-wait
		queueArgs,       // arguments
	)
	if err != nil {
		return errors.WithMessagef(err, "declare queue '%s'", prefixedQueue)
	}

	for _, r := range c.RoutingPatterns {
		w.logger.Infof("[amqp] Binding queue '%s' to exchange '%s' with routing pattern '%s'", prefixedQueue, prefixedExchange, r)
		err = channel.QueueBind(prefixedQueue, r, prefixedExchange, false, nil)
		if err != nil {
			return errors.WithMessagef(err, "bind queue '%s' to exchange '%s'", prefixedQueue, prefixedExchange)
		}
	}

	// Configure Github webhook exchange
	webhookExchange := fmt.Sprintf("%s-github-webhook", prefixedExchange)
	err = channel.ExchangeDeclare(
		webhookExchange,
		amqp.ExchangeFanout,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return errors.WithMessagef(err, "declare github webhook exchange '%s'", webhookExchange)
	}

	// Configure Github webhook queue
	uuid := strings.ShortenString(uuid.NewString(), 5)
	webhookQueue := fmt.Sprintf("%s-github-webhook-%s", prefixedQueue, uuid)
	_, err = channel.QueueDeclare(
		webhookQueue,
		true,
		true,
		true,
		false,
		nil)
	if err != nil {
		return errors.WithMessagef(err, "declare github webhook queue '%s'", webhookQueue)
	}

	err = channel.QueueBind(webhookQueue, "", webhookExchange, false, nil)
	if err != nil {
		return errors.WithMessagef(err, "bind github webhook queue '%s' to exchange '%s'", webhookQueue, webhookExchange)
	}

	return nil
}

type handlerFunc func(*amqp.Delivery) error

func (handler handlerFunc) Handle(ctx context.Context, msg amqp.Delivery) interface{} {
	return handler(&msg)
}

// ConsumerConf is the configuration struct for a RabbitMQ consumer
type ConsumerConfig struct {
	// the exchange
	Exchange string
	// the queue to bind to the exchange
	Queue string
	// whether to create the queue as a durable queue. Non durable queues will be deleted and the binding removed when the last consumer unsubscribes
	DurableQueue bool
	// the routing patterns to bind to
	RoutingPatterns []string
	// the prefetch size, i.e. the limit on the number of unacknowledged messages can be received at once. Set to 0 to disable.
	Prefetch int
	// the handler func. The handler func must itself take care of ack/nack/rejecting messages
	Handle func(*amqp.Delivery) error
	// The number of concurrent workers. If it is not set then it will default to 1.
	WorkerCount int
}
