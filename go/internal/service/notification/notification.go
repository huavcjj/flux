package notification

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	emailRepo "github.com/huavcjj/flux/internal/domain/email"
	gmailRepo "github.com/huavcjj/flux/internal/domain/gmail"
	lineRepo "github.com/huavcjj/flux/internal/domain/line"
	userRepo "github.com/huavcjj/flux/internal/domain/user"
	"golang.org/x/oauth2"
)

const (
	maxUnreadEmails = 10
	maxPushEmails   = 5
)

type Service struct {
	gmailRepo   gmailRepo.GmailRepo
	lineRepo    lineRepo.LineRepo
	userRepo    userRepo.UserRepo
	emailRepo   emailRepo.EmailRepo
	pendingAuth map[string]bool
}

func NewService(gmailRepo gmailRepo.GmailRepo, lineRepo lineRepo.LineRepo, userRepo userRepo.UserRepo, emailRepo emailRepo.EmailRepo) *Service {
	return &Service{
		gmailRepo:   gmailRepo,
		lineRepo:    lineRepo,
		userRepo:    userRepo,
		emailRepo:   emailRepo,
		pendingAuth: make(map[string]bool),
	}
}

func (s *Service) IsAuthPending(userID string) bool {
	return s.pendingAuth[userID]
}

func (s *Service) getUserToken(user *userRepo.User) *oauth2.Token {
	var expiry time.Time
	if user.GmailTokenExpiresAt != nil {
		expiry = time.Unix(*user.GmailTokenExpiresAt, 0)
	}

	token := &oauth2.Token{Expiry: expiry}
	if user.GmailAccessToken != nil {
		token.AccessToken = *user.GmailAccessToken
	}
	if user.GmailRefreshToken != nil {
		token.RefreshToken = *user.GmailRefreshToken
	}

	return token
}

func (s *Service) getAuthenticatedUser(ctx context.Context, userID string) (*userRepo.User, error) {
	user, err := s.userRepo.GetUserByLineUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil || user.GmailAccessToken == nil || *user.GmailAccessToken == "" {
		return nil, fmt.Errorf("user not authenticated with Gmail")
	}

	return user, nil
}

func (s *Service) SendUnreadEmailList(ctx context.Context, userID string) error {
	if s.gmailRepo == nil {
		return s.lineRepo.PushMessage(ctx, userID, "Gmail機能は現在利用できません。設定を確認してください。")
	}

	user, err := s.getAuthenticatedUser(ctx, userID)
	if err != nil {
		return s.lineRepo.PushMessage(ctx, userID, "Gmail連携が必要です。「Gmail連携」を送信して認証してください。")
	}

	messages, err := s.gmailRepo.GetUnreadMessages(ctx, s.getUserToken(user), maxUnreadEmails)
	if err != nil {
		return fmt.Errorf("failed to get unread messages: %w", err)
	}

	if len(messages) == 0 {
		return s.lineRepo.PushMessage(ctx, userID, "📭 未読メールはありません")
	}

	text := fmt.Sprintf("📬 未読メール (%d件)\n\n", len(messages))
	for i, msg := range messages {
		text += fmt.Sprintf("%d. %s\n件名: %s\n%s\n\n", i+1, msg.From, msg.Subject, msg.Snippet)
	}

	slog.Info("unread email list sent", "user_id", userID, "count", len(messages))
	return s.lineRepo.PushMessage(ctx, userID, text)
}

func (s *Service) SendEmailList(ctx context.Context, userID string, maxResults int64) error {
	if s.gmailRepo == nil {
		return s.lineRepo.PushMessage(ctx, userID, "Gmail機能は現在利用できません。設定を確認してください。")
	}

	user, err := s.getAuthenticatedUser(ctx, userID)
	if err != nil {
		return s.lineRepo.PushMessage(ctx, userID, "Gmail連携が必要です。「Gmail連携」を送信して認証してください。")
	}

	messages, err := s.gmailRepo.GetLatestMessages(ctx, s.getUserToken(user), maxResults)
	if err != nil {
		return fmt.Errorf("failed to get latest messages: %w", err)
	}

	if len(messages) == 0 {
		return s.lineRepo.PushMessage(ctx, userID, "📭 メールはありません")
	}

	text := fmt.Sprintf("📨 最新メール (%d件)\n\n", len(messages))
	for i, msg := range messages {
		text += fmt.Sprintf("%d. %s\n件名: %s\n%s\n\n", i+1, msg.From, msg.Subject, msg.Snippet)
	}

	slog.Info("email list sent", "user_id", userID, "count", len(messages))
	return s.lineRepo.PushMessage(ctx, userID, text)
}

func (s *Service) StartGmailAuth(ctx context.Context, userID string) error {
	if s.gmailRepo == nil {
		return s.lineRepo.PushMessage(ctx, userID, "Gmail機能は現在利用できません。管理者にお問い合わせください。")
	}

	s.pendingAuth[userID] = true
	authURL := s.gmailRepo.GetAuthURL(userID)

	instructionMsg := `Gmail連携を開始します。

【重要】以下の手順で認証してください：

1. 次のメッセージのURLを長押し
2. 「Safariで開く」または「Chromeで開く」を選択
3. Googleアカウントで認証

※ LINEアプリ内で開くとエラーになります`

	if err := s.lineRepo.PushMessage(ctx, userID, instructionMsg); err != nil {
		return fmt.Errorf("failed to send instruction: %w", err)
	}

	if err := s.lineRepo.PushMessage(ctx, userID, authURL); err != nil {
		return fmt.Errorf("failed to send auth URL: %w", err)
	}

	if err := s.lineRepo.PushMessage(ctx, userID, "認証が完了すると自動的に連携されます。"); err != nil {
		return fmt.Errorf("failed to send completion message: %w", err)
	}

	slog.Info("Gmail auth started", "user_id", userID)
	return nil
}

func (s *Service) CompleteGmailAuth(ctx context.Context, userID, authCode string) error {
	if s.gmailRepo == nil {
		return fmt.Errorf("gmail repository not initialized")
	}

	token, err := s.gmailRepo.ExchangeCode(ctx, authCode)
	if err != nil {
		delete(s.pendingAuth, userID)
		return fmt.Errorf("failed to exchange code: %w", err)
	}

	user, err := s.userRepo.GetUserByLineUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		user = &userRepo.User{LineUserID: userID}
		if err := s.userRepo.CreateUser(ctx, user); err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
	}

	if err := s.userRepo.UpdateGmailTokens(ctx, userID, token); err != nil {
		delete(s.pendingAuth, userID)
		return fmt.Errorf("failed to save tokens: %w", err)
	}

	if err := s.gmailRepo.WatchMailbox(ctx, token, os.Getenv("PUBSUB_TOPIC")); err != nil {
		slog.Warn("failed to setup Gmail watch", "user_id", userID, "error", err)
	} else {
		slog.Info("Gmail watch setup successfully", "user_id", userID)
	}

	delete(s.pendingAuth, userID)

	successMsg := "✅ Gmail連携が完了しました！\n\n新着メールが届くと自動で通知されます。\n\n手動確認: 「未読mail」または「mail一覧」を送信"
	if err := s.lineRepo.PushMessage(ctx, userID, successMsg); err != nil {
		return fmt.Errorf("failed to send success message: %w", err)
	}

	slog.Info("Gmail auth completed", "user_id", userID)
	return nil
}

func (s *Service) ProcessGmailPushNotification(ctx context.Context) error {
	users, err := s.userRepo.GetAllActiveUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active users: %w", err)
	}

	for _, user := range users {
		if user.GmailAccessToken == nil || *user.GmailAccessToken == "" {
			continue
		}

		messages, err := s.gmailRepo.GetUnreadMessages(ctx, s.getUserToken(&user), maxPushEmails)
		if err != nil {
			slog.Error("failed to get unread messages", "user_id", user.LineUserID, "error", err)
			continue
		}

		for _, msg := range messages {
			text := fmt.Sprintf("📧 新着メール\n\n差出人: %s\n件名: %s\n\n%s", msg.From, msg.Subject, msg.Snippet)
			if err := s.lineRepo.PushMessage(ctx, user.LineUserID, text); err != nil {
				slog.Error("failed to send LINE notification", "user_id", user.LineUserID, "message_id", msg.ID, "error", err)
				continue
			}
			slog.Info("push notification sent", "user_id", user.LineUserID, "message_id", msg.ID, "subject", msg.Subject)
		}
	}

	return nil
}
