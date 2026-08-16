package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrDocumentForbidden = errors.New("attestation requires ACTIVE or AMENDED status")
	ErrNoMinIOStore      = errors.New("document storage not configured")
)

// DocumentGenerator produces raw PDF bytes for a policy.
type DocumentGenerator interface {
	GenerateAttestation(p *Policy) ([]byte, error)
}

// DocumentStorer uploads bytes and returns (minioKey, presignedURL, error).
type DocumentStorer interface {
	Upload(ctx context.Context, policyID uuid.UUID, data []byte) (string, string, error)
	PresignKey(ctx context.Context, key string) (string, error)
}

// DocumentService orchestrates PDF generation + MinIO upload + metadata persistence.
type DocumentService struct {
	policyRepo PolicyRepository
	docRepo    DocumentRepository
	gen        DocumentGenerator
	store      DocumentStorer
	urlTTL     time.Duration
}

func NewDocumentService(
	policyRepo PolicyRepository,
	docRepo DocumentRepository,
	gen DocumentGenerator,
	store DocumentStorer,
	urlTTL time.Duration,
) *DocumentService {
	return &DocumentService{
		policyRepo: policyRepo,
		docRepo:    docRepo,
		gen:        gen,
		store:      store,
		urlTTL:     urlTTL,
	}
}

type AttestationResult struct {
	Document  *Document
	URL       string
	ExpiresAt time.Time
}

// GenerateAttestation generates a PDF attestation for a policy (AC-1, AC-2, AC-5).
// Only ACTIVE or AMENDED policies are eligible.
func (s *DocumentService) GenerateAttestation(ctx context.Context, policyID uuid.UUID) (*AttestationResult, error) {
	if s.store == nil {
		return nil, ErrNoMinIOStore
	}

	p, err := s.policyRepo.GetByID(ctx, policyID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrPolicyNotFound
		}
		return nil, err
	}
	if p.Status != StatusActive && p.Status != StatusAmended {
		return nil, fmt.Errorf("%w: current status is %s", ErrDocumentForbidden, p.Status)
	}

	data, err := s.gen.GenerateAttestation(p)
	if err != nil {
		return nil, fmt.Errorf("pdf generation: %w", err)
	}

	key, presigned, err := s.store.Upload(ctx, policyID, data)
	if err != nil {
		return nil, fmt.Errorf("document upload: %w", err)
	}

	doc := &Document{
		PolicyID:  policyID,
		Type:      "attestation",
		MinIOKey:  key,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.docRepo.Create(ctx, doc); err != nil {
		return nil, fmt.Errorf("document record: %w", err)
	}

	return &AttestationResult{
		Document:  doc,
		URL:       presigned,
		ExpiresAt: time.Now().UTC().Add(s.urlTTL),
	}, nil
}

// ListDocuments returns all documents for a policy (AC-4).
func (s *DocumentService) ListDocuments(ctx context.Context, policyID uuid.UUID) ([]*Document, []string, error) {
	if _, err := s.policyRepo.GetByID(ctx, policyID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil, ErrPolicyNotFound
		}
		return nil, nil, err
	}

	docs, err := s.docRepo.ListByPolicy(ctx, policyID)
	if err != nil {
		return nil, nil, err
	}

	var urls []string
	for _, d := range docs {
		if s.store != nil {
			u, _ := s.store.PresignKey(ctx, d.MinIOKey)
			urls = append(urls, u)
		} else {
			urls = append(urls, "")
		}
	}
	return docs, urls, nil
}
