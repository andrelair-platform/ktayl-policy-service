CREATE TABLE IF NOT EXISTS policy_documents (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id   UUID        NOT NULL REFERENCES policies(id) ON DELETE CASCADE,
    type        TEXT        NOT NULL DEFAULT 'attestation',
    minio_key   TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_policy_documents_policy_id ON policy_documents(policy_id, created_at DESC);
