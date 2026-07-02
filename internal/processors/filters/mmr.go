package filters

import "math"

type scoredCandidate struct {
	result Result
	vector []float32
}

func mmrRerank(cands []scoredCandidate, lambda float64, limit int) []Result {
	if limit > len(cands) {
		limit = len(cands)
	}

	if lambda <= 0 || lambda >= 1 {
		out := make([]Result, 0, limit)
		for i := 0; i < limit; i++ {
			out = append(out, cands[i].result)
		}
		return out
	}

	selected := make([]scoredCandidate, 0, limit)
	remaining := make([]scoredCandidate, len(cands))
	copy(remaining, cands)

	for len(selected) < limit && len(remaining) > 0 {
		bestIdx := 0
		bestMMR := math.Inf(-1)
		for i, c := range remaining {
			maxSim := 0.0
			for _, sel := range selected {
				if sim := cosine(c.vector, sel.vector); sim > maxSim {
					maxSim = sim
				}
			}
			mmr := lambda*c.result.Score - (1-lambda)*maxSim
			if mmr > bestMMR {
				bestMMR = mmr
				bestIdx = i
			}
		}
		selected = append(selected, remaining[bestIdx])
		remaining = append(remaining[:bestIdx], remaining[bestIdx+1:]...)
	}

	out := make([]Result, 0, len(selected))
	for _, c := range selected {
		out = append(out, c.result)
	}
	return out
}

func cosine(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
