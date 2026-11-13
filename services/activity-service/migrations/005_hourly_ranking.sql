USE activity_core;
SET NAMES utf8mb4;
SET time_zone = '+00:00';

ALTER TABLE activity_edition ADD COLUMN hourly_ranking_enabled BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE hourly_ranking_window (
    edition_id BIGINT UNSIGNED NOT NULL,
    cutoff_sec BIGINT NOT NULL,
    shard_no TINYINT UNSIGNED NOT NULL,
    sealed_at DATETIME(6) NULL,
    PRIMARY KEY (edition_id, cutoff_sec, shard_no),
    CHECK (shard_no < 4)
) ENGINE=InnoDB;

CREATE TABLE hourly_ranking_sequence (
    edition_id BIGINT UNSIGNED NOT NULL,
    cutoff_sec BIGINT NOT NULL,
    shard_no TINYINT UNSIGNED NOT NULL,
    lane_no TINYINT UNSIGNED NOT NULL,
    last_seq BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (edition_id, cutoff_sec, shard_no, lane_no),
    CHECK (shard_no < 4 AND lane_no < 16 AND last_seq >= 0)
) ENGINE=InnoDB;

ALTER TABLE outbox_event
    ADD COLUMN hourly_edition_id BIGINT UNSIGNED NULL,
    ADD COLUMN hourly_cutoff_sec BIGINT NULL,
    ADD COLUMN hourly_shard TINYINT UNSIGNED NULL,
    ADD COLUMN hourly_lane TINYINT UNSIGNED NULL,
    ADD COLUMN hourly_seq BIGINT NULL,
    ADD UNIQUE KEY uk_hourly_fact (hourly_edition_id, hourly_cutoff_sec, hourly_shard, hourly_lane, hourly_seq);

ALTER TABLE reward_grant DROP CHECK reward_grant_chk_2, DROP CHECK reward_grant_chk_3,
    ADD CONSTRAINT chk_reward_source_type CHECK (source_type IN ('DAY_DRAW', 'NIGHT_DRAW', 'PREVIEW', 'CHEST', 'STAGE', 'RANKING', 'HOURLY_RANKING')),
    ADD CONSTRAINT chk_reward_source_draw CHECK ((source_type IN ('DAY_DRAW', 'NIGHT_DRAW', 'PREVIEW') AND draw_id IS NOT NULL AND draw_no IS NOT NULL AND draw_no BETWEEN 1 AND 180)
        OR (source_type IN ('CHEST', 'STAGE', 'RANKING', 'HOURLY_RANKING') AND draw_id IS NULL AND draw_no IS NULL));

CREATE TABLE hourly_ranking_grant (
    settlement_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    edition_id BIGINT UNSIGNED NOT NULL,
    cutoff_sec BIGINT NOT NULL,
    final_rank SMALLINT UNSIGNED NOT NULL,
    grant_id BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (settlement_id, user_id),
    UNIQUE KEY uk_hourly_grant_window_user (edition_id, cutoff_sec, user_id),
    CHECK (final_rank BETWEEN 1 AND 100)
) ENGINE=InnoDB;
