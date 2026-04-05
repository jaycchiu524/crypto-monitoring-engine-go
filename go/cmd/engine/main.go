package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"log/slog"

	"crypto-monitor/internal/broker"
	"crypto-monitor/internal/client"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Fetch required configuration
	symbol := os.Getenv("SYMBOL")
	if symbol == "" {
		symbol = "btcusdt"
	}
	symbol = strings.ToLower(symbol)

	role := os.Getenv("APP_ROLE")
	if role == "" {
		role = "fetcher" // default
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	logger.Info("Starting Crypto Monitoring Engine", "version", "0.2.0", "role", role, "symbol", symbol)

	// Setup Graceful Shutdown Context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Initialize Redis Broker
	redisBroker, err := broker.NewRedisBroker(redisURL, logger)
	if err != nil {
		logger.Error("Failed to initialize Redis Broker", "error", err)
		os.Exit(1)
	}
	defer redisBroker.Close()

	// Define standard channel name
	const redisChannel = "crypto_prices"

	// Dispatch logic based on role
	switch role {
	case "fetcher":
		logger.Info("Initializing as Fetcher")
		
		publishHandler := func(c context.Context, payload string) {
			if err := redisBroker.Publish(c, redisChannel, payload); err != nil {
				logger.Error("Failed to publish message to Redis", "error", err)
			}
		}

		binanceClient := client.NewBinanceClient(symbol, logger, publishHandler)
		binanceClient.Start(ctx) // Blocks until context canceled

	case "processor":
		logger.Info("Initializing as Processor")
		
		processHandler := func(payload string) {
			// In Phase 5 we will parse this and increment Prometheus metrics.
			// For now, simulating the processing by logging via the structured logger.
			logger.Info("Processed Event from Redis", "payload", payload)
		}

		// Blocks and listens on channel until context canceled
		redisBroker.Subscribe(ctx, redisChannel, processHandler)

	default:
		logger.Error("Unknown APP_ROLE specified", "role", role)
		os.Exit(1)
	}

	logger.Info("Crypto Monitoring Engine successfully shut down")
}
