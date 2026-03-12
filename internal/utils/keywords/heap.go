package keywords

import (
	"cmp"
	"container/heap"
	"slices"
)

type Score struct {
	Keyword string
	Score   float64
}

// TopK maintains the K highest scoring keywords using a min-heap.
// Add is O(log K), making the full pass O(N log K) instead of O(N log N).
type TopK struct {
	k    int
	data []Score
}

func NewTopK(k int) *TopK {
	return &TopK{k: k, data: make([]Score, 0, k)}
}

func (t *TopK) Add(keyword string, score float64) {
	if len(t.data) < t.k {
		heap.Push(t, Score{keyword, score})
	} else if score > t.data[0].Score {
		t.data[0] = Score{keyword, score}
		heap.Fix(t, 0)
	}
}

// Best returns the highest scoring result and whether one exists.
func (t *TopK) Best() (Score, bool) {
	top := t.Top(1)
	if len(top) == 0 {
		return Score{}, false
	}
	return top[0], true
}

// Top returns up to n results in descending score order.
func (t *TopK) Top(n int) []Score {
	result := make([]Score, len(t.data))
	copy(result, t.data)
	slices.SortFunc(result, func(a, b Score) int {
		return cmp.Compare(b.Score, a.Score)
	})
	if n < len(result) {
		return result[:n]
	}
	return result
}

// heap.Interface
func (t *TopK) Len() int           { return len(t.data) }
func (t *TopK) Less(i, j int) bool { return t.data[i].Score < t.data[j].Score }
func (t *TopK) Swap(i, j int)      { t.data[i], t.data[j] = t.data[j], t.data[i] }

func (t *TopK) Push(x any) {
	t.data = append(t.data, x.(Score))
}

func (t *TopK) Pop() any {
	old := t.data
	n := len(old)
	x := old[n-1]
	t.data = old[:n-1]
	return x
}
