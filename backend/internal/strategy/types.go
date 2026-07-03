package strategy

import "time"

type Signal string

const (
	BUY  Signal = "BUY"
	SELL Signal = "SELL"
	HOLD Signal = "HOLD"
)

type CrossDirection string

const (
	CrossBullish CrossDirection = "BULLISH"
	CrossBearish CrossDirection = "BEARISH"
	CrossNone    CrossDirection = "NONE"
)

type CrossEvent struct {
	Symbol    string
	Direction CrossDirection
	CrossTime time.Time
	Used      bool
	ExpiresAt time.Time
}

type Result struct {
	Signal     Signal
	Reason     string
	EMAFast    float64
	EMASlow    float64
	RSI        float64
	ATR        float64
	Volume     float64
	VolumeMA   float64
	Entry      float64
	StopLoss   float64
	TakeProfit float64
}

type Candle struct {
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}
