package repository

import (
	"context"
	"time"

	"github.com/andrelair-platform/ktayl-policy-service/internal/domain"
	"github.com/google/uuid"
)

type PolicyRepository interface {
	Create(ctx context.Context, p *domain.Policy) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Policy, error)
	GetByPolicyNumber(ctx context.Context, number string) (*domain.Policy, error)
	Update(ctx context.Context, p *domain.Policy) error
	List(ctx context.Context, status *domain.PolicyStatus) ([]*domain.Policy, error)
}

type CoverageRepository interface {
	Create(ctx context.Context, c *domain.Coverage) error
	ListByPolicy(ctx context.Context, policyID uuid.UUID) ([]*domain.Coverage, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type PremiumRepository interface {
	Create(ctx context.Context, p *domain.Premium) error
	ListByPolicy(ctx context.Context, policyID uuid.UUID) ([]*domain.Premium, error)
	MarkPaid(ctx context.Context, id uuid.UUID, paidAt time.Time) error
}
