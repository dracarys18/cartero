package filters

import (
	"math"

	"github.com/viterin/vek/vek32"
)

func cosine(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	na := float64(vek32.Dot(a, a))
	nb := float64(vek32.Dot(b, b))
	if na == 0 || nb == 0 {
		return 0
	}
	return float64(vek32.Dot(a, b)) / math.Sqrt(na*nb)
}
