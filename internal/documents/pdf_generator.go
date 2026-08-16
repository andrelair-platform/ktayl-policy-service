package documents

import (
	"bytes"
	"fmt"
	"time"

	"github.com/andrelair-platform/ktayl-policy-service/internal/domain"
	"github.com/go-pdf/fpdf"
)

// productNames maps internal product codes to human-readable French names.
var productNames = map[string]string{
	"IARD-AUTO-RC":  "Responsabilité Civile Auto",
	"IARD-HAB-MRH":  "Multirisque Habitation",
	"PREV-IND-01":   "Prévoyance Individuelle",
}

// lobNames maps product codes to their line of business.
var lobNames = map[string]string{
	"IARD-AUTO-RC":  "IARD — Automobile",
	"IARD-HAB-MRH":  "IARD — Habitation",
	"PREV-IND-01":   "Vie & Prévoyance",
}

// GenerateAttestation builds a PDF attestation (ACPR Art.L113-5) and returns the raw bytes.
// No CGO — uses go-pdf/fpdf (pure Go).
func GenerateAttestation(p *domain.Policy) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 20, 20)
	pdf.AddPage()

	// ── Header ──────────────────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 18)
	pdf.SetTextColor(10, 60, 120)
	pdf.CellFormat(0, 10, "ATTESTATION D'ASSURANCE", "", 1, "C", false, 0, "")

	pdf.SetFont("Helvetica", "", 11)
	pdf.SetTextColor(80, 80, 80)
	pdf.CellFormat(0, 7, "ktayl solution — Courtier en Assurances", "", 1, "C", false, 0, "")
	pdf.Ln(6)

	// ── Policy Info Box ──────────────────────────────────────────────────────────
	pdf.SetDrawColor(10, 60, 120)
	pdf.SetLineWidth(0.5)
	pdf.Rect(20, pdf.GetY(), 170, 54, "D")

	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(10, 60, 120)
	pdf.SetX(24)
	pdf.CellFormat(0, 8, "INFORMATIONS DE LA POLICE", "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(30, 30, 30)

	rows := []struct{ label, value string }{
		{"Numéro de police", p.PolicyNumber},
		{"Assuré", p.HolderName},
		{"Produit", productName(p.ProductCode)},
		{"Branche", lob(p.ProductCode)},
		{"Date d'effet", p.EffectiveDate.UTC().Format("02/01/2006")},
		{"Date d'échéance", p.ExpiryDate.UTC().Format("02/01/2006")},
	}
	for _, row := range rows {
		pdf.SetX(24)
		pdf.SetFont("Helvetica", "B", 10)
		pdf.CellFormat(55, 7, row.label+" :", "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
		pdf.CellFormat(0, 7, row.value, "", 1, "L", false, 0, "")
	}

	pdf.Ln(8)

	// ── Status banner ────────────────────────────────────────────────────────────
	pdf.SetFillColor(220, 240, 220)
	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetTextColor(10, 100, 10)
	pdf.CellFormat(0, 9, fmt.Sprintf("  Statut : %s", statusLabel(p.Status)), "", 1, "L", true, 0, "")
	pdf.Ln(6)

	// ── Legal notice ─────────────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "I", 9)
	pdf.SetTextColor(100, 100, 100)
	notice := "Ce document constitue une attestation d'assurance conformément à l'article L113-5 " +
		"du Code des assurances et à la réglementation ACPR en vigueur. Il est valable uniquement " +
		"pour la période mentionnée ci-dessus."
	pdf.MultiCell(0, 5, notice, "", "L", false)

	pdf.Ln(6)

	// ── Generation timestamp ─────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(140, 140, 140)
	pdf.CellFormat(0, 5,
		fmt.Sprintf("Document généré le %s UTC", time.Now().UTC().Format("02/01/2006 à 15:04:05")),
		"", 1, "R", false, 0, "")

	// ── Footer rule ──────────────────────────────────────────────────────────────
	pdf.SetY(-20)
	pdf.SetDrawColor(180, 180, 180)
	pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetTextColor(150, 150, 150)
	pdf.CellFormat(0, 5, "ktayl solution SAS — ORIAS n° 00000000 — Document non contractuel", "", 0, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf render: %w", err)
	}
	return buf.Bytes(), nil
}

func productName(code string) string {
	if n, ok := productNames[code]; ok {
		return n
	}
	return code
}

func lob(code string) string {
	if l, ok := lobNames[code]; ok {
		return l
	}
	return "Autre"
}

func statusLabel(s domain.PolicyStatus) string {
	switch s {
	case domain.StatusActive:
		return "ACTIVE"
	case domain.StatusAmended:
		return "EN COURS D'AVENANT"
	default:
		return string(s)
	}
}
