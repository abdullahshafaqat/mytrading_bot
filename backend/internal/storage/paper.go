package storage

import (
	"time"

	"github.com/abdullahshafaqat/trading-bot/internal/logger"
	"github.com/abdullahshafaqat/trading-bot/internal/paper"
)

func (db *DB) SavePaperTrade(t paper.Trade) error {
	query := `
		INSERT INTO paper_trades 
			(id, trade_id, symbol, side, entry, sl, tp, opened_at, outcome, pnl, market_regime, cross_id, entry_to_tp_minutes, entry_to_sl_minutes, ttl_minutes)
		VALUES 
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (id) DO UPDATE SET
			closed_at = EXCLUDED.closed_at,
			outcome = EXCLUDED.outcome,
			pnl = EXCLUDED.pnl,
			entry_to_tp_minutes = EXCLUDED.entry_to_tp_minutes,
			entry_to_sl_minutes = EXCLUDED.entry_to_sl_minutes,
			ttl_minutes = EXCLUDED.ttl_minutes
	`
	_, err := db.conn.Exec(query,
		t.ID, t.TradeID, t.Symbol, t.Side, t.Entry, t.SL, t.TP, t.OpenedAt, t.Outcome, t.PnL, t.MarketRegime, t.CrossID, nullFloat64(t.EntryToTPMinutes), nullFloat64(t.EntryToSLMinutes), t.TTLMinutes,
	)
	if err != nil {
		logger.DBf("SavePaperTrade error: %v", err)
	}
	return err
}

func (db *DB) UpdatePaperTradeClose(t paper.Trade) error {
	query := `
		UPDATE paper_trades
		SET closed_at = $1, outcome = $2, pnl = $3, entry_to_tp_minutes = $4, entry_to_sl_minutes = $5
		WHERE id = $6
	`
	_, err := db.conn.Exec(query,
		t.ClosedAt, t.Outcome, t.PnL, nullFloat64(t.EntryToTPMinutes), nullFloat64(t.EntryToSLMinutes), t.ID,
	)
	if err != nil {
		logger.DBf("UpdatePaperTradeClose error: %v", err)
	}
	return err
}

func (db *DB) LoadOpenPaperTrades() ([]paper.Trade, error) {
	query := `
		SELECT id, trade_id, symbol, side, entry, sl, tp, opened_at, outcome, pnl, market_regime, cross_id, ttl_minutes
		FROM paper_trades
		WHERE outcome = 'OPEN'
	`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []paper.Trade
	for rows.Next() {
		var t paper.Trade
		err := rows.Scan(
			&t.ID, &t.TradeID, &t.Symbol, &t.Side, &t.Entry, &t.SL, &t.TP, &t.OpenedAt, &t.Outcome, &t.PnL, &t.MarketRegime, &t.CrossID, &t.TTLMinutes,
		)
		if err == nil {
			trades = append(trades, t)
		}
	}
	return trades, nil
}

func (db *DB) GetPaperMetrics() (paper.Metrics, error) {
	query := `SELECT wins, losses, expired, total_pnl, max_drawdown, updated_at FROM paper_metrics WHERE id = 1`
	var m paper.Metrics
	err := db.conn.QueryRow(query).Scan(&m.Wins, &m.Losses, &m.Expired, &m.TotalPnL, &m.MaxDrawdown, &m.UpdatedAt)
	return m, err
}

func (db *DB) SavePaperMetrics(m paper.Metrics) error {
	query := `
		UPDATE paper_metrics
		SET wins = $1, losses = $2, expired = $3, total_pnl = $4, max_drawdown = $5, updated_at = $6
		WHERE id = 1
	`
	_, err := db.conn.Exec(query, m.Wins, m.Losses, m.Expired, m.TotalPnL, m.MaxDrawdown, m.UpdatedAt)
	if err != nil {
		logger.DBf("SavePaperMetrics error: %v", err)
	}
	return err
}

func (db *DB) GetPaperHistory(limit int) ([]paper.Trade, error) {
	query := `
		SELECT id, trade_id, symbol, side, entry, sl, tp, opened_at, closed_at, outcome, pnl, market_regime, cross_id
		FROM paper_trades
		WHERE outcome != 'OPEN'
		ORDER BY closed_at DESC
		LIMIT $1
	`
	rows, err := db.conn.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []paper.Trade
	for rows.Next() {
		var t paper.Trade
		err := rows.Scan(
			&t.ID, &t.TradeID, &t.Symbol, &t.Side, &t.Entry, &t.SL, &t.TP, &t.OpenedAt, &t.ClosedAt, &t.Outcome, &t.PnL, &t.MarketRegime, &t.CrossID,
		)
		if err == nil {
			trades = append(trades, t)
		}
	}
	return trades, nil
}

func (db *DB) GetPaperReport() (map[string]interface{}, error) {
	q7d := `
		SELECT 
			COALESCE(SUM(CASE WHEN outcome = 'WIN' THEN 1 ELSE 0 END)::FLOAT / NULLIF(COUNT(*), 0), 0) as win_rate,
			COALESCE(SUM(CASE WHEN pnl > 0 THEN pnl ELSE 0 END) / NULLIF(ABS(SUM(CASE WHEN pnl < 0 THEN pnl ELSE 0 END)), 0), SUM(CASE WHEN pnl > 0 THEN pnl ELSE 0 END)) as pf
		FROM paper_trades
		WHERE outcome IN ('WIN', 'LOSS') AND closed_at >= NOW() - INTERVAL '7 days'
	`
	var winRate7d, pf7d float64
	db.conn.QueryRow(q7d).Scan(&winRate7d, &pf7d)

	qSides := `
		SELECT side, 
			COALESCE(SUM(CASE WHEN outcome = 'WIN' THEN 1 ELSE 0 END)::FLOAT / NULLIF(COUNT(*), 0), 0)
		FROM paper_trades
		WHERE outcome IN ('WIN', 'LOSS')
		GROUP BY side
	`
	rows, _ := db.conn.Query(qSides)
	buyWinRate, sellWinRate := 0.0, 0.0
	for rows != nil && rows.Next() {
		var side string
		var wr float64
		rows.Scan(&side, &wr)
		if side == "BUY" {
			buyWinRate = wr
		} else if side == "SELL" {
			sellWinRate = wr
		}
	}
	if rows != nil {
		rows.Close()
	}

	qHold := `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (closed_at - opened_at))/60), 0)
		FROM paper_trades
		WHERE outcome != 'OPEN' AND closed_at IS NOT NULL
	`
	var avgHold float64
	db.conn.QueryRow(qHold).Scan(&avgHold)

	qOpen := `SELECT COUNT(*) FROM paper_trades WHERE outcome = 'OPEN'`
	var openTrades int
	db.conn.QueryRow(qOpen).Scan(&openTrades)

	return map[string]interface{}{
		"last_7d_win_rate": winRate7d,
		"last_7d_pf":       pf7d,
		"buy_win_rate":     buyWinRate,
		"sell_win_rate":    sellWinRate,
		"avg_hold_minutes": avgHold,
		"open_trades":      openTrades,
	}, nil
}

type EquityPoint struct {
	Timestamp time.Time
	TradeID   string
	Equity    float64
	Drawdown  float64
}

func (db *DB) SaveEquityPoint(tradeID string, equity, drawdown float64, ts time.Time) error {
	query := `INSERT INTO paper_equity (trade_id, equity, drawdown, ts) VALUES ($1, $2, $3, $4)`
	_, err := db.conn.Exec(query, tradeID, equity, drawdown, ts)
	return err
}

func (db *DB) GetPaperEquity() ([]map[string]interface{}, error) {
	query := `SELECT ts, trade_id, equity, drawdown FROM paper_equity ORDER BY ts ASC`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []map[string]interface{}
	for rows.Next() {
		var ts time.Time
		var tradeID string
		var eq, dd float64
		if err := rows.Scan(&ts, &tradeID, &eq, &dd); err == nil {
			points = append(points, map[string]interface{}{
				"timestamp": ts.Format("15:04"),
				"trade_id":  tradeID,
				"equity":    eq,
				"drawdown":  dd,
			})
		}
	}
	if points == nil {
		return []map[string]interface{}{}, nil
	}
	return points, nil
}
