package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Withdrawal struct {
	ID          int64           `db:"id" json:"-"`
	UserID      int64           `db:"user_id" json:"-"`
	Order       string          `db:"order" json:"order"`
	Sum         decimal.Decimal `db:"sum" json:"sum"`
	ProcessedAt time.Time       `db:"processed_at" json:"processed_at"`
}
