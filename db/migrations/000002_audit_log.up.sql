CREATE TABLE IF NOT EXISTS policy_audit_log (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id    UUID        NOT NULL REFERENCES policies(id) ON DELETE CASCADE,
    from_status  TEXT        NOT NULL,
    to_status    TEXT        NOT NULL,
    actor_id     TEXT        NOT NULL,
    reason       TEXT        NOT NULL DEFAULT '',
    occurred_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_log_policy_id ON policy_audit_log(policy_id, occurred_at DESC);

-- Allow statuses introduced in S004 on the policies table.
ALTER TABLE policies DROP CONSTRAINT IF EXISTS policies_status_check;
ALTER TABLE policies ADD CONSTRAINT policies_status_check
    CHECK (status IN ('draft','submitted','active','rejected','amended','cancelled','expired','suspended','terminated'));
