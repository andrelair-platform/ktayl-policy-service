# ktayl-policy-service

[![CI](https://github.com/andrelair-platform/ktayl-policy-service/actions/workflows/ci.yml/badge.svg)](https://github.com/andrelair-platform/ktayl-policy-service/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-blue)](https://go.dev)
[![Supply chain: cosign](https://img.shields.io/badge/supply%20chain-cosign%20signed-green)](https://github.com/sigstore/cosign)

> Insurance policy lifecycle microservice for the **ktayl-solution IS** — a portfolio-grade simulation of a French insurance company's information system, running on a self-hosted 5-node Kubernetes platform. Manages the full contract lifecycle (creation → activation → suspension → termination) together with coverages and premium tracking, backed by PostgreSQL and deployed via ArgoCD GitOps.

**Live docs:** [andrelair-platform.github.io/ktayl-policy-service](https://andrelair-platform.github.io/ktayl-policy-service/)
**Platform docs:** [andrelair-platform.github.io/minicloud-platform-docs](https://andrelair-platform.github.io/minicloud-platform-docs/)

---

## Table of Contents

- [Domain model](#domain-model)
- [Architecture](#architecture)
- [Getting Started](#getting-started)
- [CI/CD Pipeline](#cicd-pipeline)
- [Endpoints](#endpoints)
- [Environment variables](#environment-variables)
- [Database migrations](#database-migrations)
- [Contributing](#contributing)
- [License](#license)

---

## Domain model

All monetary values are stored as **eurocents** (`int64`) to avoid floating-point errors.

```
POLICE (Policy)         1 ─── n   GARANTIE (Coverage)
  id (UUID)                          id (UUID)
  policy_number (unique)             policy_id → Policy.id
  holder_name                        type
  product_code                       insured_amount (eurocents)
  status (draft|active|              deductible (eurocents)
          suspended|terminated)
  effective_date
  expiry_date             1 ─── n   PRIME (Premium)
                                       id (UUID)
                                       policy_id → Policy.id
                                       amount (eurocents)
                                       frequency (monthly|quarterly|annual)
                                       due_date
                                       paid_at (nullable)
```

Full Merise MCD → MLD → MPD diagrams with SQL schema: [docs/data-model](https://andrelair-platform.github.io/ktayl-policy-service/data-model/mcd)

---

## Architecture

```
┌─────────────────────────────────────────────┐
│             GitHub Actions CI               │
│  govulncheck + gosec → go test (70% cov)   │
│  → Docker build → Trivy scan               │
│  → cosign sign → syft SBOM → gitops bump   │
└────────────────────┬────────────────────────┘
                     │ webhook
┌────────────────────▼────────────────────────┐
│                  ArgoCD                     │
│   watches services/ktayl-policy-service/    │
│   minicloud-1/dev  (Kustomize overlay)      │
└────────────────────┬────────────────────────┘
                     │
┌────────────────────▼────────────────────────┐
│        ktayl-policy-service Pod (k3s)       │
│  Go 1.25 · chi v5 · pgx/v5 · PostgreSQL    │
│  distroless/static-debian12:nonroot         │
└─────────────────────────────────────────────┘
```

| Component | Detail |
|---|---|
| Runtime | Go 1.25, `distroless/static-debian12:nonroot` |
| Router | `go-chi/chi v5` |
| Database | PostgreSQL via `jackc/pgx v5` (pgxpool) |
| Registry | `harbor.10.0.0.200.nip.io/library/ktayl-policy-service` |
| GitOps | ArgoCD — Kustomize overlay in `minicloud-gitops` |
| Sprint | CERT-1 (CdCF §4 — Policy Management) |

---

## Getting Started

### Prerequisites

| Tool | Version |
|---|---|
| Go | ≥ 1.25 |
| Docker | any recent |
| PostgreSQL | ≥ 15 (for local integration tests) |

### Run locally

```bash
git clone https://github.com/andrelair-platform/ktayl-policy-service.git
cd ktayl-policy-service

PORT=8080 DATABASE_URL="postgres://user:pass@localhost:5432/policydb" make run

# Health check
curl http://localhost:8080/healthz
```

### Test

```bash
make test        # unit tests with race detector
make test-cov    # unit tests + coverage gate (≥ 70% on domain + api)
make vuln        # govulncheck — stdlib + dependency CVE scan
make sec         # gosec — high-severity static analysis
```

### Build

```bash
make build        # linux/amd64 binary → bin/ktayl-policy-service
make build-image  # distroless container image (requires Harbor on Tailscale)
```

---

## CI/CD Pipeline

Every push to `dev`, `staging`, or `main` triggers `.github/workflows/ci.yml`:

```
push
 │
 ├─ 1. go test -race -count=1 ./...
 ├─ 2. Coverage gate — ≥ 70% on internal/domain + internal/api
 ├─ 3. govulncheck — fails on reachable CVEs in stdlib or deps
 ├─ 4. gosec — fails on HIGH severity findings
 ├─ 5. Connect to Tailscale → trust minicloud CA
 ├─ 6. docker build → push to Harbor (distroless, linux/amd64)
 ├─ 7. Trivy scan — fails on unfixed CRITICAL CVEs
 ├─ 8. cosign sign (keyless, staging + main only)
 ├─ 9. syft SBOM CycloneDX JSON (main only)
 └─10. kustomize bump dev overlay (dev branch, pending S010)
```

**Branch strategy:**
- `dev` — direct push, CI builds `dev-<sha>` image
- `staging` — PR required, cosign-signed image
- `main` — PR + GPG required, cosign + SBOM

### Required secrets

All 7 secrets are **org-level** on `andrelair-platform` — new repos inherit them automatically.

| Secret | Purpose |
|---|---|
| `TS_OAUTH_CLIENT_ID` | Tailscale OAuth — joins tailnet as `tag:ci` |
| `TS_OAUTH_SECRET` | Tailscale OAuth secret |
| `MINICLOUD_CA_CERT` | Self-signed CA PEM — trusts Harbor TLS |
| `HARBOR_USER` | Harbor registry username |
| `HARBOR_PASSWORD` | Harbor registry password |
| `GITOPS_TOKEN` | GitHub PAT for committing to `minicloud-gitops` |
| `GPG_PRIVATE_KEY` | Armored GPG key for signing gitops commits |

---

## Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/healthz` | Liveness probe — `{"status":"ok"}` |

Full REST API (policies CRUD, coverages, premiums) added in S003.

---

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Listen port |
| `READ_TIMEOUT_S` | `10` | HTTP read timeout (seconds) |
| `WRITE_TIMEOUT_S` | `10` | HTTP write timeout (seconds) |
| `DATABASE_URL` | — | PostgreSQL DSN (required from S003) |

---

## Database migrations

SQL migrations in `db/migrations/` follow the Flyway naming convention (`V1__description.sql`). Applied automatically on startup from S004.

| Migration | Description |
|---|---|
| `V1__init_schema.sql` | `policies`, `coverages`, `premiums` tables + indexes + check constraints |

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for branch conventions, commit style, and PR requirements.

---

## License

[MIT](LICENSE) © andrelair-platform
