package domain

import (
	"context"

	"github.com/google/uuid"
)

// NullAuditLog returns a no-op AuditLogRepository for use in tests and no-db mode.
func NullAuditLog() AuditLogRepository { return nullAuditLog{} }

type nullAuditLog struct{}

func (nullAuditLog) Insert(_ context.Context, _ any, _ *AuditLog) error { return nil }
func (nullAuditLog) ListByPolicy(_ context.Context, _ uuid.UUID) ([]*AuditLog, error) {
	return nil, nil
}
