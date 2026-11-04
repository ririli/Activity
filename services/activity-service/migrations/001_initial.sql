-- MySQL 8.4. All core tables share this schema and the same transaction connection.
USE activity_core;
SET NAMES utf8mb4;
SET time_zone = '+00:00';

CREATE TABLE activity_edition (
    edition_id BIGINT UNSIGNED NOT NULL,
    activity_code VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    name VARCHAR(128) NOT NULL,
    starts_at DATETIME(6) NOT NULL,
    ends_at DATETIME(6) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'NOT_STARTED',
    active_rule_version INT UNSIGNED NOT NULL,
    settlement_status VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (edition_id),
    KEY idx_edition_activity (activity_code, starts_at),
    CHECK (ends_at > starts_at),
    CHECK (active_rule_version > 0),
    CHECK (status IN ('NOT_STARTED', 'ACTIVE', 'ENDED')),
    CHECK (settlement_status IN ('PENDING', 'RUNNING', 'COMPLETED', 'FAILED'))
) ENGINE=InnoDB;

CREATE TABLE activity_rule_version (
    edition_id BIGINT UNSIGNED NOT NULL,
    version INT UNSIGNED NOT NULL,
    schema_version INT UNSIGNED NOT NULL,
    rule_json JSON NOT NULL,
    checksum BINARY(32) NOT NULL COMMENT 'SHA-256 of canonical rule representation',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (edition_id, version),
    CHECK (version > 0 AND schema_version > 0),
    CHECK (JSON_TYPE(rule_json) = 'OBJECT')
) ENGINE=InnoDB COMMENT='Immutable after use; application validates rule contents';

CREATE TABLE item_definition (
    item_id BIGINT UNSIGNED NOT NULL,
    item_type VARCHAR(16) NOT NULL,
    name VARCHAR(128) NOT NULL,
    icon_url VARCHAR(1024) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (item_id),
    CHECK (item_type IN ('GOLD', 'PETAL', 'RING', 'TITLE', 'FRAME'))
) ENGINE=InnoDB COMMENT='Display catalog; choice entitlement uses a reward code, not a fabricated item ID';

CREATE TABLE wallet (
    user_id BIGINT UNSIGNED NOT NULL,
    gold_balance BIGINT NOT NULL DEFAULT 0,
    petal_balance BIGINT NOT NULL DEFAULT 0,
    ledger_seq BIGINT UNSIGNED NOT NULL DEFAULT 0,
    version BIGINT UNSIGNED NOT NULL DEFAULT 0,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (user_id),
    CHECK (user_id > 0),
    CHECK (gold_balance >= 0 AND petal_balance >= 0)
) ENGINE=InnoDB COMMENT='Account-level balances; initialization is application business, not SQL defaults';

CREATE TABLE purchase_order (
    order_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    edition_id BIGINT UNSIGNED NOT NULL,
    request_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    purchase_type VARCHAR(24) NOT NULL,
    product_code VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    gold_cost BIGINT NOT NULL,
    petals_granted BIGINT NOT NULL,
    rule_version INT UNSIGNED NOT NULL,
    parent_confirm_id BIGINT UNSIGNED NULL COMMENT 'Preview ID when part of confirmation transaction',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (order_id),
    UNIQUE KEY uk_purchase_request (user_id, request_id),
    KEY idx_purchase_history (user_id, edition_id, created_at, order_id),
    CHECK (purchase_type IN ('SHOP', 'SIMULATED_GIFT', 'TOP_UP', 'PREVIEW_TOP_UP')),
    CHECK (gold_cost >= 0 AND petals_granted > 0 AND rule_version > 0),
    CHECK (purchase_type <> 'SIMULATED_GIFT' OR gold_cost = 0),
    CHECK ((purchase_type = 'PREVIEW_TOP_UP' AND parent_confirm_id IS NOT NULL)
        OR (purchase_type <> 'PREVIEW_TOP_UP' AND parent_confirm_id IS NULL))
) ENGINE=InnoDB COMMENT='Committed successes only; no real payment';

CREATE TABLE purchase_quota (
    user_id BIGINT UNSIGNED NOT NULL,
    edition_id BIGINT UNSIGNED NOT NULL,
    product_code VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    order_id BIGINT UNSIGNED NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (user_id, edition_id, product_code)
) ENGINE=InnoDB COMMENT='Only limited offers; repeatable 5-petal purchase has no quota row';

CREATE TABLE user_participation (
    user_id BIGINT UNSIGNED NOT NULL,
    edition_id BIGINT UNSIGNED NOT NULL,
    current_round_no INT UNSIGNED NOT NULL DEFAULT 1,
    formal_draw_count BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'Number of committed draw batches',
    draw_seq BIGINT UNSIGNED NOT NULL DEFAULT 0,
    draw_progress_version BIGINT UNSIGNED NOT NULL DEFAULT 0,
    version BIGINT UNSIGNED NOT NULL DEFAULT 0,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (user_id, edition_id),
    KEY idx_participation_edition (edition_id, user_id),
    CHECK (current_round_no > 0)
) ENGINE=InnoDB;

CREATE TABLE user_round (
    user_id BIGINT UNSIGNED NOT NULL,
    edition_id BIGINT UNSIGNED NOT NULL,
    round_no INT UNSIGNED NOT NULL,
    rule_version INT UNSIGNED NOT NULL,
    flower_count TINYINT UNSIGNED NOT NULL DEFAULT 0,
    consumed_gold_equivalent BIGINT NOT NULL DEFAULT 0,
    unlocked_chests TINYINT UNSIGNED NOT NULL DEFAULT 0,
    completed_chests TINYINT UNSIGNED NOT NULL DEFAULT 0,
    claimed_stage_mask TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'Six stage bits, ascending stage order',
    status VARCHAR(16) NOT NULL DEFAULT 'ACTIVE',
    started_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    completed_at DATETIME(6) NULL COMMENT 'Set when this round is closed by starting the next',
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (user_id, edition_id, round_no),
    KEY idx_round_settlement (edition_id, user_id, round_no),
    CHECK (round_no > 0 AND rule_version > 0),
    CHECK (flower_count <= 18 AND consumed_gold_equivalent >= 0),
    CHECK (unlocked_chests = flower_count DIV 6),
    CHECK (completed_chests <= unlocked_chests AND claimed_stage_mask <= 63),
    CHECK (status IN ('ACTIVE', 'FULL', 'COMPLETED')),
    CHECK ((status = 'ACTIVE' AND flower_count < 18 AND completed_at IS NULL)
        OR (status = 'FULL' AND flower_count = 18 AND completed_at IS NULL)
        OR (status = 'COMPLETED' AND flower_count = 18 AND completed_at IS NOT NULL))
) ENGINE=InnoDB;

CREATE TABLE stage_claim (
    claim_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    edition_id BIGINT UNSIGNED NOT NULL,
    round_no INT UNSIGNED NOT NULL,
    stage_code TINYINT UNSIGNED NOT NULL COMMENT 'Required flower count',
    reward_grant_id BIGINT UNSIGNED NOT NULL,
    claim_source VARCHAR(16) NOT NULL,
    claimed_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (claim_id),
    UNIQUE KEY uk_stage_claim (user_id, edition_id, round_no, stage_code),
    CHECK (round_no > 0),
    CHECK (stage_code IN (3, 5, 8, 11, 15, 17)),
    CHECK (claim_source IN ('USER', 'SETTLEMENT'))
) ENGINE=InnoDB;

CREATE TABLE chest_open (
    chest_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    edition_id BIGINT UNSIGNED NOT NULL,
    round_no INT UNSIGNED NOT NULL,
    chest_no TINYINT UNSIGNED NOT NULL,
    candidate_code VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    reward_type VARCHAR(16) NOT NULL,
    candidate_snapshot JSON NOT NULL,
    reward_grant_id BIGINT UNSIGNED NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'OPENED',
    opened_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    confirmed_at DATETIME(6) NULL,
    PRIMARY KEY (chest_id),
    UNIQUE KEY uk_chest_number (user_id, edition_id, round_no, chest_no),
    UNIQUE KEY uk_chest_candidate (user_id, edition_id, round_no, candidate_code),
    KEY idx_chest_settlement (edition_id, status, user_id, chest_id),
    CHECK (round_no > 0 AND chest_no BETWEEN 1 AND 3),
    CHECK (reward_type IN ('ITEM', 'CHOICE')),
    CHECK ((status = 'OPENED' AND confirmed_at IS NULL)
        OR (status = 'CONFIRMED' AND confirmed_at IS NOT NULL))
) ENGINE=InnoDB COMMENT='Reward obtained/selected state is independent from chest confirmation';

CREATE TABLE preview_order (
    preview_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    edition_id BIGINT UNSIGNED NOT NULL,
    round_no INT UNSIGNED NOT NULL,
    rule_version INT UNSIGNED NOT NULL,
    start_progress_snapshot JSON NOT NULL,
    start_progress_version BIGINT UNSIGNED NOT NULL,
    requested_count SMALLINT UNSIGNED NOT NULL DEFAULT 180,
    actual_count SMALLINT UNSIGNED NOT NULL,
    required_petals BIGINT NOT NULL,
    result_json JSON NOT NULL COMMENT 'Immutable ordered per-draw result array',
    expected_progress_json JSON NOT NULL,
    result_hash BINARY(32) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    confirmed_draw_id BIGINT UNSIGNED NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    confirmed_at DATETIME(6) NULL,
    abandoned_at DATETIME(6) NULL,
    PRIMARY KEY (preview_id),
    UNIQUE KEY uk_preview_user_edition (user_id, edition_id),
    CHECK (round_no > 0 AND rule_version > 0),
    CHECK (requested_count = 180 AND actual_count BETWEEN 1 AND 180 AND required_petals > 0),
    CHECK (JSON_TYPE(result_json) = 'ARRAY' AND JSON_LENGTH(result_json) = actual_count),
    CHECK ((status = 'PENDING' AND confirmed_draw_id IS NULL AND confirmed_at IS NULL AND abandoned_at IS NULL)
        OR (status = 'CONFIRMED' AND confirmed_draw_id IS NOT NULL AND confirmed_at IS NOT NULL AND abandoned_at IS NULL)
        OR (status = 'ABANDONED' AND confirmed_draw_id IS NULL AND confirmed_at IS NULL AND abandoned_at IS NOT NULL))
) ENGINE=InnoDB COMMENT='No business expiration; activity guard is enforced by application';

CREATE TABLE reward_grant (
    grant_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    edition_id BIGINT UNSIGNED NOT NULL,
    round_no INT UNSIGNED NULL,
    source_type VARCHAR(24) NOT NULL,
    source_id BIGINT UNSIGNED NOT NULL,
    source_entry_no SMALLINT UNSIGNED NOT NULL,
    draw_id BIGINT UNSIGNED NULL,
    draw_no SMALLINT UNSIGNED NULL,
    original_reward_code VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    original_item_name VARCHAR(128) NOT NULL,
    quantity BIGINT NOT NULL,
    candidate_snapshot JSON NULL,
    status VARCHAR(24) NOT NULL,
    final_item_id BIGINT UNSIGNED NULL,
    final_item_name VARCHAR(128) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    selected_at DATETIME(6) NULL,
    PRIMARY KEY (grant_id),
    UNIQUE KEY uk_reward_source (user_id, source_type, source_id, source_entry_no),
    UNIQUE KEY uk_reward_draw (user_id, draw_id, draw_no),
    KEY idx_reward_summary (user_id, edition_id, status, final_item_id),
    KEY idx_reward_pending (user_id, edition_id, status, grant_id),
    KEY idx_reward_chest (user_id, edition_id, source_type, created_at DESC, grant_id DESC),
    KEY idx_reward_owned (user_id, final_item_id, status),
    KEY idx_reward_settlement (edition_id, status, user_id, grant_id),
    CHECK (quantity > 0 AND source_entry_no > 0),
    CHECK (source_type IN ('DAY_DRAW', 'NIGHT_DRAW', 'PREVIEW', 'CHEST', 'STAGE', 'RANKING')),
    CHECK ((source_type IN ('DAY_DRAW', 'NIGHT_DRAW', 'PREVIEW') AND draw_id IS NOT NULL AND draw_no IS NOT NULL AND draw_no BETWEEN 1 AND 180)
        OR (source_type IN ('CHEST', 'STAGE', 'RANKING') AND draw_id IS NULL AND draw_no IS NULL)),
    CHECK ((status = 'OBTAINED' AND final_item_id IS NOT NULL AND final_item_name IS NOT NULL AND selected_at IS NULL)
        OR (status = 'PENDING_CHOICE' AND quantity = 1 AND candidate_snapshot IS NOT NULL AND final_item_id IS NULL AND final_item_name IS NULL AND selected_at IS NULL)
        OR (status = 'SELECTED' AND quantity = 1 AND candidate_snapshot IS NOT NULL AND final_item_id IS NOT NULL AND final_item_name IS NOT NULL AND selected_at IS NOT NULL))
) ENGINE=InnoDB COMMENT='One row per actual draw or external reward item; selection updates this row';

CREATE TABLE idempotency_request (
    user_id BIGINT UNSIGNED NOT NULL,
    operation_type VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    operation_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    request_hash BINARY(32) NOT NULL,
    accepted_at DATETIME(6) NOT NULL,
    recover_until DATETIME(6) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'SUCCEEDED',
    result_user_id BIGINT UNSIGNED NOT NULL,
    result_draw_id BIGINT UNSIGNED NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (user_id, operation_type, operation_id),
    CHECK (result_user_id = user_id),
    CHECK (status = 'SUCCEEDED'),
    CHECK (recover_until = DATE_ADD(accepted_at, INTERVAL 24 HOUR))
) ENGINE=InnoDB COMMENT='Successful ordinary draw requests; never delete solely because recovery window elapsed';

CREATE TABLE settlement_task (
    edition_id BIGINT UNSIGNED NOT NULL,
    step VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    partition_no TINYINT NOT NULL DEFAULT -1 COMMENT '-1 for unsharded step, 0..3 for record shard',
    cursor_json JSON NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    retry_count INT UNSIGNED NOT NULL DEFAULT 0,
    lease_owner VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
    lease_until DATETIME(6) NULL,
    last_error VARCHAR(1024) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (edition_id, step, partition_no),
    KEY idx_task_dispatch (status, lease_until),
    CHECK (partition_no BETWEEN -1 AND 3),
    CHECK (status IN ('PENDING', 'RUNNING', 'COMPLETED', 'FAILED'))
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
