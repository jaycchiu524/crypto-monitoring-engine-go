package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// binanceTradeEvent models the subset of the Binance WebSocket trade event.
type binanceTradeEvent struct {
	EventType string      `json:"e"` // Event type, e.g., "trade"
	EventTime json.Number `json:"E"` // Event time, handles both string or int64
	Symbol    string      `json:"s"` // Symbol, e.g., "BTCUSDT"
	Price     string      `json:"p"` // The live trade price (Binance sends this as a string)
}

// combinedStreamEvent is the wrapper Binance uses when multiple streams are requested.
type combinedStreamEvent struct {
	Stream string          `json:"stream"`
	Data   json.RawMessage `json:"data"`
}

// BinanceClient manages the WebSocket connection for many symbols.
type BinanceClient struct {
	symbols []string
	logger  *slog.Logger
	handler func(context.Context, string)
}

// NewBinanceClient creates a client for a list of trading symbols.
func NewBinanceClient(symbols []string, logger *slog.Logger, handler func(context.Context, string)) *BinanceClient {
	return &BinanceClient{
		symbols: symbols,
		logger:  logger,
		handler: handler,
	}
}

// Start begins the Combined WebSocket connection.
func (c *BinanceClient) Start(ctx context.Context) {
	// Construct the combined stream URL:
	// wss://stream.binance.com:9443/stream?streams=btcusdt@trade/ethusdt@trade
	var streams []string
	for _, s := range c.symbols {
		streams = append(streams, fmt.Sprintf("%s@trade", strings.ToLower(s)))
	}
	url := fmt.Sprintf("wss://stream.binance.com:9443/stream?streams=%s", strings.Join(streams, "/"))

	backoff := 1 * time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Shutdown requested, stopping Multi-WebSocket client.")
			return
		default:
		}

		c.logger.Info("Connecting to Binance Multi-Stream WebSocket...", "url", url)

		conn, _, err := websocket.DefaultDialer.DialContext(ctx, url, nil)
		if err != nil {
			c.logger.Error("Failed to connect to combined stream", "error", err, "reconnecting_in", backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			}
		}

		backoff = 1 * time.Second
		c.logger.Info("Connected to Binance combined stream successfully")

		c.readLoop(ctx, conn)
		conn.Close()
	}
}

// readLoop reads the multi-stream messages and extracts the 'data' portion.
func (c *BinanceClient) readLoop(ctx context.Context, conn *websocket.Conn) {
	errChan := make(chan error, 1)

	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				errChan <- fmt.Errorf("read error: %w", err)
				return
			}

			// Since we are using combined streams, the message is wrapped.
			var combined combinedStreamEvent
			if err := json.Unmarshal(message, &combined); err != nil {
				c.logger.Warn("Failed to unmarshal outer combined stream", "error", err)
				continue
			}

			// We pass the inner 'data' payload to the handler (this keeps the handler format the same as Phase 2).
			if c.handler != nil {
				c.handler(ctx, string(combined.Data))
			}
		}
	}()

	select {
	case <-ctx.Done():
		c.logger.Info("Closing Multi-WebSocket connection due to shutdown")
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	case err := <-errChan:
		c.logger.Error("Multi-Stream connection dropped", "reason", err)
	}
}
