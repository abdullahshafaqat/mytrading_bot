package strategy

import "time"

type Config struct {
	RSIOversold     float64
	RSIOverbought   float64
	ATRMultiplierSL float64
	ATRMultiplierTP float64
	VolumeFilter    bool
	Confirm4h       bool
}

func DefaultConfig() Config {
	return Config{
		RSIOversold:     30.0,
		RSIOverbought:   70.0,
		ATRMultiplierSL: 1.5,
		ATRMultiplierTP: 2.5,
		VolumeFilter:    true,
		Confirm4h:       true,
	}
}

type Engine struct {
	cfg          Config
	crossTracker *CrossTracker
}

func NewEngine(cfg Config, crossWindow int) *Engine {
	return &Engine{
		cfg:          cfg,
		crossTracker: NewCrossTracker(crossWindow),
	}
}

func (e *Engine) Analyze(
	fast float64,
	slow float64,
	rsi float64,
	atr float64,
	volume float64,
	volumeMA float64,
	close float64,
	confirm4hBullish bool,
	candleTime time.Time,
	symbol string,
) Result {

	result := Result{
		Signal:   HOLD,
		Reason:   "No condition met",
		EMAFast:  fast,
		EMASlow:  slow,
		RSI:      rsi,
		ATR:      atr,
		Volume:   volume,
		VolumeMA: volumeMA,
		Entry:    close,
	}

	e.crossTracker.Update(fast, slow, symbol, candleTime)

	if !e.crossTracker.IsValid(candleTime) {
		result.Reason = "No active cross event"
		return result
	}

	cross := e.crossTracker.Current()

	if e.cfg.VolumeFilter && volume < volumeMA {
		result.Reason = "Volume too low"
		return result
	}

	if cross.Direction == CrossBullish {

		if rsi >= e.cfg.RSIOversold {
			result.Reason = "RSI not oversold"
			return result
		}

		if e.cfg.Confirm4h && !confirm4hBullish {
			result.Reason = "4h trend not bullish"
			return result
		}

		e.crossTracker.MarkUsed()
		result.Signal = BUY
		result.Reason = "EMA cross bullish + RSI oversold + Volume ok"
		result.StopLoss = close - (atr * e.cfg.ATRMultiplierSL)
		result.TakeProfit = close + (atr * e.cfg.ATRMultiplierTP)
		return result
	}

	if cross.Direction == CrossBearish {

		if rsi <= e.cfg.RSIOverbought {
			result.Reason = "RSI not overbought"
			return result
		}

		if e.cfg.Confirm4h && confirm4hBullish {
			result.Reason = "4h trend not bearish"
			return result
		}

		e.crossTracker.MarkUsed()
		result.Signal = SELL
		result.Reason = "EMA cross bearish + RSI overbought + Volume ok"
		result.StopLoss = close + (atr * e.cfg.ATRMultiplierSL)
		result.TakeProfit = close - (atr * e.cfg.ATRMultiplierTP)
		return result
	}

	return result
}
