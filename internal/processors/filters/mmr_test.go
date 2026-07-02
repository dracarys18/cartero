package filters

import (
	"math"
	"testing"

	"cartero/internal/storage"
)

func cand(id string, score float64, vec []float32) scoredCandidate {
	return scoredCandidate{
		result: Result{Entry: storage.FeedEntry{ID: id}, Score: score},
		vector: vec,
	}
}

func ids(rs []Result) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.Entry.ID
	}
	return out
}

func TestMMRDiversifiesNearDuplicates(t *testing.T) {
	a := []float32{1, 0}
	aDup := []float32{1, 0.01}
	b := []float32{0, 1}

	cands := []scoredCandidate{
		cand("a1", 1.0, a),
		cand("a2", 0.98, aDup),
		cand("b1", 0.90, b),
	}

	got := ids(mmrRerank(cands, 0.5, 2))
	if got[0] != "a1" {
		t.Fatalf("expected highest-score a1 first, got %v", got)
	}
	if got[1] != "b1" {
		t.Fatalf("expected diverse b1 second over near-duplicate a2, got %v", got)
	}
}

func TestMMRLambdaOneIsPureScoreOrder(t *testing.T) {
	cands := []scoredCandidate{
		cand("a1", 1.0, []float32{1, 0}),
		cand("a2", 0.98, []float32{1, 0.01}),
		cand("b1", 0.90, []float32{0, 1}),
	}
	got := ids(mmrRerank(cands, 1, 3))
	want := []string{"a1", "a2", "b1"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("pure score order: want %v got %v", want, got)
		}
	}
}

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
