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

-- name: GetUserByLineUserID :one
SELECT * FROM users
WHERE line_user_id = ? AND is_active = true
LIMIT 1;

-- name: UpdateUserGmailTokens :exec
UPDATE users
SET gmail_access_token = ?,
    gmail_refresh_token = ?,
    gmail_token_expires_at = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE line_user_id = ?;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = ? AND is_active = true
LIMIT 1;

-- name: GetAllActiveUsers :many
SELECT * FROM users
WHERE is_active = true;
