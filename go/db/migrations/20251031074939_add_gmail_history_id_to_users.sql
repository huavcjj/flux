-- migrate:up

ALTER TABLE users ADD COLUMN gmail_history_id BIGINT UNSIGNED DEFAULT NULL AFTER gmail_token_expires_at;

-- migrate:down

ALTER TABLE users DROP COLUMN gmail_history_id;