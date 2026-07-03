package indicator

type EMAResult struct {
	Fast float64
	Slow float64
}

type RSIResult struct {
	Value float64
}

type ATRResult struct {
	Value float64
}

type Result struct {
	EMA EMAResult
	RSI RSIResult
	ATR ATRResult
}
