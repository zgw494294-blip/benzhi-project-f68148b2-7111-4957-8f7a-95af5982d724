package analysis

import "benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/domain"

type Chronology struct {
	ID        string
	StartYear int
	Species   string
	Region    string
	Widths    []float64
}

func BuiltinChronology() Chronology {
	return Chronology{
		ID: "CN-NORTH-PINE-DEMO-001", StartYear: 1850,
		Species: "松属", Region: "中国北方历史木构参考序列",
		Widths: []float64{
			1.24, 1.38, 1.19, 1.55, 1.62, 1.41, 1.08, 0.94, 1.13, 1.47,
			1.71, 1.66, 1.32, 1.18, 0.88, 0.97, 1.21, 1.44, 1.39, 1.06,
			0.91, 1.02, 1.36, 1.58, 1.77, 1.52, 1.27, 0.99, 0.82, 1.11,
			1.49, 1.63, 1.42, 1.16, 0.93, 1.07, 1.31, 1.69, 1.82, 1.61,
			1.28, 1.03, 0.89, 0.96, 1.22, 1.53, 1.74, 1.48, 1.17, 0.86,
			0.78, 1.04, 1.37, 1.65, 1.56, 1.29, 1.12, 0.92, 1.08, 1.43,
			1.67, 1.51, 1.23, 1.01, 0.84, 1.16, 1.46, 1.72, 1.59, 1.34,
			1.09, 0.95, 1.14, 1.32, 1.57, 1.79, 1.63, 1.26, 1.04, 0.87,
			1.02, 1.27, 1.48, 1.68, 1.45, 1.21, 0.98, 0.81, 1.06, 1.35,
			1.62, 1.54, 1.31, 1.13, 0.96, 1.09, 1.41, 1.73, 1.66, 1.38,
			1.12, 0.90, 0.83, 1.19, 1.51, 1.70, 1.47, 1.25, 1.01, 0.93,
			1.17, 1.39, 1.60, 1.76, 1.55, 1.24, 1.07, 0.88, 1.10, 1.34,
		},
	}
}

func DefaultParameters(minimumOverlap int) domain.AnalysisParameters {
	return domain.AnalysisParameters{MinimumOverlap: minimumOverlap, MaxCandidates: 5, LowCorrelation: 0.45, OutlierZScore: 2.8}
}
