USE activity_ranking;
SET NAMES utf8mb4;
SET time_zone = '+00:00';

CREATE TABLE hourly_settlement (
    settlement_id BIGINT UNSIGNED NOT NULL,
    edition_id BIGINT UNSIGNED NOT NULL,
    cutoff_sec BIGINT NOT NULL,
    previous_cutoff_sec BIGINT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING_BARRIER',
    watermark JSON NULL,
    lease_owner VARCHAR(64) NULL,
    lease_until DATETIME(6) NULL,
    next_retry_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    retry_count INT UNSIGNED NOT NULL DEFAULT 0,
    barrier_received_at DATETIME(6) NULL,
    frozen_at DATETIME(6) NULL,
    completed_at DATETIME(6) NULL,
    redis_retired_at DATETIME(6) NULL,
    PRIMARY KEY (settlement_id),
    UNIQUE KEY uk_hourly_settlement_window (edition_id, cutoff_sec),
    KEY idx_hourly_settlement_claim (status, next_retry_at, lease_until),
    KEY idx_hourly_settlement_retire (redis_retired_at, frozen_at),
    CHECK (status IN ('PENDING_BARRIER','WAITING_WATERMARK','AWARDING','COMPLETED'))
) ENGINE=InnoDB;

CREATE TABLE hourly_award (
    edition_id BIGINT UNSIGNED NOT NULL,
    cutoff_sec BIGINT NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    settlement_id BIGINT UNSIGNED NOT NULL,
    final_rank SMALLINT UNSIGNED NOT NULL,
    score BIGINT NOT NULL,
    reached_at_sec BIGINT NOT NULL,
    quantity SMALLINT UNSIGNED NOT NULL,
    grant_id BIGINT UNSIGNED NULL,
    delivered_at DATETIME(6) NULL,
    lease_owner VARCHAR(64) NULL,
    lease_until DATETIME(6) NULL,
    next_retry_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    retry_count INT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (edition_id, cutoff_sec, user_id),
    UNIQUE KEY uk_hourly_award_rank (settlement_id, final_rank),
    KEY idx_hourly_award_claim (delivered_at, next_retry_at, lease_until),
    CHECK (final_rank BETWEEN 1 AND 100 AND score >= 0 AND quantity > 0),
    CHECK ((grant_id IS NULL) = (delivered_at IS NULL))
) ENGINE=InnoDB;
