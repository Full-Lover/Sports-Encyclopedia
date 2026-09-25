CREATE TABLE IF NOT EXISTS registry_baselines (
  baseline_token CHAR(64) NOT NULL PRIMARY KEY,
  originating_run_id VARCHAR(128) NOT NULL,
  content JSON NOT NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  CONSTRAINT fk_registry_baseline_run FOREIGN KEY (originating_run_id) REFERENCES registry_runs (run_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS registry_compilations (
  compilation_id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  run_id VARCHAR(128) NOT NULL,
  baseline_token CHAR(64) NOT NULL DEFAULT '',
  profile ENUM('PREVIEW', 'V1') NOT NULL,
  configuration_fingerprint CHAR(64) NOT NULL,
  next_baseline_token CHAR(64) NOT NULL,
  requested_at DATETIME(6) NOT NULL,
  compilation JSON NOT NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  UNIQUE KEY uq_registry_compilation_request (run_id, baseline_token, profile, configuration_fingerprint),
  UNIQUE KEY uq_registry_compilation_token (next_baseline_token),
  CONSTRAINT fk_registry_compilation_run FOREIGN KEY (run_id) REFERENCES registry_runs (run_id),
  CONSTRAINT fk_registry_compilation_baseline FOREIGN KEY (next_baseline_token) REFERENCES registry_baselines (baseline_token)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS registry_slug_history (
  slug VARCHAR(160) NOT NULL PRIMARY KEY,
  team_id VARCHAR(64) NOT NULL,
  first_run_id VARCHAR(128) NOT NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  KEY ix_registry_slug_team (team_id),
  CONSTRAINT fk_registry_slug_run FOREIGN KEY (first_run_id) REFERENCES registry_runs (run_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
