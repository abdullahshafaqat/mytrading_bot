package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/abdullahshafaqat/trading-bot/internal/config"
	"github.com/abdullahshafaqat/trading-bot/internal/indicator"
	"github.com/abdullahshafaqat/trading-bot/internal/logger"
	"github.com/abdullahshafaqat/trading-bot/internal/market"
	"github.com/abdullahshafaqat/trading-bot/internal/paper"
	"github.com/abdullahshafaqat/trading-bot/internal/risk"
	"github.com/abdullahshafaqat/trading-bot/internal/signal"
	"github.com/abdullahshafaqat/trading-bot/internal/storage"
	"github.com/abdullahshafaqat/trading-bot/internal/strategy"
)

func main() {
	var healthMu sync.RWMutex
	var ws15mOk, ws4hOk bool
	var lastSignalTime *time.Time
	startTime := time.Now()

	logger.Bot("BOT STARTED")
	cfg := config.Load()
	if err := logger.Init(os.Getenv("LOG_FILE")); err != nil {
		log.Fatalf("logger init failed: %v", err)
	}
	defer logger.Close()

	emaFast := indicator.NewEMA(cfg.EMAFast)
	emaSlow := indicator.NewEMA(cfg.EMASlow)
	rsi := indicator.NewRSI(cfg.RSIPeriod)
	atr := indicator.NewATR(cfg.ATRPeriod)
	volMA := indicator.NewVolumeMA(cfg.VolumePeriod)

	ema4hFast := indicator.NewEMA(cfg.EMAFast)
	ema4hSlow := indicator.NewEMA(cfg.EMASlow)
	confirm4h := strategy.NewConfirm4h()

	logger.Bot("Seeding 15m...")
	body15m, err := market.FetchHistorical(cfg.Symbol, cfg.PrimaryTimeframe, 50)
	if err != nil {
		log.Fatal(err)
	}
	candles15m, err := market.ParseHistorical(body15m)
	if err != nil {
		log.Fatal(err)
	}
	for _, c := range candles15m {
		emaFast.Add(c.Close)
		emaSlow.Add(c.Close)
		rsi.Add(c.Close)
		atr.Add(c.High, c.Low, c.Close)
		volMA.Add(c.Volume)
	}
	logger.Bot("Indicators Ready")

	logger.Bot("Seeding 4h...")
	body4h, err := market.FetchHistorical(cfg.Symbol, cfg.ConfirmTimeframe, 50)
	if err != nil {
		log.Fatal(err)
	}
	candles4h, err := market.ParseHistorical(body4h)
	if err != nil {
		log.Fatal(err)
	}
	for _, c := range candles4h {
		ema4hFast.Add(c.Close)
		ema4hSlow.Add(c.Close)
	}
	confirm4h.Update(ema4hFast.Value(), ema4hSlow.Value())
	logger.Bot("Confirmation Ready")

	strEngine := strategy.NewEngine(
		strategy.DefaultConfig(),
		cfg.CrossWindow,
	)

	sigFilter := signal.NewFilter(
		cfg.CrossWindow,
		cfg.SignalTTLHours*60,
	)

	riskMgr := risk.NewManager(risk.Config{
		RiskPerTrade:       cfg.RiskPerTrade,
		DailyLossLimit:     cfg.DailyLossLimit,
		MaxConsecutiveLoss: 3,
	}, 1000.0)

	db, err := storage.NewDB()
	if err != nil {
		log.Fatal("DB connection failed:", err)
	}
	defer db.Close()
	if err := db.Migrate(); err != nil {
		log.Fatal("DB migration failed:", err)
	}
	logger.DB("DB Connected")

	paperTracker := paper.NewTracker(db)
	paperExecutor := paper.NewExecutor(paperTracker)

	go func() {
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			healthMu.RLock()
			defer healthMu.RUnlock()

			indicatorsReady := emaFast.Ready() && emaSlow.Ready() && rsi.Ready() && volMA.Ready() && confirm4h.Ready()

			status := "ok"
			if db == nil {
				status = "degraded"
			} else if !ws15mOk || !ws4hOk {
				status = "starting"
			} else if !indicatorsReady {
				status = "warming"
			}

			resp := map[string]interface{}{
				"status":           status,
				"db":               db != nil,
				"ws_15m":           ws15mOk,
				"ws_4h":            ws4hOk,
				"indicators_ready": indicatorsReady,
				"uptime_seconds":   int(time.Since(startTime).Seconds()),
				"last_signal_time": lastSignalTime,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		})

		http.HandleFunc("/paper/stats", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(paperTracker.GetStats())
		})
		http.HandleFunc("/paper/open", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(paperTracker.GetOpen())
		})
		http.HandleFunc("/paper/history", func(w http.ResponseWriter, r *http.Request) {
			trades, _ := db.GetPaperHistory(50)
			if trades == nil {
				trades = []paper.Trade{}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(trades)
		})
		http.HandleFunc("/paper/report", func(w http.ResponseWriter, r *http.Request) {
			report, _ := db.GetPaperReport()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(report)
		})
		http.HandleFunc("/paper/export.csv", func(w http.ResponseWriter, r *http.Request) {
			trades, err := db.GetPaperHistory(10000)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/csv")
			w.Header().Set("Content-Disposition", `attachment; filename="paper_trades.csv"`)

			fmt.Fprintln(w, "opened_at,closed_at,side,entry,sl,tp,outcome,pnl,market_regime,hold_minutes")
			for _, t := range trades {
				hold := 0.0
				if t.ClosedAt != nil {
					hold = t.ClosedAt.Sub(t.OpenedAt).Minutes()
				}
				closedStr := ""
				if t.ClosedAt != nil {
					closedStr = t.ClosedAt.Format(time.RFC3339)
				}
				fmt.Fprintf(w, "%s,%s,%s,%.2f,%.2f,%.2f,%s,%.2f,%s,%.2f\n",
					t.OpenedAt.Format(time.RFC3339),
					closedStr,
					t.Side,
					t.Entry,
					t.SL,
					t.TP,
					t.Outcome,
					t.PnL,
					t.MarketRegime,
					hold,
				)
			}
		})

		logger.Bot("Health server listening on :8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			logger.Botf("Health server error: %v", err)
		}
	}()

	candle4hChan := make(chan market.Candle)
	go market.Stream(cfg.Symbol, cfg.ConfirmTimeframe, candle4hChan)

	candle15mChan := make(chan market.Candle)
	go market.Stream(cfg.Symbol, cfg.PrimaryTimeframe, candle15mChan)

	candle1mChan := make(chan market.Candle)
	go market.Stream(cfg.Symbol, "1m", candle1mChan)

	logger.WS("Streams Started")

	if emaFast.Ready() && emaSlow.Ready() {
		logger.Bot("[READY] EMA")
	}
	if rsi.Ready() {
		logger.Bot("[READY] RSI")
	}
	if atr.Ready() {
		logger.Bot("[READY] ATR")
	}
	if volMA.Ready() {
		logger.Bot("[READY] VOLMA")
	}
	if confirm4h.Ready() {
		logger.Bot("[READY] 4H_CONFIRM")
	}

	for {
		select {

		case candle := <-candle1mChan:
			paperTracker.Update(candle.High, candle.Low, candle.Close, candle.OpenTime)

		case candle := <-candle4hChan:
			healthMu.Lock()
			ws4hOk = true
			healthMu.Unlock()

			ema4hFast.Add(candle.Close)
			ema4hSlow.Add(candle.Close)
			confirm4h.Update(ema4hFast.Value(), ema4hSlow.Value())
			logger.Botf("[4H] %s | EMA9: %.2f | EMA21: %.2f | Bullish: %v\n",
				candle.OpenTime.Format("15:04"),
				ema4hFast.Value(),
				ema4hSlow.Value(),
				confirm4h.IsBullish(),
			)

		case candle := <-candle15mChan:
			healthMu.Lock()
			ws15mOk = true
			healthMu.Unlock()

			emaFast.Add(candle.Close)
			emaSlow.Add(candle.Close)
			rsiVal := rsi.Add(candle.Close)
			atr.Add(candle.High, candle.Low, candle.Close)
			volMAVal := volMA.Add(candle.Volume)

			db.SaveCandle(
				candle.Symbol,
				candle.Interval,
				candle.Open,
				candle.High,
				candle.Low,
				candle.Close,
				candle.Volume,
				candle.OpenTime,
			)

			if !emaFast.Ready() ||
				!emaSlow.Ready() ||
				!rsi.Ready() ||
				!volMA.Ready() {
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
				cfg.Symbol,
			)

			fmt.Printf(
				"[%s] %s | C: %.2f | EMA9: %.2f | EMA21: %.2f | RSI: %.2f | ATR: %.2f | Vol: %.2f/%.2f | 4h: %v | Reason: %s\n",
				result.Signal,
				candle.OpenTime.Format("15:04"),
				candle.Close,
				emaFast.Value(),
				emaSlow.Value(),
				rsiVal,
				atr.Value(),
				candle.Volume,
				volMAVal,
				confirm4h.IsBullish(),
				result.Reason,
			)

			sig := sigFilter.Process(result, cfg.Symbol, candle.OpenTime, confirm4h.IsBullish())
			if sig == nil {
				continue
			}

			now := time.Now()
			healthMu.Lock()
			lastSignalTime = &now
			healthMu.Unlock()

			canTrade, blockReason := riskMgr.CanTrade()
			size, sizeErr := riskMgr.PositionSize(sig.Entry, sig.StopLoss)

			fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			fmt.Printf("🚨 SIGNAL: %s %s\n", sig.Side, sig.Symbol)
			fmt.Printf("   Entry:   %.2f\n", sig.Entry)
			fmt.Printf("   SL:      %.2f\n", sig.StopLoss)
			fmt.Printf("   TP:      %.2f\n", sig.TakeProfit)
			fmt.Printf("   Reason:  %s\n", sig.Reason)
			fmt.Printf("   Expires: %s\n", sig.ExpiresAt.Format("15:04"))

			if canTrade && sizeErr == nil {
				fmt.Printf("   Size:    %.6f BTC\n", size)
				fmt.Printf("   Risk:    %.2f USDT (%.1f%%)\n",
					1000.0*(cfg.RiskPerTrade/100),
					cfg.RiskPerTrade,
				)
				db.SaveSignal(storage.SignalRecord{
					ID:               sig.ID,
					TradeID:          sig.TradeID,
					CrossID:          sig.CrossID,
					SignalLatencyMs:  sig.SignalLatencyMs,
					MarketRegime:     sig.MarketRegime,
					EntryToTPMinutes: sig.EntryToTPMinutes,
					EntryToSLMinutes: sig.EntryToSLMinutes,
					Symbol:           sig.Symbol,
					Side:             sig.Side,
					Entry:            sig.Entry,
					StopLoss:         sig.StopLoss,
					TakeProfit:       sig.TakeProfit,
					Reason:           sig.Reason,
					CreatedAt:        sig.CreatedAt,
					ExpiresAt:        sig.ExpiresAt,
					Outcome:          "PENDING",
				})

				if err := paperExecutor.Execute(sig); err != nil {
					fmt.Printf("   ⛔ Paper execution failed: %s\n", err.Error())
				}
			} else if sizeErr != nil {
				fmt.Printf("   ⛔ Risk block: %s\n", sizeErr.Error())
			} else {
				fmt.Printf("   ⛔ Risk block: %s\n", blockReason)
			}
			fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		}
	}
}
