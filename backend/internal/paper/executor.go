package paper

import (
	"errors"
	"time"

	"github.com/abdullahshafaqat/trading-bot/internal/signal"
)

type Executor struct {
	tracker *Tracker
}

func NewExecutor(tracker *Tracker) *Executor {
	return &Executor{tracker: tracker}
}

func (e *Executor) Execute(sig *signal.Signal) error {
	if sig.Side == "BUY" {
		if !(sig.StopLoss < sig.Entry && sig.Entry < sig.TakeProfit) {
			return errors.New("invalid BUY levels")
		}
	} else if sig.Side == "SELL" {
		if !(sig.TakeProfit < sig.Entry && sig.Entry < sig.StopLoss) {
			return errors.New("invalid SELL levels")
		}
	}

	// Calculate TTL minutes from expiresAt
	ttlMinutes := int(sig.ExpiresAt.Sub(sig.CreatedAt).Minutes())
	if ttlMinutes <= 0 {
		ttlMinutes = 8 * 60 // Default fallback 8h
	}

	trade := Trade{
		ID:               sig.ID,
		TradeID:          sig.TradeID,
		Symbol:           sig.Symbol,
		Side:             sig.Side,
		Entry:            sig.Entry,
		SL:               sig.StopLoss,
		TP:               sig.TakeProfit,
		OpenedAt:         time.Now(),
		MarketRegime:     sig.MarketRegime,
		CrossID:          sig.CrossID,
		EntryToTPMinutes: sig.EntryToTPMinutes,
		EntryToSLMinutes: sig.EntryToSLMinutes,
		TTLMinutes:       ttlMinutes,
	}

	e.tracker.AddTrade(trade)
	return nil
}
