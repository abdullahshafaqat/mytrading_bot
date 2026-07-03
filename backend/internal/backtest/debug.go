package backtest

import (
	"fmt"

	"github.com/abdullahshafaqat/trading-bot/internal/strategy"
)

type DebugStats struct {
	SignalsEvaluated int
	CrossRejected    int
	RSIRejected      int
	VolumeRejected   int
	ConfirmRejected  int
	AcceptedSignals  int
}

func (d *DebugStats) Record(result strategy.Result, filterAccepted bool) {
	d.SignalsEvaluated++

	switch result.Reason {
	case "No active cross event":
		d.CrossRejected++
	case "Volume too low":
		d.VolumeRejected++
	case "RSI not oversold", "RSI not overbought":
		d.RSIRejected++
	case "4h trend not bullish", "4h trend not bearish":
		d.ConfirmRejected++
	}

	if filterAccepted {
		d.AcceptedSignals++
	}
}

func (d DebugStats) Print() {
	fmt.Println("────────────")
	fmt.Printf("Cross: %d\n", d.CrossRejected)
	fmt.Printf("RSI Reject: %d\n", d.RSIRejected)
	fmt.Printf("Volume Reject: %d\n", d.VolumeRejected)
	fmt.Printf("4H Reject: %d\n", d.ConfirmRejected)
	fmt.Printf("Accepted: %d\n", d.AcceptedSignals)
	fmt.Println("────────────")
}
