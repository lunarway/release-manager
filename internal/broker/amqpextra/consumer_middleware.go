package amqpextra

import (
	"context"
	"fmt"
	"time"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/makasim/amqpextra"
	"github.com/streadway/amqp"
)

func loggerMiddleware(logger *log.Logger) func(w amqpextra.Worker) amqpextra.Worker {
	return func(w amqpextra.Worker) amqpextra.Worker {
		return amqpextra.WorkerFunc(func(ctx context.Context, msg amqp.Delivery) interface{} {
			logger := logger.With(
				"routingKey", msg.RoutingKey,
				"messageId", msg.MessageId,
				"eventType", msg.Type,
				"correlationId", msg.CorrelationId,
				"redelivered", msg.Redelivered,
				"headers", fmt.Sprintf("%#v", msg.Headers),
			)
			logger.Infof("[consumer] Received message type=%s from exchange=%s routingKey=%s messageId=%s correlationId=%s timestamp=%s", msg.Type, msg.Exchange, msg.RoutingKey, msg.MessageId, msg.CorrelationId, msg.Timestamp)
			now := time.Now()

			err := w.ServeMsg(ctx, msg)

			duration := time.Since(now).Milliseconds()

			if err != nil {
				res := map[string]interface{}{
					"status":       "failed",
					"responseTime": duration,
					"error":        fmt.Sprintf("%+v", err),
					"redelivered":  msg.Redelivered,
				}
				logger.With("res", res).Errorf("[consumer] [FAILED] Failed to handle message: %v", err)
				return err
			}
			logger.With("res", map[string]interface{}{
				"status":       "ok",
				"responseTime": duration,
				"redelivered":  msg.Redelivered,
			}).Info("[consumer] [OK] Event handled successfully")
			return nil
		})
	}
}
