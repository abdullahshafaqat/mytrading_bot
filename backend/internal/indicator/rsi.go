package indicator

type RSI struct {
	period    int
	gains     float64
	losses    float64
	prevClose float64
	count     int
	ready     bool
}

func NewRSI(period int) *RSI {
	return &RSI{period: period}
}

func (r *RSI) Add(price float64) float64 {
	if r.count == 0 {
		r.prevClose = price
		r.count++
		return 0
	}

	change := price - r.prevClose
	r.prevClose = price

	gain := 0.0
	loss := 0.0

	if change > 0 {
		gain = change
	} else {
		loss = -change
	}

	if r.count <= r.period {
		r.gains += gain
		r.losses += loss
		r.count++

		if r.count > r.period {
			r.gains /= float64(r.period)
			r.losses /= float64(r.period)
			r.ready = true
		}
		return 0
	}

	r.gains = (r.gains*float64(r.period-1) + gain) / float64(r.period)
	r.losses = (r.losses*float64(r.period-1) + loss) / float64(r.period)

	if r.losses == 0 {
		return 100
	}

	rs := r.gains / r.losses
	return 100 - (100 / (1 + rs))
}

func (r *RSI) Ready() bool {
	return r.ready
}
