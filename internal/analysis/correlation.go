package analysis

import "math"

func Pearson(a, b []float64) float64 {
	if len(a) != len(b) || len(a) < 2 {
		return 0
	}
	var sumA, sumB float64
	for i := range a {
		sumA += a[i]
		sumB += b[i]
	}
	meanA, meanB := sumA/float64(len(a)), sumB/float64(len(b))
	var numerator, squareA, squareB float64
	for i := range a {
		da, db := a[i]-meanA, b[i]-meanB
		numerator += da * db
		squareA += da * da
		squareB += db * db
	}
	denominator := math.Sqrt(squareA * squareB)
	if denominator == 0 {
		return 0
	}
	return numerator / denominator
}

func roundScore(value float64) float64 { return math.Round(value*1000000) / 1000000 }
