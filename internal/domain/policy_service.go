package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrPolicyNotFound = errors.New("policy not found")
	ErrCannotCancel   = errors.New("only draft policies can be cancelled via this endpoint")
)

type PolicyService struct {
	repo PolicyRepository
}

func NewPolicyService(repo PolicyRepository) *PolicyService {
	return &PolicyService{repo: repo}
}

func (s *PolicyService) Create(ctx context.Context, p *Policy) error {
	if err := p.Validate(); err != nil {
		return err
	}
	p.ID = uuid.New()
	p.Status = StatusDraft
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now
	if err := s.repo.Create(ctx, p); err != nil {
		return err
	}
	return nil
}

func (s *PolicyService) GetByID(ctx context.Context, id uuid.UUID) (*Policy, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrPolicyNotFound
		}
		return nil, err
	}
	return p, nil
}

func (s *PolicyService) List(ctx context.Context, params ListParams) ([]*Policy, error) {
	return s.repo.List(ctx, params)
}

func (s *PolicyService) Update(ctx context.Context, id uuid.UUID, upd *Policy) (*Policy, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrPolicyNotFound
		}
		return nil, err
	}
	p.HolderName = upd.HolderName
	p.ProductCode = upd.ProductCode
	p.EffectiveDate = upd.EffectiveDate
	p.ExpiryDate = upd.ExpiryDate
	p.UpdatedAt = time.Now().UTC()
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// Cancel soft-cancels a DRAFT policy by moving it to terminated.
// State machine transitions (submit/activate) are handled in S004.
func (s *PolicyService) Cancel(ctx context.Context, id uuid.UUID) error {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrPolicyNotFound
		}
		return err
	}
	if p.Status != StatusDraft {
		return ErrCannotCancel
	}
	p.Status = StatusTerminated
	p.UpdatedAt = time.Now().UTC()
	return s.repo.Update(ctx, p)
}
