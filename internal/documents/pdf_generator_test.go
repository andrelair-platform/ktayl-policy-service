package documents_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/andrelair-platform/ktayl-policy-service/internal/documents"
	"github.com/andrelair-platform/ktayl-policy-service/internal/domain"
	"github.com/google/uuid"
)

func activePolicy() *domain.Policy {
	return &domain.Policy{
		ID:            uuid.New(),
		PolicyNumber:  "POL-TEST-001",
		HolderName:    "Jean Dupont",
		ProductCode:   "IARD-AUTO-RC",
		Status:        domain.StatusActive,
		EffectiveDate: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		ExpiryDate:    time.Date(2027, 9, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestGenerateAttestation_NonEmpty(t *testing.T) {
	data, err := documents.GenerateAttestation(activePolicy())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty PDF bytes")
	}
}

func TestGenerateAttestation_PDFHeader(t *testing.T) {
	data, err := documents.GenerateAttestation(activePolicy())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// PDF magic bytes
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatalf("output does not start with PDF magic bytes; got: %q", data[:10])
	}
}

func TestGenerateAttestation_UnknownProduct(t *testing.T) {
	p := activePolicy()
	p.ProductCode = "UNKNOWN-CODE"
	data, err := documents.GenerateAttestation(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty PDF for unknown product code")
	}
}
