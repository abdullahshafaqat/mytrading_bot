package signal

import (
	"fmt"
	"time"

	"github.com/abdullahshafaqat/trading-bot/internal/logger"
	"github.com/abdullahshafaqat/trading-bot/internal/strategy"
)

type Filter struct {
	state         State
	cooldown      int
	cooldownCount int
	ttlMinutes    int
	lastSignal    *Signal
}

func NewFilter(cooldownCandles int, ttlMinutes int) *Filter {
	return &Filter{
		state:      StateWait,
		cooldown:   cooldownCandles,
		ttlMinutes: ttlMinutes,
	}
}

func (f *Filter) Process(result strategy.Result, symbol string, candleTime time.Time, confirm4hBullish bool) *Signal {

	if f.cooldownCount > 0 {
		f.cooldownCount--
		logger.Signalf("Cooldown active — %d candles remaining\n", f.cooldownCount)
		return nil
	}

	if result.Signal == strategy.HOLD {
		return nil
	}

	if result.Signal == strategy.BUY && f.state == StateBuyActive {
		logger.Signal("Blocked — BUY already active")
		return nil
	}
	if result.Signal == strategy.SELL && f.state == StateSellActive {
		logger.Signal("Blocked — SELL already active")
		return nil
	}

	now := time.Now().UTC()
	tradeID := fmt.Sprintf("%s-%s-%d", symbol, result.Signal, now.UnixMilli())
	crossID := fmt.Sprintf("%s-%d", symbol, candleTime.UnixMilli())
	regime := "range"
	if confirm4hBullish {
		if result.EMAFast > result.EMASlow {
			regime = "bull_trend"
		}
	} else if result.EMAFast < result.EMASlow {
		regime = "bear_trend"
	}
	sig := &Signal{
		ID:              tradeID,
		TradeID:         tradeID,
		CrossID:         crossID,
		SignalLatencyMs: now.Sub(candleTime).Milliseconds(),
		MarketRegime:    regime,
		Symbol:          symbol,
		Side:            string(result.Signal),
		Entry:           result.Entry,
		StopLoss:        result.StopLoss,
		TakeProfit:      result.TakeProfit,
		Reason:          result.Reason,
		CreatedAt:       now,
		ExpiresAt:       now.Add(time.Duration(f.ttlMinutes) * time.Minute),
	}

	if result.Signal == strategy.BUY {
		f.state = StateBuyActive
	} else {
		f.state = StateSellActive
	}

	f.cooldownCount = f.cooldown
	f.lastSignal = sig

	return sig
}

func (f *Filter) State() State {
	return f.state
}

func (f *Filter) Reset() {
	f.state = StateWait
	f.cooldownCount = 0
	f.lastSignal = nil
}
