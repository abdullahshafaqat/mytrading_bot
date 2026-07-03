package indicator

import "math"

type ATR struct {
	period    int
	value     float64
	prevClose float64
	count     int
	ready     bool
}

func NewATR(period int) *ATR {
	return &ATR{period: period}
}

func (a *ATR) Add(high, low, close float64) float64 {
	tr := high - low

	if a.count > 0 {
		tr = math.Max(tr, math.Abs(high-a.prevClose))
		tr = math.Max(tr, math.Abs(low-a.prevClose))
	}

	a.prevClose = close
	a.count++

	if a.count <= a.period {
		a.value += tr
		if a.count == a.period {
			a.value /= float64(a.period)
			a.ready = true
		}
		return a.value
	}

	a.value = (a.value*float64(a.period-1) + tr) / float64(a.period)
	return a.value
}

func (a *ATR) Ready() bool {
	return a.ready
}

func (a *ATR) Value() float64 {
	return a.value
}
