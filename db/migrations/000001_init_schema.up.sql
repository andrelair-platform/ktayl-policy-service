CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE policies (
    id             UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    policy_number  TEXT        NOT NULL UNIQUE,
    holder_name    TEXT        NOT NULL,
    product_code   TEXT        NOT NULL,
    status         TEXT        NOT NULL DEFAULT 'draft',
    effective_date TIMESTAMPTZ NOT NULL,
    expiry_date    TIMESTAMPTZ NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_policy_dates  CHECK (effective_date < expiry_date),
    CONSTRAINT chk_policy_status CHECK (status IN ('draft','active','suspended','terminated'))
);

CREATE TABLE coverages (
    id             UUID   PRIMARY KEY DEFAULT uuid_generate_v4(),
    policy_id      UUID   NOT NULL REFERENCES policies(id) ON DELETE CASCADE,
    type           TEXT   NOT NULL,
    insured_amount BIGINT NOT NULL,
    deductible     BIGINT NOT NULL DEFAULT 0,
    CONSTRAINT chk_insured_amount CHECK (insured_amount > 0),
    CONSTRAINT chk_deductible     CHECK (deductible >= 0)
);

CREATE TABLE premiums (
    id        UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    policy_id UUID        NOT NULL REFERENCES policies(id) ON DELETE CASCADE,
    amount    BIGINT      NOT NULL,
    frequency TEXT        NOT NULL,
    due_date  TIMESTAMPTZ NOT NULL,
    paid_at   TIMESTAMPTZ,
    CONSTRAINT chk_premium_amount    CHECK (amount > 0),
    CONSTRAINT chk_premium_frequency CHECK (frequency IN ('monthly','quarterly','annual'))
);

CREATE INDEX idx_policies_status        ON policies(status);
CREATE INDEX idx_policies_policy_number ON policies(policy_number);
CREATE INDEX idx_coverages_policy_id    ON coverages(policy_id);
CREATE INDEX idx_premiums_policy_id     ON premiums(policy_id);
CREATE INDEX idx_premiums_due_date      ON premiums(due_date);
