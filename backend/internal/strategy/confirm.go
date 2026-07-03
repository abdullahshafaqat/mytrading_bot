package strategy

type Confirm4h struct {
	emaFast *float64
	emaSlow *float64
	ready   bool
}

func NewConfirm4h() *Confirm4h {
	return &Confirm4h{}
}

func (c *Confirm4h) Update(fast float64, slow float64) {
	c.emaFast = &fast
	c.emaSlow = &slow
	c.ready = true
}

func (c *Confirm4h) IsBullish() bool {
	if !c.ready {
		return false
	}
	return *c.emaFast > *c.emaSlow
}

func (c *Confirm4h) IsBearish() bool {
	if !c.ready {
		return false
	}
	return *c.emaFast < *c.emaSlow
}

func (c *Confirm4h) Ready() bool {
	return c.ready
}
