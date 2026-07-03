package signal

import "time"

type State string

const (
	StateWait       State = "WAIT"
	StateBuyActive  State = "BUY_ACTIVE"
	StateSellActive State = "SELL_ACTIVE"
)

type Signal struct {
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
}
