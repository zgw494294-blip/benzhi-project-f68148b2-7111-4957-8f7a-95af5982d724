package persistence

type Health struct {
	Ready      bool  `json:"ready"`
	EventCount int64 `json:"eventCount"`
}

func (r *Repository) Health() Health {
	return Health{Ready: true, EventCount: r.Sequence()}
}
