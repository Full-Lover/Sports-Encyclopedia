CREATE TABLE IF NOT EXISTS publication_snapshots (
  snapshot_id VARCHAR(64) NOT NULL PRIMARY KEY,
  run_id VARCHAR(128) NOT NULL,
  candidate_hash CHAR(64) NOT NULL UNIQUE,
  base_snapshot_id VARCHAR(64) NULL,
  base_registry_token VARCHAR(255) NULL,
  next_registry_token VARCHAR(255) NOT NULL,
  profile ENUM('PREVIEW', 'V1') NOT NULL,
  schema_version INT NOT NULL,
  map_document JSON NOT NULL,
  status ENUM('CANDIDATE', 'PUBLISHED') NOT NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  published_at DATETIME(6) NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS publication_active_pointer (
  singleton_id TINYINT NOT NULL PRIMARY KEY,
  snapshot_id VARCHAR(64) NULL,
  registry_token VARCHAR(255) NULL,
  profile ENUM('PREVIEW', 'V1') NULL,
  schema_version INT NULL,
  CONSTRAINT chk_publication_pointer_singleton CHECK (singleton_id = 1)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS publication_lease (
  singleton_id TINYINT NOT NULL PRIMARY KEY,
  run_id VARCHAR(128) NULL,
  token CHAR(64) NULL,
  expires_at DATETIME(6) NULL,
  CONSTRAINT chk_publication_lease_singleton CHECK (singleton_id = 1)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS publication_receipts (
  run_id VARCHAR(128) NOT NULL PRIMARY KEY,
  snapshot_id VARCHAR(64) NOT NULL,
  candidate_hash CHAR(64) NOT NULL UNIQUE,
  committed_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS publication_schema_gate (
  singleton_id TINYINT NOT NULL PRIMARY KEY,
  allowed_version INT NOT NULL,
  paused BOOLEAN NOT NULL DEFAULT FALSE,
  CONSTRAINT chk_publication_gate_singleton CHECK (singleton_id = 1)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO publication_active_pointer (singleton_id) VALUES (1);
INSERT IGNORE INTO publication_lease (singleton_id) VALUES (1);
INSERT IGNORE INTO publication_schema_gate (singleton_id, allowed_version, paused) VALUES (1, 1, FALSE);
