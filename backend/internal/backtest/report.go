package backtest

import (
	"fmt"
	"math"
)

func BuildReport(trades []Trade, startingBalance float64, minWinRate float64) Report {
	report := Report{
		TotalTrades:     len(trades),
		StartingBalance: startingBalance,
		FinalBalance:    startingBalance,
	}

	if len(trades) == 0 {
		report.Result = "FAIL"
		return report
	}

	grossProfit := 0.0
	grossLoss := 0.0
	rrSum := 0.0
	balance := startingBalance
	peak := startingBalance
	maxDD := 0.0

	for _, t := range trades {
		balance += t.PnL
		rrSum += t.RR

		if t.PnL >= 0 {
			report.Wins++
			grossProfit += t.PnL
		} else {
			report.Losses++
			grossLoss += math.Abs(t.PnL)
		}

		if balance > peak {
			peak = balance
		}
		if peak > 0 {
			dd := (peak - balance) / peak * 100
			if dd > maxDD {
				maxDD = dd
			}
		}
	}

	report.FinalBalance = balance
	report.WinRate = float64(report.Wins) / float64(report.TotalTrades) * 100
	report.LossRate = float64(report.Losses) / float64(report.TotalTrades) * 100
	report.MaxDrawdown = maxDD
	report.AvgRR = rrSum / float64(report.TotalTrades)

	if grossLoss == 0 {
		if grossProfit > 0 {
			report.ProfitFactor = 999
		}
	} else {
		report.ProfitFactor = grossProfit / grossLoss
	}

	report.Passed = report.WinRate >= minWinRate
	if report.Passed {
		report.Result = "PASS"
	} else {
		report.Result = "FAIL"
	}

	return report
}

func FormatReport(r Report) string {
	return fmt.Sprintf(
		"Trades: %d\nWinRate: %.1f%%\nDrawdown: %.1f%%\nProfitFactor: %.2f\nAvgRR: %.2f\nFinalBalance: %.2f\nResult: %s",
		r.TotalTrades,
		r.WinRate,
		r.MaxDrawdown,
		r.ProfitFactor,
		r.AvgRR,
		r.FinalBalance,
		r.Result,
	)
}
