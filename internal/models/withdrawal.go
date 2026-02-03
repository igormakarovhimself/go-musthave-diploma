package models

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

type Withdrawal struct {
	ID          int64           `db:"id" json:"-"`
	UserID      int64           `db:"user_id" json:"-"`
	Order       string          `db:"order" json:"-"`
	Sum         decimal.Decimal `db:"sum" json:"-"`
	ProcessedAt time.Time       `db:"processed_at" json:"-"`
}

func (w Withdrawal) MarshalJSON() ([]byte, error) {
	sum, _ := w.Sum.Float64()

	return json.Marshal(&struct {
		Order       string    `json:"order"`
		Sum         float64   `json:"sum"`
		ProcessedAt time.Time `json:"processed_at"`
	}{
		Order:       w.Order,
		Sum:         sum,
		ProcessedAt: w.ProcessedAt,
	})
}

func (w *Withdrawal) UnmarshalJSON(data []byte) error {
	var tmp struct {
		Order       string    `json:"order"`
		Sum         float64   `json:"sum"`
		ProcessedAt time.Time `json:"processed_at"`
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	w.Order = tmp.Order
	w.Sum = decimal.NewFromFloat(tmp.Sum)
	w.ProcessedAt = tmp.ProcessedAt

	return nil
}
