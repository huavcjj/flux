-- name: CreateUser :execresult
INSERT INTO users (
    id,
    line_user_id,
    gmail_access_token,
    gmail_refresh_token,
    gmail_token_expires_at,
    is_active
) VALUES (
    ?, ?, ?, ?, ?, ?
);
