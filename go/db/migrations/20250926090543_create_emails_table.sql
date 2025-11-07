-- migrate:up
CREATE TABLE emails (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    gmail_message_id VARCHAR(255) NOT NULL UNIQUE,
    sender_email VARCHAR(255) NOT NULL,
    subject VARCHAR(500),
    body_preview TEXT,
    received_at TIMESTAMP NOT NULL,
    is_notified BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- migrate:down

