package throughput

import "time"

// Meter is a basic throughput meter
type Meter struct {
	start time.Time
	bytes uint64
}

func New() *Meter {
	return &Meter{
		start: time.Now(),
		bytes: 0,
	}
}

func (m *Meter) Mark(s uint64) {
	m.bytes += s
}

func (m *Meter) Total() uint64 {
	return m.bytes
}

func (m *Meter) Per(duration time.Duration) float64 {
	passed := time.Since(m.start)
	return float64(m.bytes) * (float64(duration) / float64(passed))
}
