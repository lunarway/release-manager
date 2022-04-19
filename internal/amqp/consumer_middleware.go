package amqp

import (
	"context"
	"fmt"
	"time"

	"github.com/makasim/amqpextra/consumer"
	amqp "github.com/rabbitmq/amqp091-go"
)

func loggerMiddleware(logger Logger) consumer.Middleware {
	return func(next consumer.Handler) consumer.Handler {
		return consumer.HandlerFunc(func(ctx context.Context, msg amqp.Delivery) interface{} {
			loggingContext := fmt.Sprintf("type='%s' exchange='%s' routingKey='%s' messageId='%s' correlationId='%s'", msg.Type, msg.Exchange, msg.RoutingKey, msg.MessageId, msg.CorrelationId)
			logger.Infof("[amqp] Received message %s", loggingContext)
			now := time.Now()

			err := next.Handle(ctx, msg)

			duration := time.Since(now).Milliseconds()

			if err != nil {
				logger.Errorf("[amqp] [FAILED] Failed to handle message %s status='failed' responseTime='%d' error='%v'", loggingContext, duration, err)
				return err
			}
			logger.Infof("[amqp] [OK] Event handled successfully %s status='ok' responseTime='%d'", loggingContext, duration)
			return nil
		})
	}
}
