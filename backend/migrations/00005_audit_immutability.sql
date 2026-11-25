-- +goose Up

CREATE TRIGGER trg_audit_logs_append_only_update
BEFORE UPDATE ON audit_logs
FOR EACH ROW
SIGNAL SQLSTATE '45000'
SET MESSAGE_TEXT = 'audit_logs is append-only';

CREATE TRIGGER trg_audit_logs_append_only_delete
BEFORE DELETE ON audit_logs
FOR EACH ROW
SIGNAL SQLSTATE '45000'
SET MESSAGE_TEXT = 'audit_logs is append-only';

-- +goose Down

DROP TRIGGER IF EXISTS trg_audit_logs_append_only_delete;
DROP TRIGGER IF EXISTS trg_audit_logs_append_only_update;
