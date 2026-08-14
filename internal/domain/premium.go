package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Frequency string

const (
	FrequencyMonthly   Frequency = "monthly"
	FrequencyQuarterly Frequency = "quarterly"
	FrequencyAnnual    Frequency = "annual"
)

type Premium struct {
	ID        uuid.UUID
	PolicyID  uuid.UUID
	Amount    int64 // eurocents
	Frequency Frequency
	DueDate   time.Time
	PaidAt    *time.Time
}

var ErrNonPositivePremiumAmount = errors.New("premium amount must be positive")

func (p *Premium) Validate() error {
	if p.Amount <= 0 {
		return ErrNonPositivePremiumAmount
	}
	return nil
}

func (p *Premium) IsPaid() bool {
	return p.PaidAt != nil
}
