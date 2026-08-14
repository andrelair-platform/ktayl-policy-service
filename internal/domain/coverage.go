package domain

import (
	"errors"

	"github.com/google/uuid"
)

type Coverage struct {
	ID            uuid.UUID
	PolicyID      uuid.UUID
	Type          string
	InsuredAmount int64 // eurocents
	Deductible    int64 // eurocents
}

var (
	ErrEmptyCoverageType        = errors.New("coverage type cannot be empty")
	ErrNonPositiveInsuredAmount = errors.New("insured amount must be positive")
	ErrNegativeDeductible       = errors.New("deductible cannot be negative")
)

func (c *Coverage) Validate() error {
	if c.Type == "" {
		return ErrEmptyCoverageType
	}
	if c.InsuredAmount <= 0 {
		return ErrNonPositiveInsuredAmount
	}
	if c.Deductible < 0 {
		return ErrNegativeDeductible
	}
	return nil
}
