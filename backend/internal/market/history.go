package market

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

var historyClient = &http.Client{
	Timeout: 10 * time.Second,
}

func FetchHistorical(symbol, interval string, limit int) ([]byte, error) {
	url := fmt.Sprintf(
		"https://api.binance.com/api/v3/klines?symbol=%s&interval=%s&limit=%d",
		symbol, interval, limit,
	)

	resp, err := historyClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("historical fetch failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("historical read failed: %w", err)
	}

	return body, nil
}

func ParseHistorical(body []byte) ([]Candle, error) {
	var raw [][]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("historical parse failed: %w", err)
	}

	candles := make([]Candle, 0, len(raw))
	for _, r := range raw {
		if len(r) < 6 {
			continue
		}

		openTime, _ := strconv.ParseInt(string(r[0]), 10, 64)

		candles = append(candles, Candle{
			OpenTime: time.UnixMilli(openTime),
			Open:     parseHistFloat(r[1]),
			High:     parseHistFloat(r[2]),
			Low:      parseHistFloat(r[3]),
			Close:    parseHistFloat(r[4]),
			Volume:   parseHistFloat(r[5]),
		})
	}

	return candles, nil
}

func FetchHistoricalRange(symbol, interval string, start, end time.Time) ([]Candle, error) {
	var all []Candle
	cursor := start

	for !cursor.After(end) {
		url := fmt.Sprintf(
			"https://api.binance.com/api/v3/klines?symbol=%s&interval=%s&startTime=%d&endTime=%d&limit=1000",
			symbol, interval, cursor.UnixMilli(), end.UnixMilli(),
		)

		resp, err := historyClient.Get(url)
		if err != nil {
			return nil, fmt.Errorf("historical range fetch failed: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("historical range read failed: %w", err)
		}

		batch, err := ParseHistorical(body)
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}

		all = append(all, batch...)
		last := batch[len(batch)-1].OpenTime
		cursor = last.Add(intervalStep(interval))

		if len(batch) < 1000 {
			break
		}
	}

	return all, nil
}

func intervalStep(interval string) time.Duration {
	switch interval {
	case "1m":
		return time.Minute
	case "15m":
		return 15 * time.Minute
	case "4h":
		return 4 * time.Hour
	default:
		return 15 * time.Minute
	}
}

func parseHistFloat(r json.RawMessage) float64 {
	var s string
	if err := json.Unmarshal(r, &s); err != nil {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
