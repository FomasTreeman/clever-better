package models

import (
	"time"

	"github.com/google/uuid"
)

// OddsSnapshot represents a point-in-time snapshot of betting odds
type OddsSnapshot struct {
	Time        time.Time  `db:"time" json:"time" validate:"required"`
	RaceID      uuid.UUID  `db:"race_id" json:"race_id" validate:"required,uuid4"`
	RunnerID    uuid.UUID  `db:"runner_id" json:"runner_id" validate:"required,uuid4"`
	BackPrice   *float64   `db:"back_price" json:"back_price"`
	BackSize    *float64   `db:"back_size" json:"back_size"`
	LayPrice    *float64   `db:"lay_price" json:"lay_price"`
	LaySize     *float64   `db:"lay_size" json:"lay_size"`
	LTP         *float64   `db:"ltp" json:"ltp"`
	TotalVolume *float64   `db:"total_volume" json:"total_volume"`
}

// GetSpread returns the bid-ask spread (lay_price - back_price)
func (o *OddsSnapshot) GetSpread() float64 {
	if o.LayPrice == nil || o.BackPrice == nil {
		return 0
	}
	return *o.LayPrice - *o.BackPrice
}

// GetMidPrice returns the mid price between back and lay
func (o *OddsSnapshot) GetMidPrice() float64 {
	if o.LayPrice == nil || o.BackPrice == nil {
		if o.LTP != nil {
			return *o.LTP
		}
		return 0
	}
	return (*o.BackPrice + *o.LayPrice) / 2
}

// GetImpliedProbability returns the implied probability from mid price
func (o *OddsSnapshot) GetImpliedProbability() float64 {
	mid := o.GetMidPrice()
	if mid <= 0 {
		return 0
	}
	return 1.0 / mid
}
