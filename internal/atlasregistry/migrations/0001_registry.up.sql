CREATE TABLE IF NOT EXISTS registry_source_capabilities (
  source_id VARCHAR(128) NOT NULL,
  capability_key VARCHAR(128) NOT NULL,
  capability_kind ENUM('FACTS', 'MEDIA') NOT NULL,
  league ENUM('NBA', 'NFL', 'MLB', 'NHL') NULL,
  fact_groups JSON NOT NULL,
  media_kinds JSON NOT NULL,
  official_authority BOOLEAN NOT NULL,
  independence_key VARCHAR(128) NOT NULL,
  allow_website BOOLEAN NOT NULL,
  allow_repository BOOLEAN NOT NULL,
  allow_public_api BOOLEAN NOT NULL,
  policy_source_url VARCHAR(2048) NOT NULL,
  reviewed_by VARCHAR(160) NOT NULL,
  reviewed_at DATETIME(6) NOT NULL,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  PRIMARY KEY (source_id, capability_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS registry_runs (
  run_id VARCHAR(128) NOT NULL PRIMARY KEY,
  state ENUM('OPEN', 'SEALED', 'ABANDONED') NOT NULL DEFAULT 'OPEN',
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  sealed_at DATETIME(6) NULL,
  abandoned_at DATETIME(6) NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS registry_fact_batches (
  batch_id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  run_id VARCHAR(128) NOT NULL,
  source_id VARCHAR(128) NOT NULL,
  capability_key VARCHAR(128) NOT NULL,
  group_kind ENUM('IDENTITY', 'VENUE', 'LEADER', 'ROSTER') NOT NULL,
  league ENUM('NBA', 'NFL', 'MLB', 'NHL') NOT NULL,
  season VARCHAR(40) NOT NULL,
  source_url VARCHAR(2048) NOT NULL,
  fetched_at DATETIME(6) NOT NULL,
  content_hash CHAR(64) NOT NULL,
  payload_hash CHAR(64) NOT NULL,
  complete_pagination BOOLEAN NOT NULL,
  payload JSON NOT NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  UNIQUE KEY uq_registry_fact_revision (run_id, source_id, capability_key, group_kind, league, season, content_hash),
  KEY ix_registry_fact_run (run_id, group_kind, league),
  CONSTRAINT fk_registry_fact_run FOREIGN KEY (run_id) REFERENCES registry_runs (run_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS registry_media_batches (
  batch_id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  run_id VARCHAR(128) NOT NULL,
  source_id VARCHAR(128) NOT NULL,
  capability_key VARCHAR(128) NOT NULL,
  asset_id VARCHAR(128) NOT NULL,
  media_kind ENUM('LOGO', 'VENUE_PHOTO', 'PLAYER_PHOTO') NOT NULL,
  entity_id VARCHAR(64) NOT NULL,
  content_hash CHAR(64) NOT NULL,
  payload_hash CHAR(64) NOT NULL,
  payload JSON NOT NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  UNIQUE KEY uq_registry_media_revision (run_id, source_id, capability_key, asset_id, content_hash),
  KEY ix_registry_media_run (run_id, media_kind),
  CONSTRAINT fk_registry_media_run FOREIGN KEY (run_id) REFERENCES registry_runs (run_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
