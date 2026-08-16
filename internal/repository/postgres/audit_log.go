package postgres

import (
	"context"
	"time"

	"github.com/andrelair-platform/ktayl-policy-service/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditLogRepo struct {
	db *pgxpool.Pool
}

func NewAuditLogRepo(db *pgxpool.Pool) *AuditLogRepo {
	return &AuditLogRepo{db: db}
}

var _ domain.AuditLogRepository = (*AuditLogRepo)(nil)

// Insert writes one audit row. tx must be a pgx.Tx obtained from pgxpool.Pool.BeginTx.
func (r *AuditLogRepo) Insert(ctx context.Context, tx any, entry *domain.AuditLog) error {
	pgxTx, ok := tx.(pgx.Tx)
	if !ok {
		return domain.ErrInvalidTransition // programming error: wrong tx type
	}
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	if entry.OccurredAt.IsZero() {
		entry.OccurredAt = time.Now().UTC()
	}
	_, err := pgxTx.Exec(ctx, `
		INSERT INTO policy_audit_log
			(id, policy_id, from_status, to_status, actor_id, reason, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		entry.ID,
		entry.PolicyID,
		string(entry.FromStatus),
		string(entry.ToStatus),
		entry.ActorID,
		entry.Reason,
		entry.OccurredAt,
	)
	return err
}

func (r *AuditLogRepo) ListByPolicy(ctx context.Context, policyID uuid.UUID) ([]*domain.AuditLog, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, policy_id, from_status, to_status, actor_id, reason, occurred_at
		FROM   policy_audit_log
		WHERE  policy_id = $1
		ORDER  BY occurred_at ASC`,
		policyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*domain.AuditLog
	for rows.Next() {
		var l domain.AuditLog
		var fromS, toS string
		if err := rows.Scan(&l.ID, &l.PolicyID, &fromS, &toS, &l.ActorID, &l.Reason, &l.OccurredAt); err != nil {
			return nil, err
		}
		l.FromStatus = domain.PolicyStatus(fromS)
		l.ToStatus = domain.PolicyStatus(toS)
		logs = append(logs, &l)
	}
	return logs, rows.Err()
}
