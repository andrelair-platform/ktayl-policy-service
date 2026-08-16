DROP INDEX IF EXISTS idx_audit_log_policy_id;
DROP TABLE IF EXISTS policy_audit_log;

ALTER TABLE policies DROP CONSTRAINT IF EXISTS policies_status_check;
ALTER TABLE policies ADD CONSTRAINT chk_policy_status
    CHECK (status IN ('draft','active','suspended','terminated'));
