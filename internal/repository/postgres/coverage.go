package postgres

import (
	"context"

	"github.com/andrelair-platform/ktayl-policy-service/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CoverageRepo struct {
	db *pgxpool.Pool
}

func NewCoverageRepo(db *pgxpool.Pool) *CoverageRepo {
	return &CoverageRepo{db: db}
}

func (r *CoverageRepo) Create(ctx context.Context, c *domain.Coverage) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO coverages (id, policy_id, type, insured_amount, deductible)
		VALUES ($1,$2,$3,$4,$5)`,
		c.ID, c.PolicyID, c.Type, c.InsuredAmount, c.Deductible,
	)
	return err
}

func (r *CoverageRepo) ListByPolicy(ctx context.Context, policyID uuid.UUID) ([]*domain.Coverage, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id,policy_id,type,insured_amount,deductible FROM coverages WHERE policy_id=$1`,
		policyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Coverage
	for rows.Next() {
		c := &domain.Coverage{}
		if err := rows.Scan(&c.ID, &c.PolicyID, &c.Type, &c.InsuredAmount, &c.Deductible); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *CoverageRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM coverages WHERE id=$1`, id)
	return err
}
