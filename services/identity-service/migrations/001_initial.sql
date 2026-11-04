-- MySQL 8.4. One-time initial migration; intentionally no IF NOT EXISTS on tables.
USE activity_identity;
SET NAMES utf8mb4;
SET time_zone = '+00:00';

CREATE TABLE account (
    user_id BIGINT UNSIGNED NOT NULL COMMENT 'Application-generated global ID',
    account_name VARCHAR(20) CHARACTER SET ascii COLLATE ascii_bin NOT NULL COMMENT 'Case-sensitive; application validates letters, digits and underscore',
    password_hash VARCHAR(255) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    nickname VARCHAR(64) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'ACTIVE',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (user_id),
    UNIQUE KEY uk_account_name (account_name),
    CHECK (user_id > 0),
    CHECK (CHAR_LENGTH(account_name) BETWEEN 3 AND 20),
    CHECK (status IN ('ACTIVE', 'DISABLED'))
) ENGINE=InnoDB COMMENT='Accounts; no plaintext passwords';

CREATE TABLE login_session (
    session_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    refresh_token_hash BINARY(32) NOT NULL COMMENT 'SHA-256 of cryptographically random refresh token',
    expires_at DATETIME(6) NOT NULL,
    revoked_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (session_id),
    UNIQUE KEY uk_session_token (refresh_token_hash),
    KEY idx_session_user (user_id, revoked_at),
    KEY idx_session_expiry (expires_at),
    CHECK (expires_at > created_at)
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
