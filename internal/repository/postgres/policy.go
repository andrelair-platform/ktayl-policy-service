package postgres

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/andrelair-platform/ktayl-policy-service/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PolicyRepo struct {
	db *pgxpool.Pool
}

func NewPolicyRepo(db *pgxpool.Pool) *PolicyRepo {
	return &PolicyRepo{db: db}
}

// compile-time interface check
var _ domain.PolicyRepository = (*PolicyRepo)(nil)

func (r *PolicyRepo) Create(ctx context.Context, p *domain.Policy) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO policies
			(id, policy_number, holder_name, product_code, status, effective_date, expiry_date, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		p.ID, p.PolicyNumber, p.HolderName, p.ProductCode,
		p.Status, p.EffectiveDate, p.ExpiryDate, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return domain.ErrPolicyNumberTaken
		}
	}
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

// EncodeCursor serialises the last-seen (created_at, id) pair into an opaque base64 cursor.
func EncodeCursor(createdAt time.Time, id uuid.UUID) string {
	raw := createdAt.UTC().Format(time.RFC3339Nano) + "|" + id.String()
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

// decodeCursor parses a cursor produced by EncodeCursor.
func decodeCursor(cursor string) (time.Time, uuid.UUID, error) {
	b, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("invalid cursor encoding")
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, uuid.Nil, fmt.Errorf("malformed cursor")
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("malformed cursor timestamp")
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("malformed cursor id")
	}
	return t, id, nil
}

func (r *PolicyRepo) List(ctx context.Context, params domain.ListParams) ([]*domain.Policy, error) {
	limit := params.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var (
		rows pgx.Rows
		err  error
	)

	const cols = `SELECT id,policy_number,holder_name,product_code,status,effective_date,expiry_date,created_at,updated_at FROM policies`

	if params.Cursor == "" {
		if params.Status != nil {
			rows, err = r.db.Query(ctx,
				cols+` WHERE status=$1 ORDER BY created_at DESC, id DESC LIMIT $2`,
				*params.Status, limit)
		} else {
			rows, err = r.db.Query(ctx,
				cols+` ORDER BY created_at DESC, id DESC LIMIT $1`, limit)
		}
	} else {
		cur, curID, cerr := decodeCursor(params.Cursor)
		if cerr != nil {
			return nil, cerr
		}
		if params.Status != nil {
			rows, err = r.db.Query(ctx,
				cols+` WHERE status=$1 AND (created_at,id) < ($2,$3) ORDER BY created_at DESC, id DESC LIMIT $4`,
				*params.Status, cur, curID, limit)
		} else {
			rows, err = r.db.Query(ctx,
				cols+` WHERE (created_at,id) < ($1,$2) ORDER BY created_at DESC, id DESC LIMIT $3`,
				cur, curID, limit)
		}
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
			return nil, domain.ErrNotFound
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
