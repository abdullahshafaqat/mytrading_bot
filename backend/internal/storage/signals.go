package storage

import (
	"time"

	"github.com/abdullahshafaqat/trading-bot/internal/logger"
)

type SignalRecord struct {
	ID               string
	TradeID          string
	CrossID          string
	SignalLatencyMs  int64
	MarketRegime     string
	EntryToTPMinutes *float64
	EntryToSLMinutes *float64
	Symbol           string
	Side             string
	Entry            float64
	StopLoss         float64
	TakeProfit       float64
	Reason           string
	CreatedAt        time.Time
	ExpiresAt        time.Time
	Outcome          string
}

func (db *DB) SaveSignal(s SignalRecord) error {
	query := `
		INSERT INTO signals 
			(id, trade_id, cross_id, signal_latency_ms, market_regime, entry_to_tp_minutes, entry_to_sl_minutes, symbol, side, entry, stop_loss, take_profit, reason, created_at, expires_at, outcome)
		VALUES 
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (id) DO NOTHING
	`

	_, err := db.conn.Exec(query,
		s.ID,
		s.TradeID,
		s.CrossID,
		s.SignalLatencyMs,
		s.MarketRegime,
		nullFloat64(s.EntryToTPMinutes),
		nullFloat64(s.EntryToSLMinutes),
		s.Symbol,
		s.Side,
		s.Entry,
		s.StopLoss,
		s.TakeProfit,
		s.Reason,
		s.CreatedAt,
		s.ExpiresAt,
		s.Outcome,
	)

	if err != nil {
		logger.DB("SaveSignal error:", err)
		return err
	}

	logger.DBf("Signal saved: %s %s @ %.2f (regime=%s, latency=%dms)\n", s.Side, s.Symbol, s.Entry, s.MarketRegime, s.SignalLatencyMs)
	return nil
}

func (db *DB) SaveCandle(symbol, interval string, o, h, l, c, v float64, ts time.Time) error {
	query := `
		INSERT INTO market_data (symbol, interval, open, high, low, close, volume, ts)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := db.conn.Exec(query, symbol, interval, o, h, l, c, v, ts)
	if err != nil {
		logger.DB("SaveCandle error:", err)
		return err
	}

	return nil
}

func (db *DB) GetRecentSignals(limit int) ([]SignalRecord, error) {
	query := `
		SELECT id, trade_id, cross_id, signal_latency_ms, market_regime, entry_to_tp_minutes, entry_to_sl_minutes, symbol, side, entry, stop_loss, take_profit, reason, created_at, expires_at, outcome
		FROM signals
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := db.conn.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var signals []SignalRecord
	for rows.Next() {
		var s SignalRecord
		err := rows.Scan(
			&s.ID, &s.TradeID, &s.CrossID, &s.SignalLatencyMs, &s.MarketRegime, &s.EntryToTPMinutes, &s.EntryToSLMinutes,
			&s.Symbol, &s.Side,
			&s.Entry, &s.StopLoss, &s.TakeProfit,
			&s.Reason, &s.CreatedAt, &s.ExpiresAt, &s.Outcome,
		)
		if err != nil {
			continue
		}
		signals = append(signals, s)
	}

	return signals, nil
}

func nullFloat64(value *float64) interface{} {
	if value == nil {
		return nil
	}
	return *value
}
