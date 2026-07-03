package indicator

type VolumeMA struct {
	period int
	values []float64
	ready  bool
}

func NewVolumeMA(period int) *VolumeMA {
	return &VolumeMA{
		period: period,
		values: make([]float64, 0, period),
	}
}

func (v *VolumeMA) Add(volume float64) float64 {
	v.values = append(v.values, volume)

	if len(v.values) > v.period {
		v.values = v.values[1:]
	}

	if len(v.values) >= v.period {
		v.ready = true
	}

	return v.Value()
}

func (v *VolumeMA) Value() float64 {
	if len(v.values) == 0 {
		return 0
	}
	sum := 0.0
	for _, val := range v.values {
		sum += val
	}
	return sum / float64(len(v.values))
}

func (v *VolumeMA) Ready() bool {
	return v.ready
}
