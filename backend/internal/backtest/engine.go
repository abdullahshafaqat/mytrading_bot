package backtest

import (
	"fmt"
	"time"

	"github.com/abdullahshafaqat/trading-bot/internal/config"
	"github.com/abdullahshafaqat/trading-bot/internal/indicator"
	"github.com/abdullahshafaqat/trading-bot/internal/market"
	"github.com/abdullahshafaqat/trading-bot/internal/risk"
	"github.com/abdullahshafaqat/trading-bot/internal/signal"
	"github.com/abdullahshafaqat/trading-bot/internal/strategy"
)

type Engine struct {
	cfg    EngineConfig
	appCfg config.Config
}

func NewEngine(cfg EngineConfig, appCfg *config.Config) *Engine {
	return &Engine{cfg: cfg, appCfg: *appCfg}
}

func (e *Engine) Run() (EngineResult, error) {
	warmupStart := e.cfg.Start.Add(-time.Duration(e.cfg.WarmupCandles) * 15 * time.Minute)

	candles15m, err := LoadCandles(e.cfg.Symbol, e.cfg.SignalTF, warmupStart, e.cfg.End)
	if err != nil {
		return EngineResult{}, err
	}

	candles1m, err := LoadCandles(e.cfg.Symbol, e.cfg.ReplayTF, e.cfg.Start, e.cfg.End)
	if err != nil {
		return EngineResult{}, err
	}

	warmup4h := e.cfg.WarmupCandles / 16
	if warmup4h < 50 {
		warmup4h = 50
	}
	warmup4hStart := e.cfg.Start.Add(-time.Duration(warmup4h) * 4 * time.Hour)
	candles4h, err := LoadCandles(e.cfg.Symbol, e.cfg.ConfirmTF, warmup4hStart, e.cfg.End)
	if err != nil {
		return EngineResult{}, err
	}

	emaFast := indicator.NewEMA(e.appCfg.EMAFast)
	emaSlow := indicator.NewEMA(e.appCfg.EMASlow)
	rsi := indicator.NewRSI(e.appCfg.RSIPeriod)
	atr := indicator.NewATR(e.appCfg.ATRPeriod)
	volMA := indicator.NewVolumeMA(e.appCfg.VolumePeriod)

	ema4hFast := indicator.NewEMA(e.appCfg.EMAFast)
	ema4hSlow := indicator.NewEMA(e.appCfg.EMASlow)
	confirm4h := strategy.NewConfirm4h()

	warmup15m := filterBefore(candles15m, e.cfg.Start)
	for _, c := range warmup15m {
		emaFast.Add(c.Close)
		emaSlow.Add(c.Close)
		rsi.Add(c.Close)
		atr.Add(c.High, c.Low, c.Close)
		volMA.Add(c.Volume)
	}

	warmup4hCandles := filterBefore(candles4h, e.cfg.Start)
	for _, c := range warmup4hCandles {
		ema4hFast.Add(c.Close)
		ema4hSlow.Add(c.Close)
	}
	if ema4hFast.Ready() && ema4hSlow.Ready() {
		confirm4h.Update(ema4hFast.Value(), ema4hSlow.Value())
	}

	strCfg := strategy.DefaultConfig()
	crossWindow := e.appCfg.CrossWindow
	applyExperiment(&strCfg, &crossWindow, e.cfg.Experiment)

	strEngine := strategy.NewEngine(strCfg, crossWindow)
	sigFilter := signal.NewFilter(crossWindow, e.appCfg.SignalTTLHours*60)
	riskMgr := risk.NewManager(risk.Config{
		RiskPerTrade:       e.appCfg.RiskPerTrade,
		DailyLossLimit:     e.appCfg.DailyLossLimit,
		MaxConsecutiveLoss: 3,
	}, e.cfg.StartingBalance)

	signalCandles := filterFrom(candles15m, e.cfg.Start)
	var trades []Trade
	var debug DebugStats
	var signalRows []SignalRow
	idx4h := len(warmup4hCandles)

	for _, candle := range signalCandles {
		for idx4h < len(candles4h) && !candles4h[idx4h].OpenTime.After(candle.OpenTime) {
			c4h := candles4h[idx4h]
			ema4hFast.Add(c4h.Close)
			ema4hSlow.Add(c4h.Close)
			if ema4hFast.Ready() && ema4hSlow.Ready() {
				confirm4h.Update(ema4hFast.Value(), ema4hSlow.Value())
			}
			idx4h++
		}

		emaFast.Add(candle.Close)
		emaSlow.Add(candle.Close)
		rsiVal := rsi.Add(candle.Close)
		atr.Add(candle.High, candle.Low, candle.Close)
		volMAVal := volMA.Add(candle.Volume)

		if !emaFast.Ready() || !emaSlow.Ready() || !rsi.Ready() || !volMA.Ready() {
			continue
		}
		if !confirm4h.Ready() {
			continue
		}

		result := strEngine.Analyze(
			emaFast.Value(),
			emaSlow.Value(),
			rsiVal,
			atr.Value(),
			candle.Volume,
			volMAVal,
			candle.Close,
			confirm4h.IsBullish(),
			candle.OpenTime,
			e.cfg.Symbol,
		)

		sig := sigFilter.Process(result, e.cfg.Symbol, candle.OpenTime, confirm4h.IsBullish())
		filterAccepted := sig != nil
		debug.Record(result, filterAccepted)
		signalRows = append(signalRows, rowFromResult(candle.OpenTime, result, confirm4h.IsBullish()))

		if sig == nil {
			continue
		}

		canTrade, _ := riskMgr.CanTrade()
		if !canTrade {
			continue
		}

		size, err := riskMgr.PositionSize(sig.Entry, sig.StopLoss)
		if err != nil {
			continue
		}

		expiresAt := candle.OpenTime.Add(time.Duration(e.appCfg.SignalTTLHours) * time.Hour)
		replay := filterAfter(candles1m, candle.OpenTime)

		trade, ok := simulateTrade(sig.Side, sig.Entry, sig.StopLoss, sig.TakeProfit, size, candle.OpenTime, expiresAt, replay)
		if !ok {
			continue
		}

		trades = append(trades, trade)
		riskMgr.AdjustBalance(trade.PnL)
		if trade.PnL >= 0 {
			riskMgr.RecordWin()
		} else {
			riskMgr.RecordLoss(mathAbs(trade.PnL))
		}
	}

	report := BuildReport(trades, e.cfg.StartingBalance, e.cfg.MinWinRate)
	exportPath, err := ExportSignalsCSV(signalRows, e.cfg.Experiment)
	if err != nil {
		return EngineResult{}, err
	}

	return EngineResult{
		Trades:     trades,
		Report:     report,
		Debug:      debug,
		ExportPath: exportPath,
	}, nil
}

func applyExperiment(strCfg *strategy.Config, crossWindow *int, experiment string) {
	switch experiment {
	case "A":
		strCfg.VolumeFilter = false
	case "B":
		strCfg.Confirm4h = false
	case "C":
		*crossWindow = 5
	}
}

func ParseExperiment(value string) string {
	switch value {
	case "A", "B", "C", "D":
		return value
	default:
		return ""
	}
}

func simulateTrade(
	side string,
	entry, stopLoss, takeProfit, size float64,
	entryTime, expiresAt time.Time,
	candles1m []market.Candle,
) (Trade, bool) {
	trade := Trade{
		Side:       side,
		Entry:      entry,
		StopLoss:   stopLoss,
		TakeProfit: takeProfit,
		Size:       size,
		EntryTime:  entryTime,
	}

	riskDist := mathAbs(entry - stopLoss)
	if riskDist == 0 {
		return Trade{}, false
	}

	for _, c := range candles1m {
		if c.OpenTime.After(expiresAt) {
			trade.Exit = c.Open
			trade.ExitTime = c.OpenTime
			trade.Outcome = "EXPIRED"
			break
		}

		exit, outcome, hit := checkExit(side, entry, stopLoss, takeProfit, c)
		if hit {
			trade.Exit = exit
			trade.ExitTime = c.OpenTime
			trade.Outcome = outcome
			break
		}
	}

	if trade.Outcome == "" {
		if len(candles1m) == 0 {
			return Trade{}, false
		}
		last := candles1m[len(candles1m)-1]
		trade.Exit = last.Close
		trade.ExitTime = last.OpenTime
		trade.Outcome = "EXPIRED"
	}

	if side == "BUY" {
		trade.PnL = (trade.Exit - entry) * size
	} else {
		trade.PnL = (entry - trade.Exit) * size
	}

	reward := mathAbs(trade.Exit - entry)
	trade.RR = reward / riskDist
	if trade.PnL < 0 {
		trade.RR = -trade.RR
	}

	return trade, true
}

func checkExit(side string, entry, stopLoss, takeProfit float64, c market.Candle) (float64, string, bool) {
	if side == "BUY" {
		if c.Low <= stopLoss {
			return stopLoss, "LOSS", true
		}
		if c.High >= takeProfit {
			return takeProfit, "WIN", true
		}
	} else {
		if c.High >= stopLoss {
			return stopLoss, "LOSS", true
		}
		if c.Low <= takeProfit {
			return takeProfit, "WIN", true
		}
	}
	return 0, "", false
}

func mathAbs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func DefaultConfig(appCfg *config.Config) EngineConfig {
	end := time.Now().UTC()
	start := end.AddDate(0, -3, 0)

	return EngineConfig{
		Symbol:          appCfg.Symbol,
		SignalTF:        appCfg.PrimaryTimeframe,
		ReplayTF:        "1m",
		ConfirmTF:       appCfg.ConfirmTimeframe,
		Start:           start,
		End:             end,
		StartingBalance: 1000,
		MinWinRate:      55,
		WarmupCandles:   50,
	}
}

func ParseTimeRange(startStr, endStr string) (time.Time, time.Time, error) {
	end := time.Now().UTC()
	start := end.AddDate(0, -3, 0)

	if startStr != "" {
		t, err := time.Parse("2006-01-02", startStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid BACKTEST_START: %w", err)
		}
		start = t.UTC()
	}

	if endStr != "" {
		t, err := time.Parse("2006-01-02", endStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid BACKTEST_END: %w", err)
		}
		end = t.UTC().Add(24*time.Hour - time.Second)
	}

	return start, end, nil
}
