package broker

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/beganov/L0/internal/cache"
	"github.com/beganov/L0/internal/config"
	"github.com/beganov/L0/internal/database"
	"github.com/beganov/L0/internal/logger"
	"github.com/beganov/L0/internal/metrics"
	"github.com/beganov/L0/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/segmentio/kafka-go"
)

func ConsumeKafka(ctx context.Context, reader *kafka.Reader, db *pgxpool.Pool, cache *cache.OrderCache) {
	logger.Info("Kafka consumer started")

	for {
		select {
		case <-ctx.Done():
			logger.Info("Kafka consumer stopped")
			return
		default:
			timer := prometheus.NewTimer(metrics.KafkaProcessDuration)

			// read one message from Kafka
			kfkCtx, cancel := context.WithTimeout(ctx, config.KafkaTimeOut)
			msg, err := reader.FetchMessage(kfkCtx)
			metrics.KafkaMessagesTotal.Inc()
			cancel()

			if err != nil {
				metrics.KafkaErrorsTotal.Inc()
				if errors.Is(err, context.Canceled) {
					timer.ObserveDuration()
					return
				}
				logger.Error(err, "failed to read message")
				timer.ObserveDuration()
				continue
			}

			var order models.Order
			// parse message
			if err := json.Unmarshal(msg.Value, &order); err != nil {
				metrics.KafkaErrorsTotal.Inc()
				logger.Error(err, "json parse failed")
				commitMessage(ctx, reader, msg)
				timer.ObserveDuration()
				continue
			}

			// validate order
			if err := order.Validate(); err != nil {
				metrics.KafkaErrorsTotal.Inc()
				logger.Error(err, "order invalid")
				commitMessage(ctx, reader, msg)
				timer.ObserveDuration()
				continue
			}

			// save to db
			if err := database.SaveOrder(ctx, db, order); err != nil {
				metrics.KafkaErrorsTotal.Inc()
				logger.Error(err, "db save failed")
				timer.ObserveDuration()
				continue
			}

			// commit offset
			if err := reader.CommitMessages(ctx, msg); err != nil {
				metrics.KafkaErrorsTotal.Inc()
				logger.Error(err, "commit failed")
			}

			// update cache
			cache.Set(order.OrderUID, order)
			timer.ObserveDuration()
			logger.Info("order received", "orderID", order.OrderUID)
		}
	}
}

// helper for committing messages with logging
func commitMessage(ctx context.Context, reader *kafka.Reader, msg kafka.Message) {
	if err := reader.CommitMessages(ctx, msg); err != nil {
		metrics.KafkaErrorsTotal.Inc()
		logger.Error(err, "commit failed")
	}
}
