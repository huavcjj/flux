package user

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	user_domain "github.com/huavcjj/flux/internal/domain/user"
	"github.com/huavcjj/flux/internal/infrastructure/db"
	"golang.org/x/oauth2"
)

type userRepo struct {
	queries *db.Queries
}

var _ user_domain.UserRepo = (*userRepo)(nil)

func NewUserRepo(dbConn *sql.DB) user_domain.UserRepo {
	return &userRepo{
		queries: db.New(dbConn),
	}
}

func (r *userRepo) CreateUser(ctx context.Context, user *user_domain.User) error {
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	var gmailAccessToken, gmailRefreshToken sql.NullString
	var gmailTokenExpiresAt sql.NullInt64

	if user.GmailAccessToken != nil {
		gmailAccessToken = sql.NullString{String: *user.GmailAccessToken, Valid: true}
	}
	if user.GmailRefreshToken != nil {
		gmailRefreshToken = sql.NullString{String: *user.GmailRefreshToken, Valid: true}
	}
	if user.GmailTokenExpiresAt != nil {
		gmailTokenExpiresAt = sql.NullInt64{Int64: *user.GmailTokenExpiresAt, Valid: true}
	}

	_, err := r.queries.CreateUser(ctx, db.CreateUserParams{
		ID:                  user.ID,
		LineUserID:          user.LineUserID,
		GmailAccessToken:    gmailAccessToken,
		GmailRefreshToken:   gmailRefreshToken,
		GmailTokenExpiresAt: gmailTokenExpiresAt,
		IsActive:            sql.NullBool{Bool: true, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *userRepo) GetUserByLineUserID(ctx context.Context, lineUserID string) (*user_domain.User, error) {
	dbUser, err := r.queries.GetUserByLineUserID(ctx, lineUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by line user id: %w", err)
	}

	return r.dbUserToDomain(dbUser), nil
}

func (r *userRepo) GetUserByID(ctx context.Context, userID string) (*user_domain.User, error) {
	dbUser, err := r.queries.GetUserByID(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return r.dbUserToDomain(dbUser), nil
}

func (r *userRepo) UpdateGmailTokens(ctx context.Context, lineUserID string, token *oauth2.Token) error {
	var accessToken, refreshToken sql.NullString
	var expiresAt sql.NullInt64

	if token.AccessToken != "" {
		accessToken = sql.NullString{String: token.AccessToken, Valid: true}
	}
	if token.RefreshToken != "" {
		refreshToken = sql.NullString{String: token.RefreshToken, Valid: true}
	}
	if !token.Expiry.IsZero() {
		expiresAt = sql.NullInt64{Int64: token.Expiry.Unix(), Valid: true}
	}

	err := r.queries.UpdateUserGmailTokens(ctx, db.UpdateUserGmailTokensParams{
		GmailAccessToken:    accessToken,
		GmailRefreshToken:   refreshToken,
		GmailTokenExpiresAt: expiresAt,
		LineUserID:          lineUserID,
	})
	if err != nil {
		return fmt.Errorf("failed to update gmail tokens: %w", err)
	}

	return nil
}

func (r *userRepo) dbUserToDomain(dbUser db.User) *user_domain.User {
	user := &user_domain.User{
		ID:         dbUser.ID,
		LineUserID: dbUser.LineUserID,
		IsActive:   dbUser.IsActive.Bool,
	}

	if dbUser.GmailAccessToken.Valid {
		token := dbUser.GmailAccessToken.String
		user.GmailAccessToken = &token
	}
	if dbUser.GmailRefreshToken.Valid {
		token := dbUser.GmailRefreshToken.String
		user.GmailRefreshToken = &token
	}
	if dbUser.GmailTokenExpiresAt.Valid {
		expiresAt := dbUser.GmailTokenExpiresAt.Int64
		user.GmailTokenExpiresAt = &expiresAt
	}
	if dbUser.CreatedAt.Valid {
		user.CreatedAt = dbUser.CreatedAt.Time
	}
	if dbUser.UpdatedAt.Valid {
		user.UpdatedAt = dbUser.UpdatedAt.Time
	}

	return user
}

func (r *userRepo) GetAllActiveUsers(ctx context.Context) ([]user_domain.User, error) {
	dbUsers, err := r.queries.GetAllActiveUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all active users: %w", err)
	}

	users := make([]user_domain.User, 0, len(dbUsers))
	for _, dbUser := range dbUsers {
		users = append(users, *r.dbUserToDomain(dbUser))
	}

	return users, nil
}
