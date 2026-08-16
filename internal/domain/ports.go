package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Repository errors — defined here so callers don't import the postgres package.
var (
	ErrNotFound          = errors.New("not found")
	ErrPolicyNumberTaken = errors.New("policy number already in use")
)

// ListParams controls cursor-based pagination (cursor = last seen created_at|id pair, base64-encoded).
type ListParams struct {
	Status *PolicyStatus
	Cursor string // opaque: base64(created_at_rfc3339 + "|" + id)
	Limit  int    // 1–100, default 20
}

// PolicyRepository is the port that postgres (and mocks) implement.
type PolicyRepository interface {
	Create(ctx context.Context, p *Policy) error
	GetByID(ctx context.Context, id uuid.UUID) (*Policy, error)
	GetByPolicyNumber(ctx context.Context, number string) (*Policy, error)
	Update(ctx context.Context, p *Policy) error
	List(ctx context.Context, params ListParams) ([]*Policy, error)
}

// CoverageRepository is used by S005 and later stories.
type CoverageRepository interface {
	Create(ctx context.Context, c *Coverage) error
	ListByPolicy(ctx context.Context, policyID uuid.UUID) ([]*Coverage, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// PremiumRepository is used by future premium management stories.
type PremiumRepository interface {
	Create(ctx context.Context, p *Premium) error
	ListByPolicy(ctx context.Context, policyID uuid.UUID) ([]*Premium, error)
	MarkPaid(ctx context.Context, id uuid.UUID, paidAt time.Time) error
}

// AuditLog is a single DORA Art.9 traceability record.
type AuditLog struct {
	ID         uuid.UUID
	PolicyID   uuid.UUID
	FromStatus PolicyStatus
	ToStatus   PolicyStatus
	ActorID    string
	Reason     string
	OccurredAt time.Time
}

// AuditLogRepository persists audit rows within caller-managed transactions.
type AuditLogRepository interface {
	// Insert writes one audit row. tx must be a *pgx.Tx — typed as any to avoid
	// importing pgx in the domain package.
	Insert(ctx context.Context, tx any, log *AuditLog) error
	ListByPolicy(ctx context.Context, policyID uuid.UUID) ([]*AuditLog, error)
}
