package paper

import (
	"sync"
	"time"

	"github.com/abdullahshafaqat/trading-bot/internal/logger"
)

type Store interface {
	SavePaperTrade(t Trade) error
	UpdatePaperTradeClose(t Trade) error
	LoadOpenPaperTrades() ([]Trade, error)
	GetPaperMetrics() (Metrics, error)
	SavePaperMetrics(m Metrics) error
	GetPaperHistory(limit int) ([]Trade, error)
}

type Tracker struct {
	mu      sync.Mutex
	store   Store
	open    map[string]*Trade
	metrics Metrics
}

func NewTracker(store Store) *Tracker {
	trades, _ := store.LoadOpenPaperTrades()
	openMap := make(map[string]*Trade)
	for i := range trades {
		openMap[trades[i].ID] = &trades[i]
	}

	m, err := store.GetPaperMetrics()
	if err != nil {
		m = Metrics{UpdatedAt: time.Now()}
	}

	return &Tracker{
		store:   store,
		open:    openMap,
		metrics: m,
	}
}

func (t *Tracker) AddTrade(tr Trade) {
	t.mu.Lock()
	defer t.mu.Unlock()

	tr.Outcome = "OPEN"
	t.open[tr.ID] = &tr
	t.store.SavePaperTrade(tr)
}

func (t *Tracker) Update(candleHigh, candleLow, candleClose float64, candleTime time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()

	var closed []Trade

	for id, tr := range t.open {
		outcome := ""
		pnl := 0.0

		// Check TTL
		if tr.TTLMinutes > 0 {
			expiresAt := tr.OpenedAt.Add(time.Duration(tr.TTLMinutes) * time.Minute)
			if candleTime.After(expiresAt) {
				outcome = "EXPIRED"
				if tr.Side == "BUY" {
					pnl = (candleClose - tr.Entry) / tr.Entry * 100
				} else {
					pnl = (tr.Entry - candleClose) / tr.Entry * 100
				}
			}
		}

		if outcome == "" {
			if tr.Side == "BUY" {
				if candleLow <= tr.SL {
					outcome = "LOSS"
					pnl = (tr.SL - tr.Entry) / tr.Entry * 100
				} else if candleHigh >= tr.TP {
					outcome = "WIN"
					pnl = (tr.TP - tr.Entry) / tr.Entry * 100
				}
			} else if tr.Side == "SELL" {
				if candleHigh >= tr.SL {
					outcome = "LOSS"
					pnl = (tr.Entry - tr.SL) / tr.Entry * 100
				} else if candleLow <= tr.TP {
					outcome = "WIN"
					pnl = (tr.Entry - tr.TP) / tr.Entry * 100
				}
			}
		}

		if outcome != "" {
			tr.Outcome = outcome
			now := candleTime
			tr.ClosedAt = &now
			tr.PnL = pnl
			closed = append(closed, *tr)
			delete(t.open, id)
			logger.Botf("Paper Trade %s Closed: %s (PnL: %.2f%%)", tr.ID, outcome, pnl)
		}
	}

	if len(closed) > 0 {
		t.updateMetrics(closed)
	}
}

func (t *Tracker) updateMetrics(closed []Trade) {
	for _, tr := range closed {
		t.store.UpdatePaperTradeClose(tr)

		if tr.Outcome == "WIN" {
			t.metrics.Wins++
		} else if tr.Outcome == "LOSS" {
			t.metrics.Losses++
		} else if tr.Outcome == "EXPIRED" {
			t.metrics.Expired++
		}

		t.metrics.TotalPnL += tr.PnL

		if t.metrics.TotalPnL < t.metrics.MaxDrawdown {
			t.metrics.MaxDrawdown = t.metrics.TotalPnL
		}

		// Save equity curve point
		baseEquity := 1000.0
		currentEquity := baseEquity + t.metrics.TotalPnL
		t.store.SaveEquityPoint(tr.ID, currentEquity, t.metrics.MaxDrawdown, time.Now())
	}

	t.metrics.UpdatedAt = time.Now()
	t.store.SavePaperMetrics(t.metrics)
}

func (t *Tracker) GetStats() Metrics {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.metrics
}

func (t *Tracker) GetOpen() []Trade {
	t.mu.Lock()
	defer t.mu.Unlock()
	var res []Trade
	for _, tr := range t.open {
		res = append(res, *tr)
	}
	return res
}
