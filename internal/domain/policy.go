package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type PolicyStatus string

const (
	StatusDraft      PolicyStatus = "draft"
	StatusActive     PolicyStatus = "active"
	StatusSuspended  PolicyStatus = "suspended"
	StatusTerminated PolicyStatus = "terminated"
)

type Policy struct {
	ID            uuid.UUID
	PolicyNumber  string
	HolderName    string
	ProductCode   string
	Status        PolicyStatus
	EffectiveDate time.Time
	ExpiryDate    time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

var (
	ErrEmptyPolicyNumber = errors.New("policy number cannot be empty")
	ErrEmptyHolderName   = errors.New("holder name cannot be empty")
	ErrEmptyProductCode  = errors.New("product code cannot be empty")
	ErrInvalidDateRange  = errors.New("effective date must be before expiry date")
)

func (p *Policy) Validate() error {
	if p.PolicyNumber == "" {
		return ErrEmptyPolicyNumber
	}
	if p.HolderName == "" {
		return ErrEmptyHolderName
	}
	if p.ProductCode == "" {
		return ErrEmptyProductCode
	}
	if !p.EffectiveDate.Before(p.ExpiryDate) {
		return ErrInvalidDateRange
	}
	return nil
}
