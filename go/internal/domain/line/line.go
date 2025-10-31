package line

import "context"

type NotificationMessage struct {
	UserID  string
	Message string
}

type LineRepo interface {
	SendTextMessage(ctx context.Context, userID, message string) error
	PushMessage(ctx context.Context, userID, message string) error
	SendButtonMessage(ctx context.Context, userID, text, buttonText, buttonURL string) error
}
