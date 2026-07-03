package backtest

import (
	"fmt"
	"time"

	"github.com/abdullahshafaqat/trading-bot/internal/market"
)

func LoadCandles(symbol, interval string, start, end time.Time) ([]market.Candle, error) {
	candles, err := market.FetchHistoricalRange(symbol, interval, start, end)
	if err != nil {
		return nil, fmt.Errorf("load %s candles: %w", interval, err)
	}
	return candles, nil
}

func filterFrom(candles []market.Candle, from time.Time) []market.Candle {
	out := make([]market.Candle, 0)
	for _, c := range candles {
		if !c.OpenTime.Before(from) {
			out = append(out, c)
		}
	}
	return out
}

func filterBefore(candles []market.Candle, before time.Time) []market.Candle {
	out := make([]market.Candle, 0)
	for _, c := range candles {
		if c.OpenTime.Before(before) {
			out = append(out, c)
		}
	}
	return out
}

func filterAfter(candles []market.Candle, after time.Time) []market.Candle {
	out := make([]market.Candle, 0)
	for _, c := range candles {
		if c.OpenTime.After(after) {
			out = append(out, c)
		}
	}
	return out
}
