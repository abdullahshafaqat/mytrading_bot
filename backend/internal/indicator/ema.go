package indicator

type EMA struct {
	period     int
	multiplier float64
	value      float64
	ready      bool
	count      int
	sum        float64
}

func NewEMA(period int) *EMA {
	return &EMA{
		period:     period,
		multiplier: 2.0 / float64(period+1),
	}
}

func (e *EMA) Add(price float64) float64 {
	if !e.ready {
		e.count++
		e.sum += price
		if e.count >= e.period {
			e.value = e.sum / float64(e.period)
			e.ready = true
		}
		return e.value
	}

	e.value = (price-e.value)*e.multiplier + e.value
	return e.value
}

func (e *EMA) Ready() bool {
	return e.ready
}

func (e *EMA) Value() float64 {
	return e.value
}
