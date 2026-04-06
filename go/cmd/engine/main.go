package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"crypto-monitor/internal/broker"
	"crypto-monitor/internal/client"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Define our labeled Prometheus metrics. 
var (
	cryptoPriceGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "crypto_price_usd",
		Help: "Current price of the tracked cryptocurrency in USD",
	}, []string{"symbol"})

	cryptoTradeCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "crypto_trades_total",
		Help: "Total number of trade events processed",
	}, []string{"symbol"})
)

type tradeEventInternal struct {
	Symbol string `json:"s"`
	Price  string `json:"p"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Fetch required configuration
	symbolsRaw := os.Getenv("SYMBOLS")
	if symbolsRaw == "" {
		symbolsRaw = "btcusdt"
	}
	symbols := strings.Split(symbolsRaw, ",")
	for i, s := range symbols {
		symbols[i] = strings.TrimSpace(strings.ToLower(s))
	}

	role := os.Getenv("APP_ROLE")
	if role == "" {
		role = "fetcher"
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	logger.Info("Starting Crypto Monitoring Engine", 
		"version", "0.4.0", 
		"role", role, 
		"symbols", symbols,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	redisBroker, err := broker.NewRedisBroker(redisURL, logger)
	if err != nil {
		logger.Error("Failed to initialize Redis Broker", "error", err)
		os.Exit(1)
	}
	defer redisBroker.Close()

	const redisChannel = "crypto_prices"

	if role == "processor" {
		go func() {
			logger.Info("Starting Prometheus metrics server on :8080/metrics")
			http.Handle("/metrics", promhttp.Handler())
			if err := http.ListenAndServe(":8080", nil); err != nil {
				logger.Error("Metrics server failed", "error", err)
			}
		}()
	}

	switch role {
	case "fetcher":
		logger.Info("Initializing as Multi-Stream Fetcher")
		
		publishHandler := func(c context.Context, payload string) {
			if err := redisBroker.Publish(c, redisChannel, payload); err != nil {
				logger.Error("Failed to publish to Redis", "error", err)
			}
		}

		binanceClient := client.NewBinanceClient(symbols, logger, publishHandler)
		binanceClient.Start(ctx)

	case "processor":
		logger.Info("Initializing as Labeled Metrics Processor")
		
		processHandler := func(payload string) {
			var event tradeEventInternal
			if err := json.Unmarshal([]byte(payload), &event); err != nil {
				logger.Warn("Failed to unmarshal Redis payload", "error", err)
				return
			}

			// We use the Binance symbol (which is uppercase) in the metric labels
			symbolLabel := strings.ToUpper(event.Symbol)

			var price float64
			_, err := fmt.Sscanf(event.Price, "%f", &price)
			if err != nil {
				logger.Warn("Failed to parse price", "price", event.Price, "error", err)
				return
			}

			// Update labeled metrics
			cryptoPriceGauge.WithLabelValues(symbolLabel).Set(price)
			cryptoTradeCounter.WithLabelValues(symbolLabel).Inc()

			logger.Debug("Update", "symbol", symbolLabel, "price", price)
		}

		redisBroker.Subscribe(ctx, redisChannel, processHandler)

	default:
		logger.Error("Unknown APP_ROLE", "role", role)
		os.Exit(1)
	}

	logger.Info("Crypto Monitoring Engine successfully shut down")
}
