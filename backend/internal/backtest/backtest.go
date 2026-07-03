package backtest

import (
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/abdullahshafaqat/trading-bot/internal/indicator"
	"github.com/abdullahshafaqat/trading-bot/internal/strategy"
)

type Config struct {
	Symbol           string
	Interval         string
	EMAFast          int
	EMASlow          int
	RSIPeriod        int
	ATRPeriod        int
	VolumePeriod     int
	CrossWindow      int
	SignalTTLHours   int
	RSIOversold      float64
	RSIOverbought    float64
	ATRMultSL        float64
	ATRMultTP        float64
	SkipRSI          bool
	SkipVolume       bool
	Skip4h           bool
	RSIMode          string
	RSIBuyMin        float64
	RSIBuyMax        float64
	RSISellMin       float64
	RSISellMax       float64
	UseEMA50         bool
	UseATRFilter     bool
	MinATRPct        float64
	MaxATRPct        float64
	MinTrendStrength float64
	RiskPerTrade     float64
	StartingBalance  float64
}

type Stats struct {
	TotalCandles    int
	CrossCreated    int
	CrossTriggered  int
	CrossExpired    int
	RSIReject       int
	VolumeReject    int
	ATRReject       int
	Confirm4hReject int
	TrendReject     int
	BuyAccepted     int
	SellAccepted    int
	TrendTrades     int
	RangeTrades     int
	Accepted        int
	Wins            int
	Losses          int
}

type Result struct {
	Time     time.Time
	Signal   string
	Entry    float64
	SL       float64
	TP       float64
	ATR      float64
	EMA50    float64
	Outcome  string
	Reason   string
	RSI      float64
	EMAFast  float64
	EMASlow  float64
	Volume   float64
	VolumeMA float64
}

type CrossState struct {
	Direction string
	CreatedAt time.Time
	ExpiresAt time.Time
	Used      bool
}

func Run(candles []strategy.Candle, cfg Config) (Stats, []Result) {
	emaFast := indicator.NewEMA(cfg.EMAFast)
	emaSlow := indicator.NewEMA(cfg.EMASlow)
	ema50 := indicator.NewEMA(50)
	rsi := indicator.NewRSI(cfg.RSIPeriod)
	atr := indicator.NewATR(cfg.ATRPeriod)
	volMA := indicator.NewVolumeMA(cfg.VolumePeriod)

	var stats Stats
	var results []Result

	var currentCross *CrossState
	prevFast := 0.0
	prevSlow := 0.0

	for _, candle := range candles {
		stats.TotalCandles++

		emaFast.Add(candle.Close)
		emaSlow.Add(candle.Close)
		ema50.Add(candle.Close)
		rsiVal := rsi.Add(candle.Close)
		atr.Add(candle.High, candle.Low, candle.Close)
		volMAVal := volMA.Add(candle.Volume)

		if !emaFast.Ready() || !emaSlow.Ready() || !rsi.Ready() {
			prevFast = emaFast.Value()
			prevSlow = emaSlow.Value()
			continue
		}

		fast := emaFast.Value()
		slow := emaSlow.Value()
		ema50Val := ema50.Value()

		if prevFast != 0 && prevSlow != 0 {
			wasBull := prevFast > prevSlow
			isBull := fast > slow
			wasBear := prevFast < prevSlow
			isBear := fast < slow

			if !wasBull && isBull {
				currentCross = &CrossState{
					Direction: "BUY",
					CreatedAt: candle.Time,
					ExpiresAt: candle.Time.Add(
						time.Duration(cfg.CrossWindow) * 15 * time.Minute,
					),
					Used: false,
				}
				stats.CrossCreated++
			} else if !wasBear && isBear {
				currentCross = &CrossState{
					Direction: "SELL",
					CreatedAt: candle.Time,
					ExpiresAt: candle.Time.Add(
						time.Duration(cfg.CrossWindow) * 15 * time.Minute,
					),
					Used: false,
				}
				stats.CrossCreated++
			}
		}

		prevFast = fast
		prevSlow = slow

		if currentCross == nil {
			continue
		}

		if candle.Time.After(currentCross.ExpiresAt) {
			if !currentCross.Used {
				stats.CrossExpired++
			}
			currentCross = nil
			continue
		}

		if currentCross.Used {
			continue
		}

		if !cfg.SkipVolume && volMA.Ready() && candle.Volume < volMAVal {
			stats.VolumeReject++
			results = append(results, Result{
				Time:     candle.Time,
				Signal:   currentCross.Direction,
				Reason:   "Volume reject",
				RSI:      rsiVal,
				EMAFast:  fast,
				EMASlow:  slow,
				EMA50:    ema50Val,
				Volume:   candle.Volume,
				VolumeMA: volMAVal,
			})
			continue
		}

		if !cfg.SkipRSI {
			if currentCross.Direction == "BUY" {
				var buyOK bool
				switch cfg.RSIMode {
				case "momentum", "range":
					buyOK = rsiVal >= cfg.RSIBuyMin && rsiVal <= cfg.RSIBuyMax
				default:
					buyOK = rsiVal < cfg.RSIOversold
				}
				if !buyOK {
					stats.RSIReject++
					results = append(results, Result{
						Time:     candle.Time,
						Signal:   "BUY",
						Reason:   fmt.Sprintf("RSI reject: %.2f", rsiVal),
						RSI:      rsiVal,
						EMAFast:  fast,
						EMASlow:  slow,
						EMA50:    ema50Val,
						Volume:   candle.Volume,
						VolumeMA: volMAVal,
					})
					continue
				}
			}

			if currentCross.Direction == "SELL" {
				var sellOK bool
				switch cfg.RSIMode {
				case "momentum", "range":
					sellOK = rsiVal >= cfg.RSISellMin && rsiVal <= cfg.RSISellMax
				default:
					sellOK = rsiVal > cfg.RSIOverbought
				}
				if !sellOK {
					stats.RSIReject++
					results = append(results, Result{
						Time:     candle.Time,
						Signal:   "SELL",
						Reason:   fmt.Sprintf("RSI reject: %.2f", rsiVal),
						RSI:      rsiVal,
						EMAFast:  fast,
						EMASlow:  slow,
						EMA50:    ema50Val,
						Volume:   candle.Volume,
						VolumeMA: volMAVal,
					})
					continue
				}
			}
		}

		if cfg.UseEMA50 && ema50.Ready() {
			if currentCross.Direction == "BUY" && candle.Close < ema50.Value() {
				stats.TrendReject++
				results = append(results, Result{
					Time:     candle.Time,
					Signal:   "BUY",
					Reason:   fmt.Sprintf("EMA50 reject: price %.2f below EMA50 %.2f", candle.Close, ema50.Value()),
					RSI:      rsiVal,
					EMAFast:  fast,
					EMASlow:  slow,
					EMA50:    ema50Val,
					Volume:   candle.Volume,
					VolumeMA: volMAVal,
				})
				continue
			}
			if currentCross.Direction == "SELL" && candle.Close > ema50.Value() {
				stats.TrendReject++
				results = append(results, Result{
					Time:     candle.Time,
					Signal:   "SELL",
					Reason:   fmt.Sprintf("EMA50 reject: price %.2f above EMA50 %.2f", candle.Close, ema50.Value()),
					RSI:      rsiVal,
					EMAFast:  fast,
					EMASlow:  slow,
					EMA50:    ema50Val,
					Volume:   candle.Volume,
					VolumeMA: volMAVal,
				})
				continue
			}
		}

		if cfg.UseATRFilter && atr.Ready() {
			atrPct := (atr.Value() / candle.Close) * 100
			if atrPct < cfg.MinATRPct {
				stats.ATRReject++
				results = append(results, Result{
					Time:     candle.Time,
					Signal:   currentCross.Direction,
					Reason:   "ATR too low",
					ATR:      atr.Value(),
					RSI:      rsiVal,
					EMAFast:  fast,
					EMASlow:  slow,
					EMA50:    ema50Val,
					Volume:   candle.Volume,
					VolumeMA: volMAVal,
				})
				continue
			}
			if cfg.MaxATRPct > 0 && atrPct > cfg.MaxATRPct {
				stats.ATRReject++
				results = append(results, Result{
					Time:     candle.Time,
					Signal:   currentCross.Direction,
					Reason:   "ATR too high",
					ATR:      atr.Value(),
					RSI:      rsiVal,
					EMAFast:  fast,
					EMASlow:  slow,
					EMA50:    ema50Val,
					Volume:   candle.Volume,
					VolumeMA: volMAVal,
				})
				continue
			}
		}

		if cfg.MinTrendStrength > 0 && atr.Ready() {
			atrVal := atr.Value()
			if atrVal > 0 {
				trendStrength := math.Abs(fast-slow) / atrVal
				if trendStrength < cfg.MinTrendStrength {
					stats.TrendReject++
					results = append(results, Result{
						Time:     candle.Time,
						Signal:   currentCross.Direction,
						Reason:   "Trend too weak",
						ATR:      atrVal,
						RSI:      rsiVal,
						EMAFast:  fast,
						EMASlow:  slow,
						EMA50:    ema50Val,
						Volume:   candle.Volume,
						VolumeMA: volMAVal,
					})
					continue
				}
			}
		}

		currentCross.Used = true
		stats.CrossTriggered++
		stats.Accepted++
		if currentCross.Direction == "BUY" {
			stats.BuyAccepted++
		} else if currentCross.Direction == "SELL" {
			stats.SellAccepted++
		}
		if ema50.Ready() {
			ema50Dist := math.Abs(candle.Close - ema50Val)
			if ema50Dist >= atr.Value() {
				stats.TrendTrades++
			} else {
				stats.RangeTrades++
			}
		}

		slDist := atr.Value() * cfg.ATRMultSL
		tpDist := atr.Value() * cfg.ATRMultTP

		sl := candle.Close - slDist
		tp := candle.Close + tpDist
		if currentCross.Direction == "SELL" {
			sl = candle.Close + slDist
			tp = candle.Close - tpDist
		}

		results = append(results, Result{
			Time:     candle.Time,
			Signal:   currentCross.Direction,
			Entry:    candle.Close,
			SL:       sl,
			TP:       tp,
			ATR:      atr.Value(),
			EMA50:    ema50Val,
			Outcome:  "PENDING",
			Reason:   "Signal accepted",
			RSI:      rsiVal,
			EMAFast:  fast,
			EMASlow:  slow,
			Volume:   candle.Volume,
			VolumeMA: volMAVal,
		})
	}

	return stats, results
}

func SaveCSV(results []Result, filename string) {
	f, err := os.Create(filename)
	if err != nil {
		log.Println("CSV create error:", err)
		return
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{
		"time", "signal", "entry", "sl", "tp",
		"atr", "outcome", "reason", "rsi", "ema_fast", "ema_slow",
		"volume", "volume_ma",
	})

	for _, r := range results {
		w.Write([]string{
			r.Time.Format("2006-01-02 15:04"),
			r.Signal,
			fmt.Sprintf("%.2f", r.Entry),
			fmt.Sprintf("%.2f", r.SL),
			fmt.Sprintf("%.2f", r.TP),
			fmt.Sprintf("%.2f", r.ATR),
			r.Outcome,
			r.Reason,
			fmt.Sprintf("%.2f", r.RSI),
			fmt.Sprintf("%.2f", r.EMAFast),
			fmt.Sprintf("%.2f", r.EMASlow),
			fmt.Sprintf("%.2f", r.Volume),
			fmt.Sprintf("%.2f", r.VolumeMA),
		})
	}

	log.Printf("[CSV] Saved %d rows to %s\n", len(results), filename)
}
