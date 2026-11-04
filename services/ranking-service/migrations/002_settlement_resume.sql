USE activity_ranking;
SET NAMES utf8mb4;
SET time_zone = '+00:00';

ALTER TABLE ranking_settlement
  ADD COLUMN cursor_json JSON NULL,
  ADD COLUMN lease_owner VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
  ADD COLUMN lease_until DATETIME(6) NULL,
  ADD COLUMN retry_count INT UNSIGNED NOT NULL DEFAULT 0,
  ADD COLUMN next_retry_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  ADD COLUMN last_error VARCHAR(1024) NULL,
  ADD KEY idx_settlement_claim (status, next_retry_at, lease_until);
