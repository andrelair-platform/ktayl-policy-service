# ADR-001 — Database Layer: raw pgx + golang-migrate, no ORM

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Date** | 2026-08-17 |
| **Deciders** | AndreLiar (Tech Lead) |
| **Compliance context** | DORA Art.9 (audit traceability), ACPR Art.L113-5 |

---

## Context

`ktayl-policy-service` persists insurance policies, state transitions, and audit logs to PostgreSQL. Three database-layer options were evaluated:

| Option | Migration | Query layer |
|---|---|---|
| A | golang-migrate | raw pgx (chosen) |
| B | golang-migrate | sqlx |
| C | golang-migrate | GORM |

Flyway (JVM) was listed in the CdCF as the migration standard. It was substituted with `golang-migrate` — same versioned-SQL-file concept, Go-native, embedded in the binary via `embed.FS`. The SQL files (`V1__init_schema.sql`, `V2__audit_log.sql`, `V3__documents.sql`) are identical in format to what Flyway would consume.

---

## Decision

**Use `pgx/v5` with raw SQL constants. No ORM.**

---

## Rationale

### 1. Audit trail requires inspectable SQL (DORA Art.9)

DORA Art.9 requires every business decision to be traceable. The `policy_audit_log` table records who triggered each state transition, when, and why. Regulators and auditors must be able to map a log row back to the exact SQL that produced it.

With GORM, the query is generated at runtime:

```go
// GORM — you write this
db.Where("status = ? AND holder_id = ?", "ACTIVE", holderID).
   Preload("Coverages").Find(&policies)

// GORM emits something like this — but you have to enable debug logging to see it,
// and it can change across GORM versions
SELECT `policies`.* FROM `policies` WHERE status = 'ACTIVE' AND holder_id = '...'
SELECT `coverages`.* FROM `coverages` WHERE `coverages`.`policy_id` IN (...)
```

With raw pgx, the query is a named constant that never changes without a code review:

```go
// internal/repository/postgres/policy_postgres.go
const listActivePolicies = `
    SELECT id, policy_number, holder_name, product_code,
           status, effective_date, expiry_date, created_at
    FROM policies
    WHERE status = $1 AND holder_id = $2
    ORDER BY created_at DESC`

rows, err := r.pool.Query(ctx, listActivePolicies, domain.StatusActive, holderID)
```

Any auditor can read the exact predicate. It cannot silently change at runtime.

### 2. N+1 queries are invisible with ORMs

With GORM's `Has Many` relationship, forgetting `Preload` causes one query per row:

```go
// Naïve GORM — emits 1 + N queries
var policies []Policy
db.Find(&policies)
for _, p := range policies {
    fmt.Println(p.Coverages) // triggers SELECT per policy
}

// With Preload — GORM is smarter, but the behaviour depends on the ORM version
db.Preload("Coverages").Find(&policies)
```

With raw SQL, the JOIN is explicit and never changes behaviour unexpectedly:

```sql
SELECT p.id, p.policy_number, p.status,
       c.coverage_type, c.limit_amount, c.deductible
FROM   policies p
LEFT JOIN coverages c ON c.policy_id = p.id
WHERE  p.id = $1
```

### 3. Clean domain model (no DB annotation pollution)

GORM requires embedding `gorm.Model` and DB tags on every domain struct:

```go
// GORM — domain model is coupled to the database
type Policy struct {
    gorm.Model
    PolicyNumber string     `gorm:"uniqueIndex;not null"`
    Status       Status     `gorm:"type:policy_status"`
    HolderName   string     `gorm:"column:holder_name"`
    Coverages    []Coverage `gorm:"foreignKey:PolicyID;constraint:OnDelete:CASCADE"`
}
```

The domain model now carries database concerns. Adding a gRPC layer, an in-memory event store, or a test double becomes harder because the struct is entangled with GORM internals.

The actual domain model in `internal/domain/policy.go`:

```go
// Clean — no database awareness
type Policy struct {
    ID            uuid.UUID
    PolicyNumber  string
    HolderName    string
    ProductCode   string
    Status        Status
    EffectiveDate time.Time
    ExpiryDate    time.Time
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

The PostgreSQL repository (`internal/repository/postgres/policy_postgres.go`) owns the mapping between this struct and SQL. The domain does not know storage exists.

### 4. Repository interfaces already provide the ORM's main benefit

The primary reason teams reach for ORMs is testability — swapping the real DB for a mock in unit tests. The `PolicyRepository` interface gives this without GORM:

```go
// internal/repository/policy_repository.go
type PolicyRepository interface {
    Create(ctx context.Context, p *domain.Policy) error
    GetByID(ctx context.Context, id uuid.UUID) (*domain.Policy, error)
    List(ctx context.Context, f ListFilter) ([]*domain.Policy, string, error)
    UpdateStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, s domain.Status) error
    Delete(ctx context.Context, id uuid.UUID) error
}
```

Unit tests inject `MockPolicyRepository`. Integration tests inject `PostgresPolicyRepository`. The service layer (`internal/domain/policy_service.go`) is identical in both cases.

---

## Why not sqlx?

`sqlx` is a lightweight layer on top of `database/sql` that adds struct scanning. It keeps full SQL control but removes manual `rows.Scan()` boilerplate:

```go
// Raw pgx (current) — manual scan
rows, _ := r.pool.Query(ctx, getByID, id)
var p domain.Policy
for rows.Next() {
    rows.Scan(&p.ID, &p.PolicyNumber, &p.HolderName, &p.Status,
              &p.EffectiveDate, &p.ExpiryDate, &p.CreatedAt)
}

// sqlx equivalent — struct scan via db tags
var p domain.Policy
err := r.db.GetContext(ctx, &p, getByID, id)
```

`sqlx` would be a reasonable addition if the service grows beyond ~20 query functions. At M1-M2 scope (~8 queries), the boilerplate is ~40 lines — not enough to justify a dependency. Revisit at M6 if the query count exceeds 25.

---

## Migration tool: golang-migrate vs Flyway

| | Flyway | golang-migrate |
|---|---|---|
| Runtime | JVM (separate process or Docker) | Go-native, embedded in binary |
| SQL file format | `V1__description.sql` | `V1__description.up.sql` + `.down.sql` |
| CdCF compliance | Required | Equivalent (same versioning model) |
| Test integration | Needs Docker or sidecar | `embed.FS` — runs in-process in testcontainers suite |

`golang-migrate` satisfies the CdCF requirement for versioned SQL migrations. The SQL files are portable — they can be run through Flyway if the platform ever standardises on it.

---

## Consequences

**Positive:**
- SQL is fully auditable at code-review time — no generated queries
- Domain structs are database-agnostic
- No N+1 surprises possible
- Test doubles require only implementing the repository interface
- Binary stays small (no ORM reflection overhead at startup)

**Negative / trade-offs:**
- Manual `rows.Scan()` is verbose for wide structs (mitigated by pgx's `pgx.RowToStructByName` helper)
- No automatic schema-to-struct synchronisation — adding a column requires updating both the migration and the scan code
- No built-in soft-delete, optimistic locking, or relationship loading — these must be implemented explicitly (which is also the upside: they're explicit)

---

## References

- [pgx v5 documentation](https://pkg.go.dev/github.com/jackc/pgx/v5)
- [golang-migrate](https://github.com/golang-migrate/migrate)
- DORA Regulation (EU) 2022/2554 — Art.9 (ICT risk management, auditability)
- ACPR Art.L113-5 — document retention obligations (drives 7-year policy document lifecycle)
