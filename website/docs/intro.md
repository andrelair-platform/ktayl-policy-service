---
id: intro
title: Overview
sidebar_label: Overview
slug: /
---

# ktayl Policy Service

Go microservice managing the **full lifecycle of insurance policies** for the ktayl-solution IS (CdCF §4 — Policy Management). Part of CERT-1 sprint.

## Responsibility

| In scope | Out of scope |
|---|---|
| Policy creation, activation, suspension, termination | Claims handling (ktayl-claims-service) |
| Coverage management (type, insured amount, deductible) | Premium billing / dunning (ERPNext) |
| Premium scheduling and payment tracking | Document generation (Paperless-ngx) |

## Stack

| Concern | Choice |
|---|---|
| Language | Go 1.25 |
| HTTP router | chi v5 |
| Database | PostgreSQL via pgx/v5 + pgxpool |
| Config | viper (env vars) |
| Container | distroless/static-debian12:nonroot |
| Registry | harbor.10.0.0.200.nip.io |

## Key endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/healthz` | Liveness probe |

Full REST API (CRUD on policies, coverages, premiums) — S003.

## Sprint map

| Story | Status | Description |
|---|---|---|
| S001 | Done | Go scaffold — chi router, /healthz, CI pipeline |
| S002 | Done | Domain model — Policy, Coverage, Premium + PostgreSQL repository |
| S003 | Upcoming | REST API — CRUD endpoints |
| S004 | Upcoming | Database migrations on startup |
| S005+ | Backlog | Events, auth, portal integration |

## Links

- [GitHub repository](https://github.com/andrelair-platform/ktayl-policy-service)
- [Platform documentation](https://andrelair-platform.github.io/minicloud-platform-docs/)
- [Business Applications Catalog](https://andrelair-platform.github.io/minicloud-platform-docs/insurance-platform/business-applications-catalog)
