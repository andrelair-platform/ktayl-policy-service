// Package integration_test contains L2 integration tests that require a real
// PostgreSQL database and NATS JetStream. Run via `make test-integration`.
// Tests are skipped automatically when INTEGRATION_DSN is not set.
package integration_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/andrelair-platform/ktayl-policy-service/db"
	"github.com/andrelair-platform/ktayl-policy-service/internal/domain"
	pgRepo "github.com/andrelair-platform/ktayl-policy-service/internal/repository/postgres"
	"github.com/golang-migrate/migrate/v4"
	pgx5 "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

var (
	integrationDSN     string
	integrationNATSURL string
)

func TestMain(m *testing.M) {
	integrationDSN = os.Getenv("INTEGRATION_DSN")
	if integrationDSN == "" {
		fmt.Println("INTEGRATION_DSN not set — skipping integration tests (run `make test-integration`)")
		os.Exit(0)
	}
	integrationNATSURL = os.Getenv("INTEGRATION_NATS_URL")

	if err := runMigrations(integrationDSN); err != nil {
		fmt.Fprintf(os.Stderr, "migration failed: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func runMigrations(dsn string) error {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("pool: %w", err)
	}
	defer pool.Close()

	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close() // must run before pool.Close() (LIFO order)
	driver, err := pgx5.WithInstance(sqlDB, &pgx5.Config{})
	if err != nil {
		return fmt.Errorf("migrate driver: %w", err)
	}
	src, err := iofs.New(db.Migrations, "migrations")
	if err != nil {
		return fmt.Errorf("migrate source: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

// newService creates a real PolicyService connected to the integration database.
func newService(t *testing.T) (*domain.PolicyService, *pgxpool.Pool) {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), integrationDSN)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(pool.Close)
	svc := domain.NewPolicyService(
		pgRepo.NewPolicyRepo(pool),
		pgRepo.NewAuditLogRepo(pool),
		pool,
	)
	return svc, pool
}
