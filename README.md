# ktayl-policy-service

Policy lifecycle microservice for the ktayl-solution insurance IS. Manages the full lifecycle of insurance contracts: creation, activation, suspension, and termination, along with coverages and premium tracking.

Part of the **CERT-1** sprint (CdCF §4 — Policy Management).

## Quick start

```bash
# Run locally (hot-reloads on save with `air` if installed, otherwise plain run)
make run

# Lint + test
make lint
make test-cov

# Build Linux binary
make build

# Build container image (requires Harbor on Tailscale)
make build-image
```

## Project layout

```
cmd/server/          # HTTP server entry point (main.go)
internal/
  api/               # chi router, middleware, HTTP handlers
  domain/            # core entities: Policy, Coverage, Premium + validation
  repository/        # storage interfaces
    postgres/        # PostgreSQL implementations (pgx/v5)
  events/            # domain event types (populated S005)
db/
  migrations/        # Flyway-compatible SQL — V1__init_schema.sql, ...
.github/workflows/
  ci.yml             # lint → test → build+push → Trivy → cosign → gitops bump
```

## Key commands

| Command | What it does |
|---|---|
| `make lint` | golangci-lint (errcheck, govet, staticcheck, revive…) |
| `make test` | unit tests with race detector |
| `make test-cov` | unit tests + coverage report (gate: 70% on internal/) |
| `make build` | cross-compile for linux/amd64, CGO disabled |
| `make build-image` | build distroless container image |
| `make run` | run locally on port 8080 |

## Endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/healthz` | Liveness probe — returns `{"status":"ok"}` |

Full REST API added in S003.

## Domain model

All monetary values are stored as **eurocents** (int64) to avoid floating-point errors.

```
Policy          1 ─── n   Coverage
  id (UUID)                 id (UUID)
  policy_number             policy_id → Policy.id
  holder_name               type (text)
  product_code              insured_amount (eurocents)
  status                    deductible (eurocents)
  effective_date
  expiry_date     1 ─── n   Premium
                              id (UUID)
                              policy_id → Policy.id
                              amount (eurocents)
                              frequency (monthly|quarterly|annual)
                              due_date
                              paid_at (nullable)
```

## CI pipeline

```
push/PR → lint (L0) → test 70% cov (L1) → build+push Harbor → Trivy CRITICAL scan
        → cosign sign (staging/main) → gitops bump (dev only, pending S010)
```

Branch strategy: `dev` — direct push · `staging` — PR required, cosign · `main` — PR + GPG, cosign + SBOM

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Listen port |
| `READ_TIMEOUT_S` | `10` | HTTP read timeout (seconds) |
| `WRITE_TIMEOUT_S` | `10` | HTTP write timeout (seconds) |
| `DATABASE_URL` | — | PostgreSQL DSN (added S002, required from S003) |

## Database migrations

Migrations are Flyway-compatible SQL files in `db/migrations/`. Run order is determined by the version prefix (`V1__`, `V2__`, …). Applied automatically on startup from S004.
