package main

import (
	"math"
	"testing"
)

func TestQuestionLevelStudentInterval(t *testing.T) {
	// A normal critical value, clipped bounds, or treating repetitions as
	// independent observations would fail these hand-calculated t intervals.
	for _, tc := range []struct {
		d               []float64
		mean, low, high *float64
	}{
		{nil, nil, nil, nil},
		{[]float64{1}, ptr(1), nil, nil},
		{[]float64{2, 2}, ptr(2), ptr(2), ptr(2)},
		{[]float64{0, 1, 2, 3, 4, 5, 6, 7}, ptr(3.5), ptr(1.4521753), ptr(5.5478247)},
		{[]float64{0, 1}, ptr(.5), ptr(-5.85310237), ptr(6.85310237)},
	} {
		m, l, h := questionInterval(tc.d)
		for i, got := range []*float64{m, l, h} {
			want := []*float64{tc.mean, tc.low, tc.high}[i]
			if (got == nil) != (want == nil) || (got != nil && math.Abs(*got-*want) > 1e-6) {
				t.Fatalf("%v: interval=%v,%v,%v", tc.d, m, l, h)
			}
		}
	}
}
func ptr(v float64) *float64 { return &v }
