-- migrate:up
CREATE TABLE users (
    id VARCHAR(36) PRIMARY KEY,
    line_user_id VARCHAR(255) NOT NULL UNIQUE,
    gmail_access_token TEXT,
    gmail_refresh_token TEXT,
    gmail_token_expires_at BIGINT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
-- migrate:down
DROP TABLE users;

