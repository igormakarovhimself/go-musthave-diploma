package models

import (
	"encoding/json"

	"github.com/shopspring/decimal"
)

type Balance struct {
	Current   decimal.Decimal `json:"-"`
	Withdrawn decimal.Decimal `json:"-"`
}

func (b Balance) MarshalJSON() ([]byte, error) {
	current, _ := b.Current.Float64()
	withdrawn, _ := b.Withdrawn.Float64()

	return json.Marshal(&struct {
		Current   float64 `json:"current"`
		Withdrawn float64 `json:"withdrawn"`
	}{
		Current:   current,
		Withdrawn: withdrawn,
	})
}

func (b *Balance) UnmarshalJSON(data []byte) error {
	var tmp struct {
		Current   float64 `json:"current"`
		Withdrawn float64 `json:"withdrawn"`
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	b.Current = decimal.NewFromFloat(tmp.Current)
	b.Withdrawn = decimal.NewFromFloat(tmp.Withdrawn)
	return nil
}
