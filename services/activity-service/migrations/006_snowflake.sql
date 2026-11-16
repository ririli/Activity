USE activity_core;

CREATE TABLE snowflake_reservation (
    node_id SMALLINT UNSIGNED NOT NULL,
    reserved_until_ms BIGINT NOT NULL,
    PRIMARY KEY (node_id),
    CHECK (node_id <= 1023 AND reserved_until_ms BETWEEN 0 AND 2199023255551)
) ENGINE=InnoDB;
