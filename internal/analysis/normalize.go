package analysis

import "math"

func Normalize(widths []float64) []float64 {
	if len(widths) == 0 {
		return nil
	}
	logs := make([]float64, len(widths))
	var sum float64
	for i, width := range widths {
		logs[i] = math.Log(width)
		sum += logs[i]
	}
	mean := sum / float64(len(logs))
	var variance float64
	for _, value := range logs {
		delta := value - mean
		variance += delta * delta
	}
	stddev := math.Sqrt(variance / float64(len(logs)))
	if stddev == 0 {
		return make([]float64, len(logs))
	}
	for i := range logs {
		logs[i] = (logs[i] - mean) / stddev
	}
	return logs
}

func RingIndex(widths []float64) []float64 {
	if len(widths) == 0 {
		return nil
	}
	result := make([]float64, len(widths))
	for i, width := range widths {
		from := i - 2
		if from < 0 {
			from = 0
		}
		to := i + 3
		if to > len(widths) {
			to = len(widths)
		}
		var local float64
		for _, neighbor := range widths[from:to] {
			local += neighbor
		}
		local /= float64(to - from)
		if local != 0 {
			result[i] = width / local
		}
	}
	return Normalize(result)
}
