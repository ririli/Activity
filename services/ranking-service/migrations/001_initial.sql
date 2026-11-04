-- MySQL 8.4. Ranking service owns projections and final award manifests.
USE activity_ranking;
SET NAMES utf8mb4;
SET time_zone = '+00:00';

CREATE TABLE score_fact (
    draw_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    edition_id BIGINT UNSIGNED NOT NULL,
    draw_seq BIGINT UNSIGNED NOT NULL,
    consumed_petals BIGINT NOT NULL,
    occurred_at_sec BIGINT UNSIGNED NOT NULL COMMENT 'Business epoch seconds, not message receive time',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (draw_id),
    UNIQUE KEY uk_score_sequence (user_id, edition_id, draw_seq),
    KEY idx_score_reconcile (edition_id, draw_id),
    CHECK (draw_seq > 0 AND consumed_petals > 0)
) ENGINE=InnoDB;

CREATE TABLE ranking_entry (
    edition_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    score BIGINT NOT NULL DEFAULT 0,
    score_reached_at_sec BIGINT UNSIGNED NOT NULL COMMENT 'For zero score, use participation creation epoch seconds',
    last_contiguous_seq BIGINT UNSIGNED NOT NULL DEFAULT 0,
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (edition_id, user_id),
    KEY idx_ranking_order (edition_id, score DESC, score_reached_at_sec ASC, user_id ASC),
    CHECK (score >= 0)
) ENGINE=InnoDB COMMENT='Includes zero-score users; no stored mutable rank number';

CREATE TABLE ranking_settlement (
    edition_id BIGINT UNSIGNED NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    cutoff_manifest JSON NOT NULL COMMENT 'Core manifest after accepted transactions have drained',
    reward_rule_snapshot JSON NOT NULL,
    frozen_at DATETIME(6) NULL,
    completed_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (edition_id),
    CHECK (status IN ('PENDING', 'RECONCILING', 'FROZEN', 'AWARDING', 'COMPLETED', 'FAILED')),
    CHECK (status NOT IN ('FROZEN', 'AWARDING', 'COMPLETED') OR frozen_at IS NOT NULL),
    CHECK (status <> 'COMPLETED' OR completed_at IS NOT NULL)
) ENGINE=InnoDB;

CREATE TABLE ranking_snapshot (
    edition_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    final_rank BIGINT UNSIGNED NOT NULL,
    score BIGINT NOT NULL,
    score_reached_at_sec BIGINT UNSIGNED NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (edition_id, user_id),
    UNIQUE KEY uk_snapshot_rank (edition_id, final_rank),
    CHECK (final_rank > 0 AND score >= 0)
) ENGINE=InnoDB COMMENT='Immutable final full ranking snapshot';

CREATE TABLE ranking_award (
    award_id BIGINT UNSIGNED NOT NULL,
    edition_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    final_rank BIGINT UNSIGNED NOT NULL,
    reward_entry_no SMALLINT UNSIGNED NOT NULL,
    item_id BIGINT UNSIGNED NOT NULL,
    item_name VARCHAR(128) NOT NULL,
    quantity BIGINT NOT NULL,
    delivery_status VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    delivered_at DATETIME(6) NULL COMMENT 'Set after core service confirms idempotent grant',
    PRIMARY KEY (award_id),
    UNIQUE KEY uk_ranking_award (edition_id, user_id, reward_entry_no),
    KEY idx_award_delivery (edition_id, delivery_status, award_id),
    CHECK (final_rank > 0 AND reward_entry_no > 0 AND quantity > 0),
    CHECK ((delivery_status = 'PENDING' AND delivered_at IS NULL)
        OR (delivery_status = 'DELIVERED' AND delivered_at IS NOT NULL))
) ENGINE=InnoDB;

CREATE TABLE outbox_event (
    event_id BIGINT UNSIGNED NOT NULL,
    event_type VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    aggregate_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NULL,
    payload JSON NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    retry_count INT UNSIGNED NOT NULL DEFAULT 0,
    next_retry_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    lease_owner VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
    lease_until DATETIME(6) NULL,
    last_error VARCHAR(1024) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    published_at DATETIME(6) NULL,
    PRIMARY KEY (event_id),
    KEY idx_outbox_dispatch (status, next_retry_at, event_id),
    KEY idx_outbox_lease (status, lease_until),
    CHECK (status IN ('PENDING', 'PROCESSING', 'PUBLISHED'))
) ENGINE=InnoDB;

CREATE TABLE inbox_event (
    consumer_name VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    event_id BIGINT UNSIGNED NOT NULL,
    processed_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (consumer_name, event_id)
) ENGINE=InnoDB;
