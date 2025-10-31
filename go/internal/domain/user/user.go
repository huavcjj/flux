package user

import (
	"context"
	"time"

	"golang.org/x/oauth2"
)

type User struct {
	ID                  string
	LineUserID          string
	GmailAccessToken    *string
	GmailRefreshToken   *string
	GmailTokenExpiresAt *int64
	IsActive            bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type UserRepo interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByLineUserID(ctx context.Context, lineUserID string) (*User, error)
	UpdateGmailTokens(ctx context.Context, lineUserID string, token *oauth2.Token) error
	GetUserByID(ctx context.Context, userID string) (*User, error)
	GetAllActiveUsers(ctx context.Context) ([]User, error)
}
