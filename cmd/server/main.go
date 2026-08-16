package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andrelair-platform/ktayl-policy-service/db"
	"github.com/andrelair-platform/ktayl-policy-service/internal/api"
	apimw "github.com/andrelair-platform/ktayl-policy-service/internal/api/middleware"
	"github.com/andrelair-platform/ktayl-policy-service/internal/documents"
	"github.com/andrelair-platform/ktayl-policy-service/internal/domain"
	"github.com/andrelair-platform/ktayl-policy-service/internal/events"
	pgRepo "github.com/andrelair-platform/ktayl-policy-service/internal/repository/postgres"
	"github.com/golang-migrate/migrate/v4"
	pgx5 "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/spf13/viper"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	viper.SetDefault("port", "8080")
	viper.SetDefault("read_timeout_s", 10)
	viper.SetDefault("write_timeout_s", 10)
	viper.SetDefault("dsn", "")
	viper.SetDefault("authentik_jwks_url", "")
	viper.SetDefault("minio_endpoint", "")
	viper.SetDefault("minio_access_key", "")
	viper.SetDefault("minio_secret_key", "")
	viper.SetDefault("minio_use_ssl", false)
	viper.SetDefault("minio_public_url", "")
	viper.SetDefault("document_url_ttl_s", 3600)
	viper.SetDefault("nats_url", "nats://nats.nats.svc.cluster.local:4222")
	viper.SetDefault("nats_ca_cert", "")
	viper.AutomaticEnv()

	// Background context drives the JWKS refresh goroutine for the lifetime of the process.
	bgCtx := context.Background()

	// Set up JWT middleware when AUTHENTIK_JWKS_URL is provided.
	var jwtMw func(http.Handler) http.Handler
	if jwksURL := viper.GetString("authentik_jwks_url"); jwksURL != "" {
		mw, err := apimw.NewJWKSMiddleware(bgCtx, jwksURL)
		if err != nil {
			log.Error("failed to create JWKS middleware", "url", jwksURL, "err", err)
			os.Exit(1)
		}
		jwtMw = mw
		log.Info("JWT auth enabled", "jwks_url", jwksURL)
	} else {
		log.Warn("AUTHENTIK_JWKS_URL not set — running without JWT auth")
	}

	dsn := viper.GetString("dsn")
	if dsn == "" {
		log.Warn("DSN not set — running without database (healthz only)")
	}

	var svc *domain.PolicyService
	var docSvc *domain.DocumentService
	if dsn != "" {
		pool, err := pgxpool.New(bgCtx, dsn)
		if err != nil {
			log.Error("failed to create db pool", "err", err)
			os.Exit(1)
		}
		if err := runMigrations(pool, log); err != nil {
			pool.Close()
			log.Error("migration failed", "err", err)
			os.Exit(1)
		}
		defer pool.Close()

		svc = domain.NewPolicyService(pgRepo.NewPolicyRepo(pool), pgRepo.NewAuditLogRepo(pool), pool)

		if ep := viper.GetString("minio_endpoint"); ep != "" {
			urlTTL := time.Duration(viper.GetInt("document_url_ttl_s")) * time.Second
			store, err := documents.NewMinIOStore(bgCtx, ep,
				viper.GetString("minio_access_key"),
				viper.GetString("minio_secret_key"),
				viper.GetBool("minio_use_ssl"),
				urlTTL,
				viper.GetString("minio_public_url"),
			)
			if err != nil {
				pool.Close()
				log.Error("failed to connect to MinIO", "err", err)
				os.Exit(1)
			}
			gen := documents.GeneratorFunc(documents.GenerateAttestation)
			docSvc = domain.NewDocumentService(
				pgRepo.NewPolicyRepo(pool),
				pgRepo.NewDocumentRepo(pool),
				gen,
				store,
				urlTTL,
			)
			log.Info("document service enabled", "minio_endpoint", ep)
		} else {
			log.Warn("MINIO_ENDPOINT not set — document endpoints disabled")
		}

		// NATS JetStream event publisher (AC-3, AC-6)
		if natsURL := viper.GetString("nats_url"); natsURL != "" {
			pub, err := events.Connect(bgCtx, natsURL, viper.GetString("nats_ca_cert"), log)
			if err != nil {
				log.Warn("NATS unavailable — events disabled", "err", err)
			} else {
				svc.SetPublisher(pub)
				defer pub.Close()
			}
		}
	} else {
		svc = domain.NewPolicyService(pgRepo.NewNullPolicyRepo(), pgRepo.NewNullAuditLogRepo(), nil)
	}

	port := viper.GetString("port")
	addr := net.JoinHostPort("", port)

	srv := &http.Server{
		Addr:         addr,
		Handler:      api.NewRouter(log, svc, docSvc, jwtMw),
		ReadTimeout:  time.Duration(viper.GetInt("read_timeout_s")) * time.Second,
		WriteTimeout: time.Duration(viper.GetInt("write_timeout_s")) * time.Second,
	}

	go func() {
		log.Info("starting ktayl-policy-service", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	shutCtx, cancel := context.WithTimeout(bgCtx, 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Error("shutdown error", "err", err)
	}
	log.Info("server stopped")
}

func runMigrations(pool *pgxpool.Pool, log *slog.Logger) error {
	sqlDB := stdlib.OpenDBFromPool(pool)
	driver, err := pgx5.WithInstance(sqlDB, &pgx5.Config{})
	if err != nil {
		return err
	}

	src, err := iofs.New(db.Migrations, "migrations")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	log.Info("migrations applied")
	return nil
}
