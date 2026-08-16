package postgres

import (
	"context"
	"time"

	"github.com/andrelair-platform/ktayl-policy-service/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DocumentRepo struct {
	db *pgxpool.Pool
}

func NewDocumentRepo(db *pgxpool.Pool) *DocumentRepo {
	return &DocumentRepo{db: db}
}

var _ domain.DocumentRepository = (*DocumentRepo)(nil)

func (r *DocumentRepo) Create(ctx context.Context, d *domain.Document) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	if d.CreatedAt.IsZero() {
		d.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO policy_documents (id, policy_id, type, minio_key, created_at)
		VALUES ($1, $2, $3, $4, $5)`,
		d.ID, d.PolicyID, d.Type, d.MinIOKey, d.CreatedAt,
	)
	return err
}

func (r *DocumentRepo) ListByPolicy(ctx context.Context, policyID uuid.UUID) ([]*domain.Document, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, policy_id, type, minio_key, created_at
		FROM   policy_documents
		WHERE  policy_id = $1
		ORDER  BY created_at DESC`,
		policyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*domain.Document
	for rows.Next() {
		var d domain.Document
		if err := rows.Scan(&d.ID, &d.PolicyID, &d.Type, &d.MinIOKey, &d.CreatedAt); err != nil {
			return nil, err
		}
		docs = append(docs, &d)
	}
	return docs, rows.Err()
}
