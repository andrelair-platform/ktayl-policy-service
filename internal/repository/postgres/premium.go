package postgres

import (
	"context"
	"time"

	"github.com/andrelair-platform/ktayl-policy-service/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PremiumRepo struct {
	db *pgxpool.Pool
}

func NewPremiumRepo(db *pgxpool.Pool) *PremiumRepo {
	return &PremiumRepo{db: db}
}

func (r *PremiumRepo) Create(ctx context.Context, p *domain.Premium) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO premiums (id, policy_id, amount, frequency, due_date, paid_at)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		p.ID, p.PolicyID, p.Amount, p.Frequency, p.DueDate, p.PaidAt,
	)
	return err
}

func (r *PremiumRepo) ListByPolicy(ctx context.Context, policyID uuid.UUID) ([]*domain.Premium, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id,policy_id,amount,frequency,due_date,paid_at
		 FROM premiums WHERE policy_id=$1 ORDER BY due_date`,
		policyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Premium
	for rows.Next() {
		p := &domain.Premium{}
		if err := rows.Scan(&p.ID, &p.PolicyID, &p.Amount, &p.Frequency, &p.DueDate, &p.PaidAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PremiumRepo) MarkPaid(ctx context.Context, id uuid.UUID, paidAt time.Time) error {
	_, err := r.db.Exec(ctx, `UPDATE premiums SET paid_at=$1 WHERE id=$2`, paidAt, id)
	return err
}
