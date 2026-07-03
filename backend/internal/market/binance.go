package market

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gorilla/websocket"

	"github.com/abdullahshafaqat/trading-bot/internal/logger"
)

type streamWrapper struct {
	Stream string          `json:"stream"`
	Data   json.RawMessage `json:"data"`
}

type klinePayload struct {
	K json.RawMessage `json:"k"`
}

func Stream(symbol string, interval string, out chan<- Candle) {
	url := fmt.Sprintf(
		"wss://stream.binance.com:9443/stream?streams=%s@kline_%s",
		toLower(symbol),
		interval,
	)

	for {
		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			logger.WS("WebSocket connect failed:", err)
			logger.WS("Retrying in 5s...")
			time.Sleep(5 * time.Second)
			continue
		}

		logger.WS("WebSocket connected:", symbol, interval)

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				logger.WS("WebSocket read error:", err)
				conn.Close()
				break
			}

			var wrapper streamWrapper
			if err := json.Unmarshal(msg, &wrapper); err != nil {
				continue
			}

			var payload klinePayload
			if err := json.Unmarshal(wrapper.Data, &payload); err != nil {
				continue
			}

			candle, ok := parseKline(symbol, interval, payload.K)
			if !ok {
				continue
			}

			out <- candle
		}

		logger.WS("Reconnecting in 3s...")
		time.Sleep(3 * time.Second)
	}
}

func parseKline(symbol, interval string, raw json.RawMessage) (Candle, bool) {
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &fields); err != nil {
		return Candle{}, false
	}

	if !rawBool(fields["x"]) {
		return Candle{}, false
	}

	return Candle{
		Symbol:   symbol,
		Interval: interval,
		OpenTime: time.UnixMilli(rawInt64(fields["t"])),
		Open:     rawFloat(fields["o"]),
		High:     rawFloat(fields["h"]),
		Low:      rawFloat(fields["l"]),
		Close:    rawFloat(fields["c"]),
		Volume:   rawFloat(fields["v"]),
		IsClosed: true,
	}, true
}

func rawFloat(raw json.RawMessage) float64 {
	if len(raw) == 0 {
		return 0
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0
	}
	return toFloat(s)
}

func rawInt64(raw json.RawMessage) int64 {
	if len(raw) == 0 {
		return 0
	}
	var v int64
	_ = json.Unmarshal(raw, &v)
	return v
}

func rawBool(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var v bool
	_ = json.Unmarshal(raw, &v)
	return v
}

func toFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		result[i] = c
	}
	return string(result)
}
