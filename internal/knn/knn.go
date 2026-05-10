package knn

import "container/heap"

const K = 5

type Candidate struct {
	Dist  float32
	Label uint8
}

// maxHeap keeps the K largest distances so we can prune early.
type maxHeap []Candidate

func (h maxHeap) Len() int            { return len(h) }
func (h maxHeap) Less(i, j int) bool  { return h[i].Dist > h[j].Dist }
func (h maxHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *maxHeap) Push(x any)         { *h = append(*h, x.(Candidate)) }
func (h *maxHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// Search runs exact KNN over a flat row-major float32 array.
// buf is pre-allocated storage for heap candidates (from sync.Pool).
// Returns the number of fraud labels among the K nearest neighbors.
func Search(vectors []float32, labels []uint8, query []float32, buf *[K]Candidate) int {
	dims := len(query)
	n := len(vectors) / dims

	h := maxHeap(buf[:0])

	for i := range n {
		row := vectors[i*dims : i*dims+dims]
		var d float32
		for j := range dims {
			diff := row[j] - query[j]
			d += diff * diff
		}

		if len(h) < K {
			heap.Push(&h, Candidate{d, labels[i]})
		} else if d < h[0].Dist {
			h[0] = Candidate{d, labels[i]}
			heap.Fix(&h, 0)
		}
	}

	var fraudCount int
	for _, c := range h {
		if c.Label == 1 {
			fraudCount++
		}
	}
	return fraudCount
}
