package postgres

import (
	"context"

	"github.com/andrelair-platform/ktayl-policy-service/internal/domain"
	"github.com/google/uuid"
)

// NullDocumentRepo is a no-op implementation used when no database DSN is configured.
type NullDocumentRepo struct{}

func NewNullDocumentRepo() *NullDocumentRepo { return &NullDocumentRepo{} }

var _ domain.DocumentRepository = NullDocumentRepo{}

func (NullDocumentRepo) Create(_ context.Context, _ *domain.Document) error { return nil }
func (NullDocumentRepo) ListByPolicy(_ context.Context, _ uuid.UUID) ([]*domain.Document, error) {
	return nil, nil
}

// NullAuditLogRepo is a no-op implementation used when no database DSN is configured.
type NullAuditLogRepo struct{}

func NewNullAuditLogRepo() *NullAuditLogRepo { return &NullAuditLogRepo{} }

var _ domain.AuditLogRepository = NullAuditLogRepo{}

func (NullAuditLogRepo) Insert(_ context.Context, _ any, _ *domain.AuditLog) error { return nil }
func (NullAuditLogRepo) ListByPolicy(_ context.Context, _ uuid.UUID) ([]*domain.AuditLog, error) {
	return nil, nil
}

// NullPolicyRepo is a no-op implementation used when no database DSN is configured.
type NullPolicyRepo struct{}

func NewNullPolicyRepo() *NullPolicyRepo { return &NullPolicyRepo{} }

var _ domain.PolicyRepository = NullPolicyRepo{}

func (NullPolicyRepo) Create(_ context.Context, _ *domain.Policy) error { return nil }
func (NullPolicyRepo) GetByID(_ context.Context, _ uuid.UUID) (*domain.Policy, error) {
	return nil, domain.ErrNotFound
}
func (NullPolicyRepo) GetByPolicyNumber(_ context.Context, _ string) (*domain.Policy, error) {
	return nil, domain.ErrNotFound
}
func (NullPolicyRepo) Update(_ context.Context, _ *domain.Policy) error { return nil }
func (NullPolicyRepo) List(_ context.Context, _ domain.ListParams) ([]*domain.Policy, error) {
	return nil, nil
}
