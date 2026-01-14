package models

import (
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
	Number     string           `db:"number" json:"number"`
	UserID     int64            `db:"user_id" json:"-"`
	Status     string           `db:"status" json:"status"`
	Accrual    *decimal.Decimal `db:"accrual" json:"accrual,omitempty"`
	UploadedAt time.Time        `db:"uploaded_at" json:"uploaded_at"`
}
