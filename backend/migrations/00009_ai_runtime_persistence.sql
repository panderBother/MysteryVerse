-- +goose Up
CREATE TABLE mv_ai.agent_runs (
  id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  parent_run_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
  recovery_source_run_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
  tenant_id VARCHAR(64) NULL,
  task_type VARCHAR(32) NOT NULL,
  workflow_key VARCHAR(64) NULL,
  workflow_version VARCHAR(32) NULL,
  script_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
  script_version_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
  room_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
  game_session_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
  actor_type VARCHAR(20) NOT NULL,
  actor_id VARCHAR(128) NULL,
  status VARCHAR(24) NOT NULL,
  execution_version BIGINT UNSIGNED NOT NULL DEFAULT 0,
  current_step_no INT UNSIGNED NOT NULL DEFAULT 0,
  owner_id VARCHAR(128) NULL,
  fencing_token BIGINT UNSIGNED NOT NULL DEFAULT 0,
  owner_lease_expires_at DATETIME(3) NULL,
  retry_count SMALLINT UNSIGNED NOT NULL DEFAULT 0,
  runtime_deadline_at DATETIME(3) NULL,
  trace_id VARCHAR(64) NOT NULL,
  correlation_id VARCHAR(64) NULL,
  schema_version VARCHAR(32) NOT NULL,
  started_at DATETIME(3) NULL,
  finished_at DATETIME(3) NULL,
  cancelled_at DATETIME(3) NULL,
  failure_category VARCHAR(64) NULL,
  failure_summary VARCHAR(500) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
    ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_agent_runs_status_deadline (status, runtime_deadline_at),
  KEY idx_agent_runs_owner (owner_id, fencing_token),
  CONSTRAINT fk_agent_runs_parent
    FOREIGN KEY (parent_run_id) REFERENCES mv_ai.agent_runs(id),
  CONSTRAINT fk_agent_runs_recovery_source
    FOREIGN KEY (recovery_source_run_id) REFERENCES mv_ai.agent_runs(id),
  CONSTRAINT ck_agent_runs_fencing_owner CHECK (
    (owner_id IS NULL AND owner_lease_expires_at IS NULL)
    OR
    (
      owner_id IS NOT NULL
      AND fencing_token > 0
      AND owner_lease_expires_at IS NOT NULL
    )
  )
) ENGINE=InnoDB;

CREATE TABLE mv_ai.agent_idempotency_records (
  id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  idempotency_scope VARCHAR(160) NOT NULL,
  idempotency_key VARCHAR(128) NOT NULL,
  request_hash BINARY(32) NOT NULL,
  run_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  expires_at DATETIME(3) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_agent_idempotency_scope_key (idempotency_scope, idempotency_key),
  KEY idx_agent_idempotency_expiry (expires_at),
  CONSTRAINT fk_agent_idempotency_run
    FOREIGN KEY (run_id) REFERENCES mv_ai.agent_runs(id)
) ENGINE=InnoDB;

CREATE TABLE mv_ai.agent_steps (
  id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  agent_run_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  step_no INT UNSIGNED NOT NULL,
  step_type VARCHAR(24) NOT NULL,
  node_key VARCHAR(96) NULL,
  status VARCHAR(20) NOT NULL,
  input_reference VARCHAR(1024) NULL,
  output_reference VARCHAR(1024) NULL,
  input_digest BINARY(32) NULL,
  output_digest BINARY(32) NULL,
  schema_version VARCHAR(32) NOT NULL,
  attempt_count SMALLINT UNSIGNED NOT NULL DEFAULT 0,
  started_at DATETIME(3) NULL,
  finished_at DATETIME(3) NULL,
  failure_category VARCHAR(64) NULL,
  failure_summary VARCHAR(500) NULL,
  execution_metadata JSON NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
    ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_agent_steps_run_no (agent_run_id, step_no),
  KEY idx_agent_steps_status (agent_run_id, status),
  CONSTRAINT fk_agent_steps_run
    FOREIGN KEY (agent_run_id) REFERENCES mv_ai.agent_runs(id)
) ENGINE=InnoDB;

CREATE TABLE mv_ai.agent_step_attempts (
  id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  agent_step_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  attempt_no SMALLINT UNSIGNED NOT NULL,
  status VARCHAR(20) NOT NULL,
  input_digest BINARY(32) NULL,
  output_digest BINARY(32) NULL,
  started_at DATETIME(3) NOT NULL,
  finished_at DATETIME(3) NULL,
  failure_category VARCHAR(64) NULL,
  failure_summary VARCHAR(500) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_agent_step_attempts_no (agent_step_id, attempt_no),
  CONSTRAINT fk_agent_step_attempts_step
    FOREIGN KEY (agent_step_id) REFERENCES mv_ai.agent_steps(id)
) ENGINE=InnoDB;

CREATE TABLE mv_ai.agent_checkpoints (
  id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  agent_run_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  thread_id VARCHAR(128) NULL,
  checkpoint_seq INT UNSIGNED NOT NULL,
  schema_version VARCHAR(32) NOT NULL,
  runtime_version VARCHAR(32) NOT NULL,
  status VARCHAR(20) NOT NULL,
  storage_type VARCHAR(24) NOT NULL DEFAULT 'mysql_blob',
  storage_ref VARCHAR(1024) NULL,
  compression VARCHAR(16) NOT NULL DEFAULT 'zlib',
  encrypted_blob MEDIUMBLOB NULL,
  blob_size_bytes INT UNSIGNED NOT NULL,
  key_id VARCHAR(128) NOT NULL,
  nonce VARBINARY(12) NOT NULL,
  state_hash BINARY(32) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  expires_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_agent_checkpoints_seq (agent_run_id, checkpoint_seq),
  KEY idx_agent_checkpoints_expiry (expires_at),
  CONSTRAINT fk_agent_checkpoints_run
    FOREIGN KEY (agent_run_id) REFERENCES mv_ai.agent_runs(id),
  CONSTRAINT ck_agent_checkpoints_storage CHECK (
    (storage_type = 'mysql_blob' AND encrypted_blob IS NOT NULL AND storage_ref IS NULL)
    OR
    (storage_type = 'object_storage' AND encrypted_blob IS NULL AND storage_ref IS NOT NULL)
  )
) ENGINE=InnoDB;

CREATE TABLE mv_ai.agent_recovery_attempts (
  id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  agent_run_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  attempt_no SMALLINT UNSIGNED NOT NULL,
  trigger_category VARCHAR(64) NOT NULL,
  recovery_checkpoint_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  previous_owner_id VARCHAR(128) NULL,
  new_owner_id VARCHAR(128) NOT NULL,
  fencing_token BIGINT UNSIGNED NOT NULL,
  status VARCHAR(20) NOT NULL,
  started_at DATETIME(3) NOT NULL,
  finished_at DATETIME(3) NULL,
  failure_category VARCHAR(64) NULL,
  failure_summary VARCHAR(500) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_agent_recovery_attempts_no (agent_run_id, attempt_no),
  KEY idx_agent_recovery_attempts_status (status, started_at),
  CONSTRAINT fk_agent_recovery_attempts_run
    FOREIGN KEY (agent_run_id) REFERENCES mv_ai.agent_runs(id),
  CONSTRAINT fk_agent_recovery_attempts_checkpoint
    FOREIGN KEY (recovery_checkpoint_id) REFERENCES mv_ai.agent_checkpoints(id)
) ENGINE=InnoDB;

CREATE TABLE mv_ai.runtime_audit_records (
  id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  agent_run_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
  agent_step_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
  recovery_attempt_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
  tenant_id VARCHAR(64) NULL,
  event_category VARCHAR(64) NOT NULL,
  actor_type VARCHAR(32) NULL,
  actor_id VARCHAR(128) NULL,
  result VARCHAR(32) NOT NULL,
  failure_category VARCHAR(64) NULL,
  correlation_id VARCHAR(64) NULL,
  detail_summary VARCHAR(1000) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  expires_at DATETIME(3) NOT NULL,
  PRIMARY KEY (id),
  KEY idx_runtime_audit_run (agent_run_id, created_at),
  KEY idx_runtime_audit_expiry (expires_at)
) ENGINE=InnoDB;

CREATE TABLE mv_ai.runtime_cleanup_jobs (
  id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  job_type VARCHAR(32) NOT NULL,
  resource_type VARCHAR(32) NOT NULL,
  resource_id VARCHAR(128) NOT NULL,
  status VARCHAR(20) NOT NULL,
  attempt SMALLINT UNSIGNED NOT NULL DEFAULT 1,
  started_at DATETIME(3) NOT NULL,
  finished_at DATETIME(3) NULL,
  failure_category VARCHAR(64) NULL,
  failure_summary VARCHAR(500) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_runtime_cleanup_status (status, started_at),
  KEY idx_runtime_cleanup_resource (resource_type, resource_id)
) ENGINE=InnoDB;

-- +goose Down
DROP TABLE IF EXISTS mv_ai.runtime_cleanup_jobs;
DROP TABLE IF EXISTS mv_ai.runtime_audit_records;
DROP TABLE IF EXISTS mv_ai.agent_recovery_attempts;
DROP TABLE IF EXISTS mv_ai.agent_checkpoints;
DROP TABLE IF EXISTS mv_ai.agent_step_attempts;
DROP TABLE IF EXISTS mv_ai.agent_steps;
DROP TABLE IF EXISTS mv_ai.agent_idempotency_records;
DROP TABLE IF EXISTS mv_ai.agent_runs;
