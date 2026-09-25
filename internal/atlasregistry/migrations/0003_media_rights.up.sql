CREATE TABLE IF NOT EXISTS registry_media_rights_decisions (
  source_id VARCHAR(128) NOT NULL,
  capability_key VARCHAR(128) NOT NULL,
  asset_id VARCHAR(128) NOT NULL,
  content_hash CHAR(64) NOT NULL,
  media_kind ENUM('LOGO', 'VENUE_PHOTO', 'PLAYER_PHOTO') NOT NULL,
  entity_id VARCHAR(64) NOT NULL,
  file_url VARCHAR(2048) NOT NULL,
  source_page_url VARCHAR(2048) NOT NULL,
  selected_rights JSON NOT NULL,
  reviewed_by VARCHAR(160) NOT NULL,
  reviewed_at DATETIME(6) NOT NULL,
  active BOOLEAN NOT NULL DEFAULT FALSE,
  PRIMARY KEY (source_id, capability_key, asset_id, content_hash),
  CONSTRAINT fk_registry_media_rights_policy FOREIGN KEY (source_id, capability_key)
    REFERENCES registry_source_capabilities (source_id, capability_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
