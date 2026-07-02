package filters

import (
	"math"
	"testing"
)

func TestCosine(t *testing.T) {
	if got := cosine([]float32{1, 0}, []float32{1, 0}); math.Abs(got-1) > 1e-6 {
		t.Fatalf("identical vectors cosine=%v", got)
	}
	if got := cosine([]float32{1, 0}, []float32{0, 1}); math.Abs(got) > 1e-6 {
		t.Fatalf("orthogonal vectors cosine=%v", got)
	}
	if got := cosine([]float32{1, 0}, nil); got != 0 {
		t.Fatalf("mismatched length cosine=%v", got)
	}
}
