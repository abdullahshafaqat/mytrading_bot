package backtest

import (
	"fmt"
	"math"
	"time"

	"github.com/abdullahshafaqat/trading-bot/internal/strategy"
)

type TradeOutcome struct {
	Signal    Result
	Outcome   string
	ExitPrice float64
	ExitTime  time.Time
	PnL       float64
}

type GradeStats struct {
	TotalGraded  int
	Wins         int
	Losses       int
	Expired      int
	WinRate      float64
	ProfitFactor float64
	MaxDrawdown  float64
	TotalPnL     float64
}

func Grade(signals []Result, candles []strategy.Candle, riskPerTrade float64, signalTTLHours int) (GradeStats, []TradeOutcome) {
	return GradeWithManagedExits(signals, candles, riskPerTrade, signalTTLHours, "")
}

func GradeWithManagedExits(signals []Result, candles []strategy.Candle, riskPerTrade float64, signalTTLHours int, exitMode string) (GradeStats, []TradeOutcome) {
	var outcomes []TradeOutcome

	for _, sig := range signals {
		if sig.Reason != "Signal accepted" {
			continue
		}
		outcomes = append(outcomes, gradeSignal(sig, candles, riskPerTrade, signalTTLHours, exitMode))
	}

	return buildGradeStats(outcomes), outcomes
}

func gradeSignal(sig Result, candles []strategy.Candle, riskPerTrade float64, signalTTLHours int, exitMode string) TradeOutcome {
	if signalTTLHours <= 0 {
		signalTTLHours = 4
	}
	expiry := sig.Time.Add(time.Duration(signalTTLHours) * time.Hour)

	slDist := math.Abs(sig.Entry - sig.SL)
	if slDist == 0 {
		return TradeOutcome{Signal: sig, Outcome: "EXPIRED", ExitTime: expiry, PnL: 0}
	}

	tpReward := riskPerTrade * (math.Abs(sig.TP-sig.Entry) / slDist)
	if exitMode == "" {
		for _, c := range candles {
			if !c.Time.After(sig.Time) {
				continue
			}

			if c.Time.After(expiry) {
				return TradeOutcome{
					Signal:   sig,
					Outcome:  "EXPIRED",
					ExitTime: expiry,
					PnL:      0,
				}
			}

			if sig.Signal == "BUY" {
				hitTP := c.High >= sig.TP
				hitSL := c.Low <= sig.SL

				if hitTP && hitSL {
					return TradeOutcome{
						Signal:    sig,
						Outcome:   "SL_HIT",
						ExitPrice: sig.SL,
						ExitTime:  c.Time,
						PnL:       -riskPerTrade,
					}
				}
				if hitSL {
					return TradeOutcome{
						Signal:    sig,
						Outcome:   "SL_HIT",
						ExitPrice: sig.SL,
						ExitTime:  c.Time,
						PnL:       -riskPerTrade,
					}
				}
				if hitTP {
					return TradeOutcome{
						Signal:    sig,
						Outcome:   "TP_HIT",
						ExitPrice: sig.TP,
						ExitTime:  c.Time,
						PnL:       tpReward,
					}
				}
			}

			if sig.Signal == "SELL" {
				hitTP := c.Low <= sig.TP
				hitSL := c.High >= sig.SL

				if hitTP && hitSL {
					return TradeOutcome{
						Signal:    sig,
						Outcome:   "SL_HIT",
						ExitPrice: sig.SL,
						ExitTime:  c.Time,
						PnL:       -riskPerTrade,
					}
				}
				if hitSL {
					return TradeOutcome{
						Signal:    sig,
						Outcome:   "SL_HIT",
						ExitPrice: sig.SL,
						ExitTime:  c.Time,
						PnL:       -riskPerTrade,
					}
				}
				if hitTP {
					return TradeOutcome{
						Signal:    sig,
						Outcome:   "TP_HIT",
						ExitPrice: sig.TP,
						ExitTime:  c.Time,
						PnL:       tpReward,
					}
				}
			}
		}

		return TradeOutcome{
			Signal:   sig,
			Outcome:  "EXPIRED",
			ExitTime: expiry,
			PnL:      0,
		}
	}

	entry := sig.Entry
	currentSL := sig.SL
	partialTaken := false
	realizedPnL := 0.0
	remaining := 1.0

	for _, c := range candles {
		if !c.Time.After(sig.Time) {
			continue
		}

		if c.Time.After(expiry) {
			if exitMode == "O" && partialTaken {
				return TradeOutcome{
					Signal:   sig,
					Outcome:  "EXPIRED",
					ExitTime: expiry,
					PnL:      realizedPnL,
				}
			}
			return TradeOutcome{
				Signal:   sig,
				Outcome:  "EXPIRED",
				ExitTime: expiry,
				PnL:      0,
			}
		}

		if sig.Signal == "BUY" {
			hitTP := c.High >= sig.TP
			hitSL := c.Low <= currentSL

			if hitTP && hitSL {
				if exitMode == "O" && partialTaken {
					return TradeOutcome{Signal: sig, Outcome: "SL_HIT", ExitPrice: currentSL, ExitTime: c.Time, PnL: realizedPnL}
				}
				return TradeOutcome{Signal: sig, Outcome: "SL_HIT", ExitPrice: currentSL, ExitTime: c.Time, PnL: -riskPerTrade}
			}
			if hitSL {
				if exitMode == "O" && partialTaken {
					return TradeOutcome{Signal: sig, Outcome: "SL_HIT", ExitPrice: currentSL, ExitTime: c.Time, PnL: realizedPnL}
				}
				return TradeOutcome{Signal: sig, Outcome: "SL_HIT", ExitPrice: currentSL, ExitTime: c.Time, PnL: -riskPerTrade}
			}
			if hitTP {
				if exitMode == "O" && partialTaken {
					return TradeOutcome{Signal: sig, Outcome: "TP_HIT", ExitPrice: sig.TP, ExitTime: c.Time, PnL: realizedPnL + (tpReward * remaining)}
				}
				return TradeOutcome{Signal: sig, Outcome: "TP_HIT", ExitPrice: sig.TP, ExitTime: c.Time, PnL: tpReward}
			}

			if exitMode == "N" {
				if !partialTaken && c.High >= entry+slDist {
					currentSL = entry
					partialTaken = true
				}
				if c.High >= entry+(2*slDist) {
					candidate := c.Close - sig.ATR
					if candidate > currentSL {
						currentSL = candidate
					}
				}
			}

			if exitMode == "O" && !partialTaken && c.High >= entry+slDist {
				realizedPnL += riskPerTrade * 0.5
				remaining = 0.5
				currentSL = entry
				partialTaken = true
			}
		}

		if sig.Signal == "SELL" {
			hitTP := c.Low <= sig.TP
			hitSL := c.High >= currentSL

			if hitTP && hitSL {
				if exitMode == "O" && partialTaken {
					return TradeOutcome{Signal: sig, Outcome: "SL_HIT", ExitPrice: currentSL, ExitTime: c.Time, PnL: realizedPnL}
				}
				return TradeOutcome{Signal: sig, Outcome: "SL_HIT", ExitPrice: currentSL, ExitTime: c.Time, PnL: -riskPerTrade}
			}
			if hitSL {
				if exitMode == "O" && partialTaken {
					return TradeOutcome{Signal: sig, Outcome: "SL_HIT", ExitPrice: currentSL, ExitTime: c.Time, PnL: realizedPnL}
				}
				return TradeOutcome{Signal: sig, Outcome: "SL_HIT", ExitPrice: currentSL, ExitTime: c.Time, PnL: -riskPerTrade}
			}
			if hitTP {
				if exitMode == "O" && partialTaken {
					return TradeOutcome{Signal: sig, Outcome: "TP_HIT", ExitPrice: sig.TP, ExitTime: c.Time, PnL: realizedPnL + (tpReward * remaining)}
				}
				return TradeOutcome{Signal: sig, Outcome: "TP_HIT", ExitPrice: sig.TP, ExitTime: c.Time, PnL: tpReward}
			}

			if exitMode == "N" {
				if !partialTaken && c.Low <= entry-slDist {
					currentSL = entry
					partialTaken = true
				}
				if c.Low <= entry-(2*slDist) {
					candidate := c.Close + sig.ATR
					if candidate < currentSL {
						currentSL = candidate
					}
				}
			}

			if exitMode == "O" && !partialTaken && c.Low <= entry-slDist {
				realizedPnL += riskPerTrade * 0.5
				remaining = 0.5
				currentSL = entry
				partialTaken = true
			}
		}
	}

	if exitMode == "O" && partialTaken {
		return TradeOutcome{
			Signal:   sig,
			Outcome:  "EXPIRED",
			ExitTime: expiry,
			PnL:      realizedPnL,
		}
	}

	return TradeOutcome{
		Signal:   sig,
		Outcome:  "EXPIRED",
		ExitTime: expiry,
		PnL:      0,
	}
}

func buildGradeStats(outcomes []TradeOutcome) GradeStats {
	stats := GradeStats{TotalGraded: len(outcomes)}

	grossProfit := 0.0
	grossLoss := 0.0
	equity := 0.0
	peak := 0.0
	maxDD := 0.0

	for _, o := range outcomes {
		stats.TotalPnL += o.PnL
		equity += o.PnL

		switch o.Outcome {
		case "TP_HIT":
			stats.Wins++
			grossProfit += o.PnL
		case "SL_HIT":
			stats.Losses++
			grossLoss += math.Abs(o.PnL)
		case "EXPIRED":
			stats.Expired++
		}

		if equity > peak {
			peak = equity
		}
		if peak > 0 {
			dd := (peak - equity) / peak * 100
			if dd < 0 {
				dd = 0
			}
			if dd > 100 {
				dd = 100
			}
			if dd > maxDD {
				maxDD = dd
			}
		}
	}

	if stats.TotalGraded > 0 {
		stats.WinRate = float64(stats.Wins) / float64(stats.TotalGraded) * 100
	}
	if grossLoss > 0 {
		stats.ProfitFactor = grossProfit / grossLoss
	} else if grossProfit > 0 {
		stats.ProfitFactor = 999
	}
	stats.MaxDrawdown = maxDD

	return stats
}

func FormatGradeStats(s GradeStats) string {
	pf := fmt.Sprintf("%.2f", s.ProfitFactor)
	if s.ProfitFactor >= 999 {
		pf = "∞"
	}
	return fmt.Sprintf(
		"Graded: %d | Wins: %d | Losses: %d | Expired: %d\nWinRate: %.1f%% | ProfitFactor: %s | Max Drawdown: %.1f%% | TotalPnL: %.2f",
		s.TotalGraded,
		s.Wins,
		s.Losses,
		s.Expired,
		s.WinRate,
		pf,
		s.MaxDrawdown,
		s.TotalPnL,
	)
}
