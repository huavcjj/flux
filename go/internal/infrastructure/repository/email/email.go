package email

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	email_domain "github.com/huavcjj/flux/internal/domain/email"
	"github.com/huavcjj/flux/internal/infrastructure/db"
)

type emailRepo struct {
	queries *db.Queries
}

var _ email_domain.EmailRepo = (*emailRepo)(nil)

func NewEmailRepo(dbConn *sql.DB) email_domain.EmailRepo {
	return &emailRepo{
		queries: db.New(dbConn),
	}
}

func (r *emailRepo) CreateEmail(ctx context.Context, email *email_domain.Email) error {
	var subject, bodyPreview sql.NullString

	if email.Subject != nil {
		subject = sql.NullString{String: *email.Subject, Valid: true}
	}
	if email.BodyPreview != nil {
		bodyPreview = sql.NullString{String: *email.BodyPreview, Valid: true}
	}

	_, err := r.queries.CreateEmail(ctx, db.CreateEmailParams{
		UserID:         email.UserID,
		GmailMessageID: email.GmailMessageID,
		SenderEmail:    email.SenderEmail,
		Subject:        subject,
		BodyPreview:    bodyPreview,
		ReceivedAt:     email.ReceivedAt,
		IsNotified:     sql.NullBool{Bool: email.IsNotified, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to create email: %w", err)
	}

	return nil
}

func (r *emailRepo) GetEmailByGmailMessageID(ctx context.Context, gmailMessageID string) (*email_domain.Email, error) {
	dbEmail, err := r.queries.GetEmailByGmailMessageID(ctx, gmailMessageID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get email by gmail message id: %w", err)
	}

	return r.dbEmailToDomain(dbEmail), nil
}

func (r *emailRepo) GetEmailsByUserID(ctx context.Context, userID string) ([]email_domain.Email, error) {
	dbEmails, err := r.queries.GetEmailsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get emails by user id: %w", err)
	}

	emails := make([]email_domain.Email, 0, len(dbEmails))
	for _, dbEmail := range dbEmails {
		emails = append(emails, *r.dbEmailToDomain(dbEmail))
	}

	return emails, nil
}

func (r *emailRepo) GetUnnotifiedEmailsByUserID(ctx context.Context, userID string) ([]email_domain.Email, error) {
	dbEmails, err := r.queries.GetUnnotifiedEmailsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get unnotified emails by user id: %w", err)
	}

	emails := make([]email_domain.Email, 0, len(dbEmails))
	for _, dbEmail := range dbEmails {
		emails = append(emails, *r.dbEmailToDomain(dbEmail))
	}

	return emails, nil
}

func (r *emailRepo) GetRecentEmails(ctx context.Context, userID string, since time.Time) ([]email_domain.Email, error) {
	dbEmails, err := r.queries.GetRecentEmails(ctx, db.GetRecentEmailsParams{
		UserID:     userID,
		ReceivedAt: since,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get recent emails: %w", err)
	}

	emails := make([]email_domain.Email, 0, len(dbEmails))
	for _, dbEmail := range dbEmails {
		emails = append(emails, *r.dbEmailToDomain(dbEmail))
	}

	return emails, nil
}

func (r *emailRepo) MarkEmailAsNotified(ctx context.Context, gmailMessageID string) error {
	err := r.queries.MarkEmailAsNotified(ctx, gmailMessageID)
	if err != nil {
		return fmt.Errorf("failed to mark email as notified: %w", err)
	}

	return nil
}

func (r *emailRepo) DeleteEmailsByUserID(ctx context.Context, userID string) error {
	err := r.queries.DeleteEmailsByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to delete emails by user id: %w", err)
	}

	return nil
}

func (r *emailRepo) dbEmailToDomain(dbEmail db.Email) *email_domain.Email {
	email := &email_domain.Email{
		ID:             dbEmail.ID,
		UserID:         dbEmail.UserID,
		GmailMessageID: dbEmail.GmailMessageID,
		SenderEmail:    dbEmail.SenderEmail,
		ReceivedAt:     dbEmail.ReceivedAt,
		IsNotified:     dbEmail.IsNotified.Bool,
	}

	if dbEmail.Subject.Valid {
		email.Subject = &dbEmail.Subject.String
	}
	if dbEmail.BodyPreview.Valid {
		email.BodyPreview = &dbEmail.BodyPreview.String
	}
	if dbEmail.CreatedAt.Valid {
		email.CreatedAt = dbEmail.CreatedAt.Time
	}
	if dbEmail.UpdatedAt.Valid {
		email.UpdatedAt = dbEmail.UpdatedAt.Time
	}

	return email
}
