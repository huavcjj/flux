package email

import (
	"context"
	"time"
)

type Email struct {
	ID             uint64
	UserID         string
	GmailMessageID string
	SenderEmail    string
	Subject        *string
	BodyPreview    *string
	ReceivedAt     time.Time
	IsNotified     bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type EmailRepo interface {
	CreateEmail(ctx context.Context, email *Email) error
	GetEmailByGmailMessageID(ctx context.Context, gmailMessageID string) (*Email, error)
	GetEmailsByUserID(ctx context.Context, userID string) ([]Email, error)
	GetUnnotifiedEmailsByUserID(ctx context.Context, userID string) ([]Email, error)
	GetRecentEmails(ctx context.Context, userID string, since time.Time) ([]Email, error)
	MarkEmailAsNotified(ctx context.Context, gmailMessageID string) error
	DeleteEmailsByUserID(ctx context.Context, userID string) error
}
