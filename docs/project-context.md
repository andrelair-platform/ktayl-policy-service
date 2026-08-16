# Project Context — ktayl-policy-service

BMAD agents load this file as persistent context on activation.

## What this service is

Go microservice managing the full lifecycle of insurance policies for **ktayl-solution IS** — a simulated French insurance company (modeled after HDI France). Part of the RNCP39583 CERT-1 certification portfolio.

## Platform

Self-hosted 5-node k3s cluster (minicloud). All services deployed via ArgoCD app-of-apps from `minicloud-gitops`. Docker images pushed to Harbor registry at `harbor.10.0.0.200.nip.io`. Secrets managed by Vault + ESO.

## Tech stack (decided, do not re-propose)

- **Language:** Go 1.25
- **Router:** chi v5 (net/http compatible)
- **Database:** PostgreSQL 16 via pgx/v5 + golang-migrate (Flyway naming V{N}__desc.sql)
- **Events:** NATS JetStream + CloudEvents 1.0 (`github.com/cloudevents/sdk-go/v2`)
- **Auth:** Authentik M2M JWT (`github.com/golang-jwt/jwt/v5` + `github.com/MicahParks/keyfunc/v3`)
- **Documents:** MinIO via `github.com/minio/minio-go/v7`, PDF via `github.com/go-pdf/fpdf` (no CGO)
- **Container:** distroless/static-debian12:nonroot (CGO_ENABLED=0)
- **CI:** GitHub Actions — Tailscale → Harbor push, cosign staging/main

## Domain model (implemented, S002 Done)

- `Policy`: id, policy_number, holder_name, product_code, status, effective_date, expiry_date
- `Coverage`: type, insured_amount (eurocents), deductible (eurocents)
- `Premium`: amount (eurocents), frequency (monthly/quarterly/annual), due_date, paid_at
- Status machine: DRAFT → SUBMITTED → ACTIVE → AMENDED → CANCELLED / EXPIRED / REJECTED
- Monetary amounts: **always integer eurocents (int64)**, never float

## Compliance constraints

- **DORA Art.9**: every state transition writes an immutable `policy_audit_log` row (policy_id, from_status, to_status, actor_id, reason, occurred_at)
- **ACPR Art.L113-5**: attestation PDF on demand for ACTIVE policies; 7-year MinIO retention

## Sprint state (M1-M2)

- S001 ✅ Done, S002 ✅ Done
- S003 🔵 In Progress (CRUD API — next story)
- S004–S010 🔵 Ready

## Related artefacts

- PRD: `_bmad-output/planning-artifacts/prd.md`
- Architecture: `_bmad-output/planning-artifacts/architecture.md`
- Epics: `_bmad-output/planning-artifacts/epics.md`
- IMPL-NOTES: `minicloud-gitops/bmad/stories/cert-1/m1-m2/IMPL-NOTES.md`
- CdCF: `minicloud-platform-docs/docs/certification/01-cahier-des-charges-fonctionnel.md`
