USE activity_core;
ALTER TABLE user_participation
 ADD COLUMN ranking_score BIGINT NOT NULL DEFAULT 0,
 ADD COLUMN ranking_score_reached_at_sec BIGINT UNSIGNED NOT NULL DEFAULT 0,
 ADD CHECK (ranking_score >= 0);
