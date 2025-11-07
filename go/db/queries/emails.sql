-- name: CreateEmail :execresult
INSERT INTO emails (
    user_id,
    gmail_message_id,
    sender_email,
    subject,
    body_preview,
    received_at,
    is_notified
) VALUES (
    ?, ?, ?, ?, ?, ?, ?
);

-- name: GetEmailByGmailMessageID :one
SELECT * FROM emails
WHERE gmail_message_id = ?
LIMIT 1;

-- name: GetEmailsByUserID :many
SELECT * FROM emails
WHERE user_id = ?
ORDER BY received_at DESC;

-- name: GetUnnotifiedEmailsByUserID :many
SELECT * FROM emails
WHERE user_id = ? AND is_notified = false
ORDER BY received_at DESC;

-- name: UpdateEmailNotified :exec
UPDATE emails
SET is_notified = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: MarkEmailAsNotified :exec
UPDATE emails
SET is_notified = true,
    updated_at = CURRENT_TIMESTAMP
WHERE gmail_message_id = ?;

-- name: DeleteEmailsByUserID :exec
DELETE FROM emails
WHERE user_id = ?;

-- name: GetRecentEmails :many
SELECT * FROM emails
WHERE user_id = ? AND received_at >= ?
ORDER BY received_at DESC;
