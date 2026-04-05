package broker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

// RedisBroker manages our connection to the Redis network instance
type RedisBroker struct {
	client *redis.Client
	logger *slog.Logger
}

// NewRedisBroker creates a new broker pointing to the specified URL
func NewRedisBroker(redisAddr string, logger *slog.Logger) (*RedisBroker, error) {
	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// PING the server to ensure we can connect right at startup
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("could not connect to redis: %w", err)
	}

	logger.Info("Connected to Redis successfully", "addr", redisAddr)

	return &RedisBroker{
		client: client,
		logger: logger,
	}, nil
}

// Publish broadcasts a payload to a specific Redis channel
func (b *RedisBroker) Publish(ctx context.Context, channel string, payload string) error {
	return b.client.Publish(ctx, channel, payload).Err()
}

// Subscribe listens to a specific Redis channel and passes messages to a handler function.
// It runs continuously until the context is canceled.
func (b *RedisBroker) Subscribe(ctx context.Context, channel string, handler func(string)) {
	pubsub := b.client.Subscribe(ctx, channel)
	defer pubsub.Close()

	ch := pubsub.Channel()

	b.logger.Info("Subscribed to Redis channel", "channel", channel)

	for {
		select {
		case msg := <-ch:
			// Execute the handler function for each received message
			handler(msg.Payload)
		case <-ctx.Done():
			b.logger.Info("Stopping Redis subscriber due to context cancellation")
			return
		}
	}
}

// Close explicitly closes the Redis connection
func (b *RedisBroker) Close() error {
	return b.client.Close()
}
