package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func validPolicy() Policy {
	return Policy{
		ID:            uuid.New(),
		PolicyNumber:  "POL-2026-0001",
		HolderName:    "Jean Dupont",
		ProductCode:   "IARD-HAB-MRH",
		Status:        StatusDraft,
		EffectiveDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpiryDate:    time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestPolicy_ValidateHappyPath(t *testing.T) {
	p := validPolicy()
	if err := p.Validate(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestPolicy_ValidateEmptyPolicyNumber(t *testing.T) {
	p := validPolicy()
	p.PolicyNumber = ""
	if err := p.Validate(); err != ErrEmptyPolicyNumber {
		t.Fatalf("expected ErrEmptyPolicyNumber, got %v", err)
	}
}

func TestPolicy_ValidateEmptyHolderName(t *testing.T) {
	p := validPolicy()
	p.HolderName = ""
	if err := p.Validate(); err != ErrEmptyHolderName {
		t.Fatalf("expected ErrEmptyHolderName, got %v", err)
	}
}

func TestPolicy_ValidateEmptyProductCode(t *testing.T) {
	p := validPolicy()
	p.ProductCode = ""
	if err := p.Validate(); err != ErrEmptyProductCode {
		t.Fatalf("expected ErrEmptyProductCode, got %v", err)
	}
}

func TestPolicy_ValidateInvalidDateRange_Equal(t *testing.T) {
	p := validPolicy()
	p.ExpiryDate = p.EffectiveDate
	if err := p.Validate(); err != ErrInvalidDateRange {
		t.Fatalf("expected ErrInvalidDateRange for equal dates, got %v", err)
	}
}

func TestPolicy_ValidateInvalidDateRange_Before(t *testing.T) {
	p := validPolicy()
	p.ExpiryDate = p.EffectiveDate.Add(-24 * time.Hour)
	if err := p.Validate(); err != ErrInvalidDateRange {
		t.Fatalf("expected ErrInvalidDateRange for expiry before effective, got %v", err)
	}
}
