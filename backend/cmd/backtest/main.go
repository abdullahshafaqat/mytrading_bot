package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/abdullahshafaqat/trading-bot/internal/backtest"
	"github.com/abdullahshafaqat/trading-bot/internal/config"
	"github.com/abdullahshafaqat/trading-bot/internal/market"
	"github.com/abdullahshafaqat/trading-bot/internal/strategy"
)

func main() {
	experiment := os.Getenv("BACKTEST_EXPERIMENT")
	if experiment == "" {
		experiment = "G"
	}

	candleLimit := 5000
	if v := os.Getenv("BACKTEST_CANDLES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			candleLimit = n
		}
	}

	startTime, hasStart := parseBacktestTime(os.Getenv("BACKTEST_START"), false)
	endTime, hasEnd := parseBacktestTime(os.Getenv("BACKTEST_END"), true)

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("BACKTEST — Experiment %s\n", experiment)
	fmt.Printf("Candles: %d\n", candleLimit)
	if hasStart || hasEnd {
		fmt.Printf("Range: %s -> %s\n", formatRangeTime(startTime, hasStart), formatRangeTime(endTime, hasEnd))
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	appCfg := config.Load()

	cfg := backtest.Config{
		Symbol:          appCfg.Symbol,
		Interval:        appCfg.PrimaryTimeframe,
		EMAFast:         appCfg.EMAFast,
		EMASlow:         appCfg.EMASlow,
		RSIPeriod:       appCfg.RSIPeriod,
		ATRPeriod:       appCfg.ATRPeriod,
		VolumePeriod:    appCfg.VolumePeriod,
		CrossWindow:     appCfg.CrossWindow,
		SignalTTLHours:  appCfg.SignalTTLHours,
		RSIOversold:     30.0,
		RSIOverbought:   70.0,
		ATRMultSL:       1.5,
		ATRMultTP:       2.5,
		RiskPerTrade:    appCfg.RiskPerTrade,
		StartingBalance: 1000.0,
	}

	switch experiment {
	case "D":
		cfg.SkipRSI = true
		cfg.SkipVolume = false
		cfg.Skip4h = true
		fmt.Println("Mode: RSI DISABLED")

	case "E":
		cfg.SkipRSI = false
		cfg.SkipVolume = false
		cfg.Skip4h = false
		cfg.RSIOversold = 45.0
		cfg.RSIOverbought = 55.0
		fmt.Println("Mode: RSI RELAXED 45/55")

	case "F":
		cfg.SkipRSI = false
		cfg.SkipVolume = true
		cfg.Skip4h = false
		cfg.RSIOversold = 45.0
		cfg.RSIOverbought = 55.0
		fmt.Println("Mode: RSI RELAXED + Volume DISABLED")

	case "G":
		cfg.SkipRSI = false
		cfg.SkipVolume = false
		cfg.Skip4h = false
		cfg.CrossWindow = 5
		cfg.RSIMode = "momentum"
		cfg.RSIBuyMin = 50.0
		cfg.RSIBuyMax = 70.0
		cfg.RSISellMin = 30.0
		cfg.RSISellMax = 50.0
		fmt.Println("Mode: Momentum RSI + CrossWindow 5")

	case "H":
		cfg.SkipRSI = false
		cfg.SkipVolume = true
		cfg.Skip4h = false
		cfg.CrossWindow = 5
		cfg.RSIMode = "momentum"
		cfg.RSIBuyMin = 50.0
		cfg.RSIBuyMax = 70.0
		cfg.RSISellMin = 30.0
		cfg.RSISellMax = 50.0
		fmt.Println("Mode: G settings — Volume DISABLED")

	case "I":
		cfg.SkipRSI = false
		cfg.SkipVolume = false
		cfg.Skip4h = false
		cfg.CrossWindow = 5
		cfg.RSIMode = "momentum"
		cfg.RSIBuyMin = 50.0
		cfg.RSIBuyMax = 70.0
		cfg.RSISellMin = 30.0
		cfg.RSISellMax = 50.0
		cfg.ATRMultSL = 1.0
		cfg.ATRMultTP = 3.0
		fmt.Println("Mode: ATR SL=1.0 TP=3.0")

	case "J":
		cfg.SkipRSI = false
		cfg.SkipVolume = false
		cfg.Skip4h = false
		cfg.CrossWindow = 5
		cfg.RSIMode = "momentum"
		cfg.RSIBuyMin = 50.0
		cfg.RSIBuyMax = 70.0
		cfg.RSISellMin = 30.0
		cfg.RSISellMax = 50.0
		cfg.ATRMultSL = 2.0
		cfg.ATRMultTP = 4.0
		fmt.Println("Mode: ATR SL=2.0 TP=4.0")

	case "K":
		cfg.SkipRSI = false
		cfg.SkipVolume = false
		cfg.Skip4h = false
		cfg.CrossWindow = 8
		cfg.RSIMode = "oversold"
		cfg.RSIOversold = 35.0
		cfg.RSIOverbought = 65.0
		cfg.ATRMultSL = 1.5
		cfg.ATRMultTP = 2.5
		fmt.Println("Mode: A — RSI oversold/overbought 35/65 + CrossWindow 8")

	case "L":
		cfg.SkipRSI = true
		cfg.SkipVolume = false
		cfg.Skip4h = false
		cfg.CrossWindow = 5
		cfg.UseEMA50 = true
		cfg.ATRMultSL = 1.5
		cfg.ATRMultTP = 2.5
		fmt.Println("Mode: B — EMA50 trend filter, no RSI")

	case "M":
		cfg.SkipRSI = false
		cfg.SkipVolume = false
		cfg.Skip4h = false
		cfg.CrossWindow = 5
		cfg.UseEMA50 = true
		cfg.RSIMode = "range"
		cfg.RSIBuyMin = 40.0
		cfg.RSIBuyMax = 55.0
		cfg.RSISellMin = 45.0
		cfg.RSISellMax = 60.0
		cfg.ATRMultSL = 1.5
		cfg.ATRMultTP = 2.5
		fmt.Println("Mode: M — Pullback in Trend (EMA50 + RSI range 40-55)")

	case "N":
		cfg.SkipRSI = false
		cfg.SkipVolume = false
		cfg.Skip4h = false
		cfg.CrossWindow = 5
		cfg.UseEMA50 = true
		cfg.RSIMode = "range"
		cfg.RSIBuyMin = 40.0
		cfg.RSIBuyMax = 55.0
		cfg.RSISellMin = 45.0
		cfg.RSISellMax = 60.0
		cfg.ATRMultSL = 1.5
		cfg.ATRMultTP = 2.5
		fmt.Println("Mode: N — Break-even + ATR trail")

	case "O":
		cfg.SkipRSI = false
		cfg.SkipVolume = false
		cfg.Skip4h = false
		cfg.CrossWindow = 5
		cfg.UseEMA50 = true
		cfg.RSIMode = "range"
		cfg.RSIBuyMin = 40.0
		cfg.RSIBuyMax = 55.0
		cfg.RSISellMin = 45.0
		cfg.RSISellMax = 60.0
		cfg.ATRMultSL = 1.5
		cfg.ATRMultTP = 2.5
		fmt.Println("Mode: O — Partial take profit + breakeven")

	case "P":
		cfg.SkipRSI = false
		cfg.SkipVolume = false
		cfg.Skip4h = false
		cfg.CrossWindow = 5
		cfg.UseEMA50 = true
		cfg.RSIMode = "range"
		cfg.RSIBuyMin = 40.0
		cfg.RSIBuyMax = 55.0
		cfg.RSISellMin = 45.0
		cfg.RSISellMax = 60.0
		cfg.ATRMultSL = 1.5
		cfg.ATRMultTP = 2.5
		cfg.UseATRFilter = true
		cfg.MinATRPct = 0.35
		fmt.Println("Mode: P — M + ATR volatility filter")

	case "Q":
		cfg.SkipRSI = false
		cfg.SkipVolume = false
		cfg.Skip4h = false
		cfg.CrossWindow = 5
		cfg.UseEMA50 = true
		cfg.RSIMode = "range"
		cfg.RSIBuyMin = 40.0
		cfg.RSIBuyMax = 55.0
		cfg.RSISellMin = 45.0
		cfg.RSISellMax = 60.0
		cfg.ATRMultSL = 1.5
		cfg.ATRMultTP = 2.5
		cfg.UseATRFilter = true
		cfg.MinATRPct = 0.35
		cfg.MaxATRPct = 0.70
		fmt.Println("Mode: Q — P + ATR ceiling")

	case "R":
		cfg = baseR(cfg)
		cfg.SignalTTLHours = 8
		fmt.Println("Mode: R — Q + TTL 8h")

	case "S":
		cfg = baseR(cfg)
		cfg.RiskPerTrade = 0.5
		fmt.Println("Mode: S — R + Half Risk")

	case "T":
		cfg = baseR(cfg)
		cfg.MaxATRPct = cfg.MaxATRPct * 1.15
		fmt.Println("Mode: T — R + ATR ceiling +15%")

	case "U":
		cfg = baseR(cfg)
		cfg.CrossWindow = 10
		fmt.Println("Mode: U — R + CrossWindow 10")

	case "X":
		cfg = baseR(cfg)
		cfg.MinTrendStrength = 0.25
		fmt.Println("Mode: X — R + Trend Strength Gate")

	case "A":
		cfg.SkipRSI = false
		cfg.SkipVolume = true
		cfg.Skip4h = true
		fmt.Println("Mode: Volume DISABLED")

	case "B":
		cfg.SkipRSI = false
		cfg.SkipVolume = false
		cfg.Skip4h = true
		fmt.Println("Mode: 4H DISABLED")

	case "C":
		cfg.SkipRSI = false
		cfg.SkipVolume = false
		cfg.Skip4h = false
		cfg.CrossWindow = 5
		fmt.Println("Mode: Cross window 5")
	}

	fmt.Println("Fetching historical data...")
	candles := fetchCandles(cfg.Symbol, cfg.Interval, candleLimit)
	if hasStart && hasEnd {
		candles = fetchCandlesWithRange(cfg.Symbol, cfg.Interval, candleLimit, startTime, endTime)
	}
	fmt.Printf("Loaded %d candles\n", len(candles))

	fmt.Println("Fetching 1m candles for outcome grading...")
	candles1m := fetchCandles(cfg.Symbol, "1m", candleLimit*15)
	if hasStart && hasEnd {
		candles1m = fetchCandlesWithRange(cfg.Symbol, "1m", candleLimit*15, startTime, endTime)
	}
	fmt.Printf("Loaded %d 1m candles\n", len(candles1m))

	stats, results := backtest.Run(candles, cfg)

	riskPerTrade := cfg.StartingBalance * (cfg.RiskPerTrade / 100)
	managedMode := ""
	if experiment == "N" || experiment == "O" {
		managedMode = experiment
	}
	gradeStats, outcomes := backtest.GradeWithManagedExits(results, candles1m, riskPerTrade, cfg.SignalTTLHours, managedMode)

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Total Candles:    %d\n", stats.TotalCandles)
	fmt.Printf("Cross Created:    %d\n", stats.CrossCreated)
	fmt.Printf("Cross Triggered:  %d\n", stats.CrossTriggered)
	fmt.Printf("Cross Expired:    %d\n", stats.CrossExpired)
	fmt.Printf("RSI Reject:       %d\n", stats.RSIReject)
	fmt.Printf("Volume Reject:    %d\n", stats.VolumeReject)
	fmt.Printf("ATR Reject:       %d\n", stats.ATRReject)
	fmt.Printf("Trend Reject:     %d\n", stats.TrendReject)
	fmt.Printf("Accepted:         %d\n", stats.Accepted)
	if stats.CrossCreated > 0 {
		fmt.Printf("AcceptanceRate:   %.1f%%\n", float64(stats.Accepted)/float64(stats.CrossCreated)*100)
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("OUTCOME GRADER")
	fmt.Println(backtest.FormatGradeStats(gradeStats))
	fmt.Printf("TP Hit: %d | SL Hit: %d | Expired: %d\n", gradeStats.Wins, gradeStats.Losses, gradeStats.Expired)
	if experiment == "P" {
		printExperimentPAnalysis(outcomes)
	} else if experiment == "R" {
		printRegimeReport(stats, results, outcomes)
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if err := os.MkdirAll("backtests", 0755); err != nil {
		log.Fatal(err)
	}
	filename := fmt.Sprintf("backtests/signals_%s.csv", experiment)
	backtest.SaveCSV(results, filename)
	fmt.Printf("Results saved to %s\n", filename)
}

func baseR(cfg backtest.Config) backtest.Config {
	cfg.SkipRSI = false
	cfg.SkipVolume = false
	cfg.Skip4h = false
	cfg.CrossWindow = 5
	cfg.SignalTTLHours = 8
	cfg.UseEMA50 = true
	cfg.RSIMode = "range"
	cfg.RSIBuyMin = 40.0
	cfg.RSIBuyMax = 55.0
	cfg.RSISellMin = 45.0
	cfg.RSISellMax = 60.0
	cfg.ATRMultSL = 1.5
	cfg.ATRMultTP = 2.5
	cfg.UseATRFilter = true
	cfg.MinATRPct = 0.35
	cfg.MaxATRPct = 0.70
	return cfg
}

func fetchCandles(symbol, interval string, limit int) []strategy.Candle {
	end := time.Now().UTC()
	start := end.Add(-time.Duration(limit) * intervalDuration(interval))

	raw, err := market.FetchHistoricalRange(symbol, interval, start, end)
	if err != nil {
		log.Fatal("Fetch failed:", err)
	}

	if len(raw) > limit {
		raw = raw[len(raw)-limit:]
	}

	candles := make([]strategy.Candle, 0, len(raw))
	for _, c := range raw {
		candles = append(candles, strategy.Candle{
			Time:   c.OpenTime,
			Open:   c.Open,
			High:   c.High,
			Low:    c.Low,
			Close:  c.Close,
			Volume: c.Volume,
		})
	}
	return candles
}

func fetchCandlesWithRange(symbol, interval string, limit int, startTime, endTime time.Time) []strategy.Candle {
	raw, err := market.FetchHistoricalRange(symbol, interval, startTime, endTime)
	if err != nil {
		log.Fatal("Fetch failed:", err)
	}

	if len(raw) > limit && limit > 0 {
		raw = raw[len(raw)-limit:]
	}

	candles := make([]strategy.Candle, 0, len(raw))
	for _, c := range raw {
		candles = append(candles, strategy.Candle{
			Time:   c.OpenTime,
			Open:   c.Open,
			High:   c.High,
			Low:    c.Low,
			Close:  c.Close,
			Volume: c.Volume,
		})
	}
	return candles
}

func parseBacktestTime(value string, endOfDay bool) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}

	layouts := []string{time.RFC3339, "2006-01-02"}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err != nil {
			continue
		}
		if layout == "2006-01-02" {
			parsed = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
			if endOfDay {
				parsed = parsed.Add(24*time.Hour - time.Nanosecond)
			}
		}
		return parsed.UTC(), true
	}

	log.Fatalf("invalid backtest time %q; use YYYY-MM-DD or RFC3339", value)
	return time.Time{}, false
}

func formatRangeTime(value time.Time, ok bool) string {
	if !ok {
		return "-"
	}
	return value.Format("2006-01-02 15:04:05 UTC")
}

func printRegimeReport(stats backtest.Stats, results []backtest.Result, outcomes []backtest.TradeOutcome) {
	buyWins := 0
	buyLosses := 0
	buyExpired := 0
	sellWins := 0
	sellLosses := 0
	sellExpired := 0
	winnerATR := make([]float64, 0)
	loserATR := make([]float64, 0)
	acceptedEMA50Dist := make([]float64, 0)

	for _, outcome := range outcomes {
		side := outcome.Signal.Signal
		switch outcome.Outcome {
		case "TP_HIT":
			if side == "BUY" {
				buyWins++
			} else if side == "SELL" {
				sellWins++
			}
			winnerATR = append(winnerATR, outcome.Signal.ATR)
		case "SL_HIT":
			if side == "BUY" {
				buyLosses++
			} else if side == "SELL" {
				sellLosses++
			}
			loserATR = append(loserATR, outcome.Signal.ATR)
		case "EXPIRED":
			if side == "BUY" {
				buyExpired++
			} else if side == "SELL" {
				sellExpired++
			}
		}
	}

	for _, result := range results {
		if result.Reason != "Signal accepted" {
			continue
		}
		acceptedEMA50Dist = append(acceptedEMA50Dist, absFloat(result.Entry-result.EMA50))
	}

	avg := func(values []float64) float64 {
		if len(values) == 0 {
			return 0
		}
		total := 0.0
		for _, value := range values {
			total += value
		}
		return total / float64(len(values))
	}

	buyAccepted := stats.BuyAccepted
	sellAccepted := stats.SellAccepted
	buyWinRate := 0.0
	sellWinRate := 0.0
	if buyAccepted > 0 {
		buyWinRate = float64(buyWins) / float64(buyAccepted) * 100
	}
	if sellAccepted > 0 {
		sellWinRate = float64(sellWins) / float64(sellAccepted) * 100
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━REGIME REPORT━━━━━━━━━━━━━━━━━━")
	fmt.Printf("BUY Accepted: %d | SELL Accepted: %d\n", buyAccepted, sellAccepted)
	fmt.Printf("BUY WinRate: %.1f%% | SELL WinRate: %.1f%%\n", buyWinRate, sellWinRate)
	fmt.Printf("Trend Trades: %d | Range Trades: %d\n", stats.TrendTrades, stats.RangeTrades)
	fmt.Printf("Winner ATR: %.4f | Loser ATR: %.4f\n", avg(winnerATR), avg(loserATR))
	fmt.Printf("Avg EMA50 Distance: %.4f\n", avg(acceptedEMA50Dist))
	fmt.Printf("Expired BUY: %d | Expired SELL: %d\n", buyExpired, sellExpired)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func intervalDuration(interval string) time.Duration {
	switch interval {
	case "1m":
		return time.Minute
	case "15m":
		return 15 * time.Minute
	case "4h":
		return 4 * time.Hour
	default:
		return 15 * time.Minute
	}
}

func printExperimentPAnalysis(outcomes []backtest.TradeOutcome) {
	buyWins := 0
	buyLosses := 0
	sellWins := 0
	sellLosses := 0
	expiredBuy := 0
	expiredSell := 0

	type hourCount struct {
		Hour  int
		Count int
	}

	hourLosses := map[int]int{}
	var winnerAtrs []float64
	var loserAtrs []float64

	for _, outcome := range outcomes {
		side := outcome.Signal.Signal
		hour := outcome.Signal.Time.UTC().Hour()

		switch outcome.Outcome {
		case "TP_HIT":
			if side == "BUY" {
				buyWins++
			} else if side == "SELL" {
				sellWins++
			}
			winnerAtrs = append(winnerAtrs, outcome.Signal.ATR)
		case "SL_HIT":
			if side == "BUY" {
				buyLosses++
			} else if side == "SELL" {
				sellLosses++
			}
			hourLosses[hour]++
			loserAtrs = append(loserAtrs, outcome.Signal.ATR)
		case "EXPIRED":
			if side == "BUY" {
				expiredBuy++
			} else if side == "SELL" {
				expiredSell++
			}
		}
	}

	hours := make([]hourCount, 0, len(hourLosses))
	for hour, count := range hourLosses {
		hours = append(hours, hourCount{Hour: hour, Count: count})
	}
	sort.Slice(hours, func(i, j int) bool {
		if hours[i].Count == hours[j].Count {
			return hours[i].Hour < hours[j].Hour
		}
		return hours[i].Count > hours[j].Count
	})

	avg := func(values []float64) float64 {
		if len(values) == 0 {
			return 0
		}
		total := 0.0
		for _, value := range values {
			total += value
		}
		return total / float64(len(values))
	}

	topHours := ""
	limit := 3
	if len(hours) < limit {
		limit = len(hours)
	}
	for i := 0; i < limit; i++ {
		if i > 0 {
			topHours += ", "
		}
		topHours += fmt.Sprintf("%02d:00(%d)", hours[i].Hour, hours[i].Count)
	}
	if topHours == "" {
		topHours = "none"
	}

	fmt.Println("P ANALYSIS")
	fmt.Printf("BUY wins: %d | BUY losses: %d | BUY expiries: %d\n", buyWins, buyLosses, expiredBuy)
	fmt.Printf("SELL wins: %d | SELL losses: %d | SELL expiries: %d\n", sellWins, sellLosses, expiredSell)
	fmt.Printf("Top losing hours: %s\n", topHours)
	fmt.Printf("Average ATR winners: %.4f\n", avg(winnerAtrs))
	fmt.Printf("Average ATR losers: %.4f\n", avg(loserAtrs))
}
