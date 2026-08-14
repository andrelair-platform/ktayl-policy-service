package postgres

import (
	"context"
	"errors"

	"github.com/andrelair-platform/ktayl-policy-service/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type PolicyRepo struct {
	db *pgxpool.Pool
}

func NewPolicyRepo(db *pgxpool.Pool) *PolicyRepo {
	return &PolicyRepo{db: db}
}

func (r *PolicyRepo) Create(ctx context.Context, p *domain.Policy) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO policies
			(id, policy_number, holder_name, product_code, status, effective_date, expiry_date, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		p.ID, p.PolicyNumber, p.HolderName, p.ProductCode,
		p.Status, p.EffectiveDate, p.ExpiryDate, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (r *PolicyRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Policy, error) {
	return r.scanOne(ctx,
		`SELECT id,policy_number,holder_name,product_code,status,effective_date,expiry_date,created_at,updated_at
		 FROM policies WHERE id=$1`, id)
}

func (r *PolicyRepo) GetByPolicyNumber(ctx context.Context, number string) (*domain.Policy, error) {
	return r.scanOne(ctx,
		`SELECT id,policy_number,holder_name,product_code,status,effective_date,expiry_date,created_at,updated_at
		 FROM policies WHERE policy_number=$1`, number)
}

func (r *PolicyRepo) Update(ctx context.Context, p *domain.Policy) error {
	_, err := r.db.Exec(ctx, `
		UPDATE policies SET
			holder_name=$1, product_code=$2, status=$3,
			effective_date=$4, expiry_date=$5, updated_at=$6
		WHERE id=$7`,
		p.HolderName, p.ProductCode, p.Status,
		p.EffectiveDate, p.ExpiryDate, p.UpdatedAt, p.ID,
	)
	return err
}

func (r *PolicyRepo) List(ctx context.Context, status *domain.PolicyStatus) ([]*domain.Policy, error) {
	var rows pgx.Rows
	var err error
	if status != nil {
		rows, err = r.db.Query(ctx,
			`SELECT id,policy_number,holder_name,product_code,status,effective_date,expiry_date,created_at,updated_at
			 FROM policies WHERE status=$1 ORDER BY created_at DESC`, *status)
	} else {
		rows, err = r.db.Query(ctx,
			`SELECT id,policy_number,holder_name,product_code,status,effective_date,expiry_date,created_at,updated_at
			 FROM policies ORDER BY created_at DESC`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []*domain.Policy
	for rows.Next() {
		p := &domain.Policy{}
		if err := scanPolicy(rows, p); err != nil {
			return nil, err
		}
		policies = append(policies, p)
	}
	return policies, rows.Err()
}

func (r *PolicyRepo) scanOne(ctx context.Context, q string, arg any) (*domain.Policy, error) {
	row := r.db.QueryRow(ctx, q, arg)
	p := &domain.Policy{}
	if err := scanPolicy(row, p); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanPolicy(s scanner, p *domain.Policy) error {
	return s.Scan(
		&p.ID, &p.PolicyNumber, &p.HolderName, &p.ProductCode,
		&p.Status, &p.EffectiveDate, &p.ExpiryDate, &p.CreatedAt, &p.UpdatedAt,
	)
}
