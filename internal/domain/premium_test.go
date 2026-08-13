package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func validPremium() Premium {
	return Premium{
		ID:        uuid.New(),
		PolicyID:  uuid.New(),
		Amount:    120_00, // 120 €
		Frequency: FrequencyMonthly,
		DueDate:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestPremium_ValidateHappyPath(t *testing.T) {
	p := validPremium()
	if err := p.Validate(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestPremium_ValidateZeroAmount(t *testing.T) {
	p := validPremium()
	p.Amount = 0
	if err := p.Validate(); err != ErrNonPositivePremiumAmount {
		t.Fatalf("expected ErrNonPositivePremiumAmount, got %v", err)
	}
}

func TestPremium_ValidateNegativeAmount(t *testing.T) {
	p := validPremium()
	p.Amount = -1
	if err := p.Validate(); err != ErrNonPositivePremiumAmount {
		t.Fatalf("expected ErrNonPositivePremiumAmount, got %v", err)
	}
}

func TestPremium_IsPaid_False(t *testing.T) {
	p := validPremium()
	if p.IsPaid() {
		t.Fatal("premium with nil PaidAt should not be paid")
	}
}

func TestPremium_IsPaid_True(t *testing.T) {
	p := validPremium()
	now := time.Now()
	p.PaidAt = &now
	if !p.IsPaid() {
		t.Fatal("premium with non-nil PaidAt should be paid")
	}
}
