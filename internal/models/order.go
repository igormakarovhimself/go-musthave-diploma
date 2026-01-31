package models

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

const (
	OrderStatusNew        = "NEW"
	OrderStatusProcessing = "PROCESSING"
	OrderStatusInvalid    = "INVALID"
	OrderStatusProcessed  = "PROCESSED"
)

type Order struct {
	Number     string           `db:"number" json:"-"`
	UserID     int64            `db:"user_id" json:"-"`
	Status     string           `db:"status" json:"-"`
	Accrual    *decimal.Decimal `db:"accrual" json:"-"`
	UploadedAt time.Time        `db:"uploaded_at" json:"-"`
}

func (o Order) MarshalJSON() ([]byte, error) {
	var accrual *float64
	if o.Accrual != nil {
		val, _ := o.Accrual.Float64()
		accrual = &val
	}

	return json.Marshal(&struct {
		Number     string    `json:"number"`
		Status     string    `json:"status"`
		Accrual    *float64  `json:"accrual,omitempty"`
		UploadedAt time.Time `json:"uploaded_at"`
	}{
		Number:     o.Number,
		Status:     o.Status,
		Accrual:    accrual,
		UploadedAt: o.UploadedAt,
	})
}

func (o *Order) UnmarshalJSON(data []byte) error {
	var tmp struct {
		Number     string    `json:"number"`
		Status     string    `json:"status"`
		Accrual    *float64  `json:"accrual,omitempty"`
		UploadedAt time.Time `json:"uploaded_at"`
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	o.Number = tmp.Number
	o.Status = tmp.Status
	o.UploadedAt = tmp.UploadedAt

	if tmp.Accrual != nil {
		accrual := decimal.NewFromFloat(*tmp.Accrual)
		o.Accrual = &accrual
	}

	return nil
}
