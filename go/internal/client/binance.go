package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/gorilla/websocket"
)

// binanceTradeEvent models the subset of the Binance WebSocket trade event
// that we care about (specifically the symbol and the price).
type binanceTradeEvent struct {
	EventType string      `json:"e"` // Event type, e.g., "trade"
	EventTime json.Number `json:"E"` // Event time, handles both string or int64
	Symbol    string      `json:"s"` // Symbol, e.g., "BTCUSDT"
	Price     string      `json:"p"` // The live trade price (Binance sends this as a string)
}

// BinanceClient manages the WebSocket connection to the Binance API.
type BinanceClient struct {
	symbol  string
	logger  *slog.Logger
	handler func(context.Context, string)
}

// NewBinanceClient creates a new client for a given trading symbol.
func NewBinanceClient(symbol string, logger *slog.Logger, handler func(context.Context, string)) *BinanceClient {
	return &BinanceClient{
		symbol:  symbol,
		logger:  logger,
		handler: handler,
	}
}

// Start begins the WebSocket connection to process the stream.
// It will continuously attempt to reconnect on failure using an exponential backoff.
// Complex logic: The context is used to signal a graceful shutdown. If the context is canceled,
// the loop will exit cleanly.
func (c *BinanceClient) Start(ctx context.Context) {
	// Format the stream URL based on Binance's public spec (e.g., btcusdt@trade)
	url := fmt.Sprintf("wss://stream.binance.com:9443/ws/%s@trade", c.symbol)
	
	backoff := 1 * time.Second
	maxBackoff := 30 * time.Second

	for {
		// Check if we should shut down before attempting to connect
		select {
		case <-ctx.Done():
			c.logger.Info("Shutdown requested, stopping WebSocket client.")
			return
		default:
		}

		c.logger.Info("Connecting to Binance WebSocket...", "url", url)

		conn, _, err := websocket.DefaultDialer.DialContext(ctx, url, nil)
		if err != nil {
			c.logger.Error("Failed to connect", "error", err, "reconnecting_in", backoff)
			
			// Wait for the backoff duration, but allow the context to interrupt it
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
				// Exponential backoff logic
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			}
		}

		// Reset backoff on a successful connection
		backoff = 1 * time.Second
		c.logger.Info("Connected to Binance successfully")

		// Read and process messages. This will block until the connection is closed or an error occurs.
		c.readLoop(ctx, conn)

		// Ensure we clean up the connection before trying to reconnect
		conn.Close()
	}
}

// readLoop continuously reads messages from the active WebSocket connection.
func (c *BinanceClient) readLoop(ctx context.Context, conn *websocket.Conn) {
	// A channel to pipe errors from the read routine so we can break out of the loop
	errChan := make(chan error, 1)

	// We run the connection reading in a goroutine so we can gracefully shut down via the context.
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				errChan <- fmt.Errorf("read error: %w", err)
				return
			}

			var event binanceTradeEvent
			if err := json.Unmarshal(message, &event); err != nil {
				c.logger.Warn("Failed to unmarshal message", "error", err, "payload", string(message))
				continue
			}

			// Pass the raw message up to the handler if provided
			if c.handler != nil {
				c.handler(ctx, string(message))
			}
		}
	}()

	// Wait for either an error from the reader or a context cancellation
	select {
	case <-ctx.Done():
		c.logger.Info("Closing WebSocket connection due to shutdown signal")
		// Write a close message to gracefully close the websocket on the remote side
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	case err := <-errChan:
		c.logger.Error("Connection dropped", "reason", err)
	}
}
