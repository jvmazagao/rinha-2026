package vectorize

import (
	"math"
	"time"

	"rinha/internal/dataset"
)

type LastTransaction struct {
	KmFromCurrent float64 `json:"km_from_current"`
	RequestedAt   string  `json:"requested_at"`
}

type Terminal struct {
	IsOnline    bool    `json:"is_online"`
	CardPresent bool    `json:"card_present"`
	KmFromHome  float64 `json:"km_from_home"`
}

type Merchant struct {
	ID        string  `json:"id"`
	MCC       string  `json:"mcc"`
	AvgAmount float64 `json:"avg_amount"`
}

type Customer struct {
	AvgAmount      float64  `json:"avg_amount"`
	TxCount24h     float64  `json:"tx_count_24h"`
	KnownMerchants []string `json:"known_merchants"`
}

type Transaction struct {
	Amount          float64          `json:"amount"`
	Installments    float64          `json:"installments"`
	RequestedAt     string           `json:"requested_at"`
	LastTransaction *LastTransaction `json:"last_transaction"`
	Terminal        Terminal         `json:"terminal"`
	Merchant        Merchant         `json:"merchant"`
	Customer        Customer         `json:"customer"`
}

func clamp(x float64) float32 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return float32(x)
}

func Vectorize(tx *Transaction, nc *dataset.NormConstants, mccRisk map[string]float64, out []float32) {
	// [0] amount
	out[0] = clamp(tx.Amount / nc.MaxAmount)

	// [1] installments
	out[1] = clamp(tx.Installments / nc.MaxInstallments)

	// [2] amount_vs_avg
	var avgRatio float64
	if tx.Customer.AvgAmount > 0 {
		avgRatio = (tx.Amount / tx.Customer.AvgAmount) / nc.AmountVsAvgRatio
	}
	out[2] = clamp(avgRatio)

	// [3] hour_of_day and [4] day_of_week
	t, err := time.Parse(time.RFC3339, tx.RequestedAt)
	if err != nil {
		t = time.Now().UTC()
	} else {
		t = t.UTC()
	}
	out[3] = float32(t.Hour()) / 23.0
	// time.Weekday: Sunday=0 ... Saturday=6; we need Monday=0 ... Sunday=6
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 6
	} else {
		wd--
	}
	out[4] = float32(wd) / 6.0

	// [5] minutes_since_last_tx and [6] km_from_last_tx
	if tx.LastTransaction == nil {
		out[5] = -1.0
		out[6] = -1.0
	} else {
		lastT, err := time.Parse(time.RFC3339, tx.LastTransaction.RequestedAt)
		if err != nil {
			out[5] = -1.0
		} else {
			mins := t.Sub(lastT.UTC()).Minutes()
			if mins < 0 {
				mins = 0
			}
			out[5] = clamp(mins / nc.MaxMinutes)
		}
		out[6] = clamp(tx.LastTransaction.KmFromCurrent / nc.MaxKm)
	}

	// [7] km_from_home
	out[7] = clamp(tx.Terminal.KmFromHome / nc.MaxKm)

	// [8] tx_count_24h
	out[8] = clamp(tx.Customer.TxCount24h / nc.MaxTxCount24h)

	// [9] is_online
	if tx.Terminal.IsOnline {
		out[9] = 1.0
	} else {
		out[9] = 0.0
	}

	// [10] card_present
	if tx.Terminal.CardPresent {
		out[10] = 1.0
	} else {
		out[10] = 0.0
	}

	// [11] unknown_merchant
	out[11] = 1.0
	for _, km := range tx.Customer.KnownMerchants {
		if km == tx.Merchant.ID {
			out[11] = 0.0
			break
		}
	}

	// [12] mcc_risk
	risk, ok := mccRisk[tx.Merchant.MCC]
	if !ok {
		risk = 0.5
	}
	out[12] = float32(math.Max(0, math.Min(1, risk)))

	// [13] merchant_avg_amount
	out[13] = clamp(tx.Merchant.AvgAmount / nc.MaxMerchantAvgAmount)
}
