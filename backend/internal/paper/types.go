package paper

import "time"

type Trade struct {
	ID               string
	TradeID          string
	Symbol           string
	Side             string
	Entry            float64
	SL               float64
	TP               float64
	OpenedAt         time.Time
	ClosedAt         *time.Time
	Outcome          string // "OPEN", "WIN", "LOSS", "EXPIRED"
	PnL              float64
	MarketRegime     string
	CrossID          string
	EntryToTPMinutes *float64
	EntryToSLMinutes *float64
	TTLMinutes       int
}

type Metrics struct {
	Wins        int
	Losses      int
	Expired     int
	TotalPnL    float64
	MaxDrawdown float64
	UpdatedAt   time.Time
}
