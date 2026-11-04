USE activity_core;
SET NAMES utf8mb4;
SET time_zone = '+00:00';

ALTER TABLE settlement_task
  ADD COLUMN next_retry_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) AFTER lease_until,
  ADD KEY idx_task_claim (status, next_retry_at, lease_until);

ALTER TABLE activity_edition
  ADD KEY idx_edition_settlement (settlement_status, ends_at);
