package dataset

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
)

type NormConstants struct {
	MaxAmount           float64 `json:"max_amount"`
	MaxInstallments     float64 `json:"max_installments"`
	AmountVsAvgRatio    float64 `json:"amount_vs_avg_ratio"`
	MaxMinutes          float64 `json:"max_minutes"`
	MaxKm               float64 `json:"max_km"`
	MaxTxCount24h       float64 `json:"max_tx_count_24h"`
	MaxMerchantAvgAmount float64 `json:"max_merchant_avg_amount"`
}

type Dataset struct {
	Vectors []float32
	Labels  []uint8
	Count   int
	Dims    int
}

func LoadReferences(path string) (*Dataset, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open references: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	type ref struct {
		Vector []float32 `json:"vector"`
		Label  string    `json:"label"`
	}

	var refs []ref
	if err := json.NewDecoder(gz).Decode(&refs); err != nil {
		return nil, fmt.Errorf("decode references: %w", err)
	}

	n := len(refs)
	const dims = 14
	vectors := make([]float32, n*dims)
	labels := make([]uint8, n)

	for i, r := range refs {
		copy(vectors[i*dims:], r.Vector)
		if r.Label == "fraud" {
			labels[i] = 1
		}
	}

	return &Dataset{
		Vectors: vectors,
		Labels:  labels,
		Count:   n,
		Dims:    dims,
	}, nil
}

func LoadMCCRisk(path string) (map[string]float64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open mcc_risk: %w", err)
	}
	defer f.Close()

	var m map[string]float64
	if err := json.NewDecoder(f).Decode(&m); err != nil {
		return nil, fmt.Errorf("decode mcc_risk: %w", err)
	}
	return m, nil
}

func LoadNormConstants(path string) (*NormConstants, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open normalization: %w", err)
	}
	defer f.Close()

	var nc NormConstants
	if err := json.NewDecoder(f).Decode(&nc); err != nil {
		return nil, fmt.Errorf("decode normalization: %w", err)
	}
	return &nc, nil
}
