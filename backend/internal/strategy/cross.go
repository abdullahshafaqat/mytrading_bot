package strategy

import "time"

type CrossTracker struct {
	prevFast    float64
	prevSlow    float64
	crossWindow int
	current     *CrossEvent
}

func NewCrossTracker(crossWindow int) *CrossTracker {
	return &CrossTracker{
		crossWindow: crossWindow,
	}
}

func (ct *CrossTracker) Update(
	fast float64,
	slow float64,
	symbol string,
	candleTime time.Time,
) *CrossEvent {

	if ct.prevFast == 0 && ct.prevSlow == 0 {
		ct.prevFast = fast
		ct.prevSlow = slow
		return nil
	}

	wasBullish := ct.prevFast > ct.prevSlow
	isBullish := fast > slow

	wasBearish := ct.prevFast < ct.prevSlow
	isBearish := fast < slow

	ct.prevFast = fast
	ct.prevSlow = slow

	if !wasBullish && isBullish {
		ct.current = &CrossEvent{
			Symbol:    symbol,
			Direction: CrossBullish,
			CrossTime: candleTime,
			Used:      false,
			ExpiresAt: candleTime.Add(
				time.Duration(ct.crossWindow) * 15 * time.Minute,
			),
		}
		return ct.current
	}

	if !wasBearish && isBearish {
		ct.current = &CrossEvent{
			Symbol:    symbol,
			Direction: CrossBearish,
			CrossTime: candleTime,
			Used:      false,
			ExpiresAt: candleTime.Add(
				time.Duration(ct.crossWindow) * 15 * time.Minute,
			),
		}
		return ct.current
	}

	return nil
}

func (ct *CrossTracker) Current() *CrossEvent {
	return ct.current
}

func (ct *CrossTracker) MarkUsed() {
	if ct.current != nil {
		ct.current.Used = true
	}
}

func (ct *CrossTracker) IsValid(now time.Time) bool {
	if ct.current == nil {
		return false
	}
	if ct.current.Used {
		return false
	}
	if now.After(ct.current.ExpiresAt) {
		return false
	}
	return true
}
