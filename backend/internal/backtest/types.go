package backtest

import "time"

type EngineConfig struct {
	Symbol          string
	SignalTF        string
	ReplayTF        string
	ConfirmTF       string
	Start           time.Time
	End             time.Time
	StartingBalance float64
	MinWinRate      float64
	WarmupCandles   int
	Experiment      string
}

type EngineResult struct {
	Trades     []Trade
	Report     Report
	Debug      DebugStats
	ExportPath string
}

type Trade struct {
	Side       string
	Entry      float64
	Exit       float64
	StopLoss   float64
	TakeProfit float64
	Size       float64
	PnL        float64
	RR         float64
	Outcome    string
	EntryTime  time.Time
	ExitTime   time.Time
}

type Report struct {
	TotalTrades     int
	Wins            int
	Losses          int
	WinRate         float64
	LossRate        float64
	ProfitFactor    float64
	MaxDrawdown     float64
	AvgRR           float64
	StartingBalance float64
	FinalBalance    float64
	Passed          bool
	Result          string
}
