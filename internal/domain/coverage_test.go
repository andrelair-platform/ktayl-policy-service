package domain

import (
	"testing"

	"github.com/google/uuid"
)

func validCoverage() Coverage {
	return Coverage{
		ID:            uuid.New(),
		PolicyID:      uuid.New(),
		Type:          "Responsabilité Civile",
		InsuredAmount: 50000_00, // 50 000 €
		Deductible:    500_00,   // 500 €
	}
}

func TestCoverage_ValidateHappyPath(t *testing.T) {
	c := validCoverage()
	if err := c.Validate(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCoverage_ValidateZeroDeductible(t *testing.T) {
	c := validCoverage()
	c.Deductible = 0
	if err := c.Validate(); err != nil {
		t.Fatalf("zero deductible should be valid, got %v", err)
	}
}

func TestCoverage_ValidateEmptyType(t *testing.T) {
	c := validCoverage()
	c.Type = ""
	if err := c.Validate(); err != ErrEmptyCoverageType {
		t.Fatalf("expected ErrEmptyCoverageType, got %v", err)
	}
}

func TestCoverage_ValidateZeroInsuredAmount(t *testing.T) {
	c := validCoverage()
	c.InsuredAmount = 0
	if err := c.Validate(); err != ErrNonPositiveInsuredAmount {
		t.Fatalf("expected ErrNonPositiveInsuredAmount, got %v", err)
	}
}

func TestCoverage_ValidateNegativeInsuredAmount(t *testing.T) {
	c := validCoverage()
	c.InsuredAmount = -1
	if err := c.Validate(); err != ErrNonPositiveInsuredAmount {
		t.Fatalf("expected ErrNonPositiveInsuredAmount, got %v", err)
	}
}

func TestCoverage_ValidateNegativeDeductible(t *testing.T) {
	c := validCoverage()
	c.Deductible = -1
	if err := c.Validate(); err != ErrNegativeDeductible {
		t.Fatalf("expected ErrNegativeDeductible, got %v", err)
	}
}
