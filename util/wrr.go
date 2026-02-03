package util

// Weighted round-robin picker.
// It returns names based on weights, without key affinity.

type WRR struct {
	order []string
	idx   int
}

func NewWRR(weights map[string]int) *WRR {
	order := make([]string, 0)
	for name, weight := range weights {
		if weight <= 0 {
			continue
		}
		for i := 0; i < weight; i++ {
			order = append(order, name)
		}
	}
	return &WRR{order: order, idx: 0}
}

func (w *WRR) Next() string {
	if w == nil || len(w.order) == 0 {
		return ""
	}
	name := w.order[w.idx%len(w.order)]
	w.idx = (w.idx + 1) % len(w.order)
	return name
}
