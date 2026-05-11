package main

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"

	"rinha/internal/dataset"
	"rinha/internal/knn"
	"rinha/internal/vectorize"
)

var (
	ds      *dataset.Dataset
	mccRisk map[string]float64
	nc      *dataset.NormConstants
	ready   atomic.Bool
)

type bufPool struct {
	p sync.Pool
}

func newBufPool() *bufPool {
	return &bufPool{
		p: sync.Pool{
			New: func() any {
				v := make([]float32, 14)
				return &v
			},
		},
	}
}

func (b *bufPool) get() *[]float32 { return b.p.Get().(*[]float32) }
func (b *bufPool) put(v *[]float32) { b.p.Put(v) }

var pool = newBufPool()

type knnBuf [knn.K]knn.Candidate

var knnPool = sync.Pool{New: func() any { return new(knnBuf) }}

func handleFraudScore(w http.ResponseWriter, r *http.Request) {
	if !ready.Load() {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var tx vectorize.Transaction
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	vec := pool.get()
	vectorize.Vectorize(&tx, nc, mccRisk, *vec)

	buf := knnPool.Get().(*knnBuf)
	fraudCount := knn.Search(ds.Vectors, ds.Labels, *vec, (*[knn.K]knn.Candidate)(buf))
	knnPool.Put(buf)
	pool.put(vec)

	fraudScore := float64(fraudCount) / float64(knn.K)
	approved := fraudScore < 0.6

	w.Header().Set("Content-Type", "application/json")
	resp := `{"approved":` + strconv.FormatBool(approved) + `,"fraud_score":` + strconv.FormatFloat(fraudScore, 'f', -1, 64) + `}`
	w.Write([]byte(resp))
}

func handleReady(w http.ResponseWriter, r *http.Request) {
	if ready.Load() {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	mux := http.NewServeMux()
	mux.HandleFunc("/fraud-score", handleFraudScore)
	mux.HandleFunc("/ready", handleReady)

	go func() {
		log.Println("listening on :9999")
		if err := http.ListenAndServe(":9999", mux); err != nil {
			log.Fatal(err)
		}
	}()

	var err error
	ds, err = dataset.LoadReferences("resources/references.json.gz")
	if err != nil {
		log.Fatalf("load references: %v", err)
	}
	log.Printf("loaded %d reference vectors", ds.Count)

	mccRisk, err = dataset.LoadMCCRisk("resources/mcc_risk.json")
	if err != nil {
		log.Fatalf("load mcc_risk: %v", err)
	}

	nc, err = dataset.LoadNormConstants("resources/normalization.json")
	if err != nil {
		log.Fatalf("load normalization: %v", err)
	}

	ready.Store(true)
	log.Println("ready")

	select {}
}
