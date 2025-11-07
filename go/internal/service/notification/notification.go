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

	msgGmailUnavailable     = "Gmail機能は現在利用できません。設定を確認してください。"
	msgGmailUnavailableAuth = "Gmail機能は現在利用できません。管理者にお問い合わせください。"
	msgAuthRequired         = "Gmail連携が必要です。「Gmail連携」を送信して認証してください。"
	msgNoUnreadEmails       = "📭 未読メールはありません"
	msgNoEmails             = "📭 メールはありません"
	msgAuthComplete         = "✅ Gmail連携が完了しました！\n\n新着メールが届くと自動で通知されます。\n\n手動確認: 「未読mail」または「mail一覧」を送信"
	msgAuthStart            = "Gmail連携を開始します。\n\n次のメッセージのURLからGoogleアカウントで認証してください。\n\n認証が完了すると自動的に連携されます。"

	titleUnreadEmails = "📬 未読メール"
	titleLatestEmails = "📨 最新メール"
	titleNewEmail     = "📧 新着メール"
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
		return s.lineRepo.PushMessage(ctx, userID, msgGmailUnavailable)
	}

	user, err := s.getAuthenticatedUser(ctx, userID)
	if err != nil {
		return s.lineRepo.PushMessage(ctx, userID, msgAuthRequired)
	}

	messages, err := s.gmailRepo.GetUnreadMessages(ctx, s.getUserToken(user), maxUnreadEmails)
	if err != nil {
		return fmt.Errorf("failed to get unread messages: %w", err)
	}

	if len(messages) == 0 {
		return s.lineRepo.PushMessage(ctx, userID, msgNoUnreadEmails)
	}

	slog.Info("unread email list sent", "user_id", userID, "count", len(messages))
	return s.lineRepo.PushMessage(ctx, userID, s.formatEmailList(titleUnreadEmails, messages))
}

func (s *Service) SendEmailList(ctx context.Context, userID string, maxResults int64) error {
	if s.gmailRepo == nil {
		return s.lineRepo.PushMessage(ctx, userID, msgGmailUnavailable)
	}

	user, err := s.getAuthenticatedUser(ctx, userID)
	if err != nil {
		return s.lineRepo.PushMessage(ctx, userID, msgAuthRequired)
	}

	messages, err := s.gmailRepo.GetLatestMessages(ctx, s.getUserToken(user), maxResults)
	if err != nil {
		return fmt.Errorf("failed to get latest messages: %w", err)
	}

	if len(messages) == 0 {
		return s.lineRepo.PushMessage(ctx, userID, msgNoEmails)
	}

	slog.Info("email list sent", "user_id", userID, "count", len(messages))
	return s.lineRepo.PushMessage(ctx, userID, s.formatEmailList(titleLatestEmails, messages))
}

func (s *Service) StartGmailAuth(ctx context.Context, userID string) error {
	if s.gmailRepo == nil {
		return s.lineRepo.PushMessage(ctx, userID, msgGmailUnavailableAuth)
	}

	s.pendingAuth[userID] = true
	authURL := s.gmailRepo.GetAuthURL(userID)

	if err := s.lineRepo.PushMessage(ctx, userID, msgAuthStart); err != nil {
		return fmt.Errorf("failed to send instruction: %w", err)
	}

	if err := s.lineRepo.PushMessage(ctx, userID, authURL); err != nil {
		return fmt.Errorf("failed to send auth URL: %w", err)
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

	if err := s.lineRepo.PushMessage(ctx, userID, msgAuthComplete); err != nil {
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
			existingEmail, err := s.emailRepo.GetEmailByGmailMessageID(ctx, msg.ID)
			if err != nil {
				slog.Error("failed to check email existence", "message_id", msg.ID, "error", err)
				continue
			}

			if existingEmail != nil {
				continue
			}

			email := &emailRepo.Email{
				UserID:         user.ID,
				GmailMessageID: msg.ID,
				SenderEmail:    msg.From,
				Subject:        &msg.Subject,
				BodyPreview:    &msg.Snippet,
				ReceivedAt:     msg.Date,
				IsNotified:     false,
			}

			if err := s.emailRepo.CreateEmail(ctx, email); err != nil {
				slog.Error("failed to create email record", "message_id", msg.ID, "error", err)
				continue
			}
		}

		unnotifiedEmails, err := s.emailRepo.GetUnnotifiedEmailsByUserID(ctx, user.ID)
		if err != nil {
			slog.Error("failed to get unnotified emails", "user_id", user.LineUserID, "error", err)
			continue
		}

		for _, email := range unnotifiedEmails {
			msg := &gmailRepo.Message{
				ID:      email.GmailMessageID,
				From:    email.SenderEmail,
				Subject: *email.Subject,
				Snippet: *email.BodyPreview,
				Date:    email.ReceivedAt,
			}

			if err := s.lineRepo.PushMessage(ctx, user.LineUserID, s.formatNewEmail(msg)); err != nil {
				slog.Error("failed to send LINE notification", "user_id", user.LineUserID, "message_id", msg.ID, "error", err)
				continue
			}

			if err := s.emailRepo.MarkEmailAsNotified(ctx, email.GmailMessageID); err != nil {
				slog.Error("failed to mark email as notified", "message_id", email.GmailMessageID, "error", err)
				continue
			}

			slog.Info("push notification sent", "user_id", user.LineUserID, "message_id", msg.ID, "subject", msg.Subject)
		}
	}

	return nil
}

func (s *Service) formatEmailList(title string, messages []*gmailRepo.Message) string {
	text := fmt.Sprintf("%s (%d件)\n\n", title, len(messages))
	for i, msg := range messages {
		text += fmt.Sprintf("%d. %s\n件名: %s\n%s\n\n", i+1, msg.From, msg.Subject, msg.Snippet)
	}
	return text
}

func (s *Service) formatNewEmail(msg *gmailRepo.Message) string {
	return fmt.Sprintf("%s\n\n差出人: %s\n件名: %s\n\n%s", titleNewEmail, msg.From, msg.Subject, msg.Snippet)
}
