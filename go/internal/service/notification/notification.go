package notification

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	gmail_repo "github.com/huavcjj/flux/internal/domain/gmail"
	line_repo "github.com/huavcjj/flux/internal/domain/line"
	user_repo "github.com/huavcjj/flux/internal/domain/user"
	"golang.org/x/oauth2"
)

type Service struct {
	gmailRepo   gmail_repo.GmailRepo
	lineRepo    line_repo.LineRepo
	userRepo    user_repo.UserRepo
	pubsubTopic string
	// 認証待ちのユーザーを一時的に保存（本来はRedisなどを使う）
	pendingAuth map[string]bool
}

func NewService(gmailRepo gmail_repo.GmailRepo, lineRepo line_repo.LineRepo, userRepo user_repo.UserRepo, pubsubTopic string) *Service {
	return &Service{
		gmailRepo:   gmailRepo,
		lineRepo:    lineRepo,
		userRepo:    userRepo,
		pubsubTopic: pubsubTopic,
		pendingAuth: make(map[string]bool),
	}
}

// getUserToken converts user's stored tokens to oauth2.Token
func (s *Service) getUserToken(user *user_repo.User) *oauth2.Token {
	var expiry time.Time
	if user.GmailTokenExpiresAt != nil {
		expiry = time.Unix(*user.GmailTokenExpiresAt, 0)
	}

	token := &oauth2.Token{
		Expiry: expiry,
	}
	if user.GmailAccessToken != nil {
		token.AccessToken = *user.GmailAccessToken
	}
	if user.GmailRefreshToken != nil {
		token.RefreshToken = *user.GmailRefreshToken
	}

	return token
}

func (s *Service) NotifyNewEmail(ctx context.Context, userID string, messageID string) error {
	// Get user's Gmail token
	user, err := s.userRepo.GetUserByLineUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil || user.GmailAccessToken == nil || *user.GmailAccessToken == "" {
		return fmt.Errorf("user not authenticated with Gmail")
	}

	token := s.getUserToken(user)

	message, err := s.gmailRepo.GetMessage(ctx, token, messageID)
	if err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}

	notificationText := fmt.Sprintf(
		"📧 新着メール\n\n差出人: %s\n件名: %s\n\n%s",
		message.From,
		message.Subject,
		message.Snippet,
	)

	if err := s.lineRepo.PushMessage(ctx, userID, notificationText); err != nil {
		return fmt.Errorf("failed to send LINE notification: %w", err)
	}

	slog.Info("notification sent successfully",
		"message_id", messageID,
		"subject", message.Subject,
	)

	return nil
}

func (s *Service) CheckAndNotifyNewEmails(ctx context.Context, userID string, maxResults int64) error {
	// Get user's Gmail token
	user, err := s.userRepo.GetUserByLineUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil || user.GmailAccessToken == nil || *user.GmailAccessToken == "" {
		return fmt.Errorf("user not authenticated with Gmail")
	}

	token := s.getUserToken(user)

	messages, err := s.gmailRepo.GetLatestMessages(ctx, token, maxResults)
	if err != nil {
		return fmt.Errorf("failed to get latest messages: %w", err)
	}

	for _, message := range messages {
		if err := s.NotifyNewEmail(ctx, userID, message.ID); err != nil {
			slog.Error("failed to notify email",
				"message_id", message.ID,
				"error", err,
			)
			continue
		}
	}

	return nil
}

func (s *Service) SendUnreadEmailList(ctx context.Context, userID string) error {
	if s.gmailRepo == nil {
		errorMsg := "Gmail機能は現在利用できません。設定を確認してください。"
		if err := s.lineRepo.PushMessage(ctx, userID, errorMsg); err != nil {
			return fmt.Errorf("failed to send error message: %w", err)
		}
		return nil
	}

	// Get user's Gmail token
	user, err := s.userRepo.GetUserByLineUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil || user.GmailAccessToken == nil || *user.GmailAccessToken == "" {
		errorMsg := "Gmail連携が必要です。「Gmail連携」を送信して認証してください。"
		if err := s.lineRepo.PushMessage(ctx, userID, errorMsg); err != nil {
			return fmt.Errorf("failed to send error message: %w", err)
		}
		return nil
	}

	token := s.getUserToken(user)

	messages, err := s.gmailRepo.GetUnreadMessages(ctx, token, 10)
	if err != nil {
		return fmt.Errorf("failed to get unread messages: %w", err)
	}

	if len(messages) == 0 {
		notificationText := "📭 未読メールはありません"
		if err := s.lineRepo.PushMessage(ctx, userID, notificationText); err != nil {
			return fmt.Errorf("failed to send LINE notification: %w", err)
		}
		return nil
	}

	notificationText := fmt.Sprintf("📬 未読メール (%d件)\n\n", len(messages))
	for i, message := range messages {
		notificationText += fmt.Sprintf("%d. %s\n件名: %s\n%s\n\n",
			i+1,
			message.From,
			message.Subject,
			message.Snippet,
		)
	}

	if err := s.lineRepo.PushMessage(ctx, userID, notificationText); err != nil {
		return fmt.Errorf("failed to send LINE notification: %w", err)
	}

	slog.Info("unread email list sent successfully",
		"user_id", userID,
		"count", len(messages),
	)

	return nil
}

func (s *Service) SendEmailList(ctx context.Context, userID string, maxResults int64) error {
	if s.gmailRepo == nil {
		errorMsg := "Gmail機能は現在利用できません。設定を確認してください。"
		if err := s.lineRepo.PushMessage(ctx, userID, errorMsg); err != nil {
			return fmt.Errorf("failed to send error message: %w", err)
		}
		return nil
	}

	// Get user's Gmail token
	user, err := s.userRepo.GetUserByLineUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil || user.GmailAccessToken == nil || *user.GmailAccessToken == "" {
		errorMsg := "Gmail連携が必要です。「Gmail連携」を送信して認証してください。"
		if err := s.lineRepo.PushMessage(ctx, userID, errorMsg); err != nil {
			return fmt.Errorf("failed to send error message: %w", err)
		}
		return nil
	}

	token := s.getUserToken(user)

	messages, err := s.gmailRepo.GetLatestMessages(ctx, token, maxResults)
	if err != nil {
		return fmt.Errorf("failed to get latest messages: %w", err)
	}

	if len(messages) == 0 {
		notificationText := "📭 メールはありません"
		if err := s.lineRepo.PushMessage(ctx, userID, notificationText); err != nil {
			return fmt.Errorf("failed to send LINE notification: %w", err)
		}
		return nil
	}

	notificationText := fmt.Sprintf("📨 最新メール (%d件)\n\n", len(messages))
	for i, message := range messages {
		notificationText += fmt.Sprintf("%d. %s\n件名: %s\n%s\n\n",
			i+1,
			message.From,
			message.Subject,
			message.Snippet,
		)
	}

	if err := s.lineRepo.PushMessage(ctx, userID, notificationText); err != nil {
		return fmt.Errorf("failed to send LINE notification: %w", err)
	}

	slog.Info("email list sent successfully",
		"user_id", userID,
		"count", len(messages),
	)

	return nil
}

func (s *Service) SendHelpMessage(ctx context.Context, userID string, message string) error {
	if err := s.lineRepo.PushMessage(ctx, userID, message); err != nil {
		return fmt.Errorf("failed to send help message: %w", err)
	}
	return nil
}

func (s *Service) StartGmailAuth(ctx context.Context, userID string) error {
	if s.gmailRepo == nil {
		errorMsg := "Gmail機能は現在利用できません。管理者にお問い合わせください。"
		if err := s.lineRepo.PushMessage(ctx, userID, errorMsg); err != nil {
			return fmt.Errorf("failed to send error message: %w", err)
		}
		return nil
	}

	// 認証URLを生成
	authURL := s.gmailRepo.GetAuthURL(userID)

	// 認証待ち状態に設定
	s.pendingAuth[userID] = true

	// 説明メッセージを送信
	instructionMsg := `Gmail連携を開始します。

【重要】以下の手順で認証してください：

1. 次のメッセージのURLを長押し
2. 「Safariで開く」または「Chromeで開く」を選択
3. Googleアカウントで認証

※ LINEアプリ内で開くとエラーになります`

	if err := s.lineRepo.PushMessage(ctx, userID, instructionMsg); err != nil {
		return fmt.Errorf("failed to send instruction message: %w", err)
	}

	// URLを別メッセージで送信
	if err := s.lineRepo.PushMessage(ctx, userID, authURL); err != nil {
		return fmt.Errorf("failed to send auth URL: %w", err)
	}

	// 完了メッセージを送信
	completionMsg := "認証が完了すると自動的に連携されます。"
	if err := s.lineRepo.PushMessage(ctx, userID, completionMsg); err != nil {
		return fmt.Errorf("failed to send completion message: %w", err)
	}

	slog.Info("Gmail auth started", "user_id", userID)
	return nil
}

func (s *Service) CompleteGmailAuth(ctx context.Context, userID string, authCode string) error {
	if s.gmailRepo == nil {
		return fmt.Errorf("Gmail repository not initialized")
	}

	// 認証コードをトークンに交換
	token, err := s.gmailRepo.ExchangeCode(ctx, authCode)
	if err != nil {
		delete(s.pendingAuth, userID)
		return fmt.Errorf("failed to exchange code: %w", err)
	}

	// DBに保存
	user, err := s.userRepo.GetUserByLineUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		// 新規ユーザー作成
		user = &user_repo.User{
			LineUserID: userID,
		}
		if err := s.userRepo.CreateUser(ctx, user); err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
	}

	// トークンを更新
	if err := s.userRepo.UpdateGmailTokens(ctx, userID, token); err != nil {
		delete(s.pendingAuth, userID)
		return fmt.Errorf("failed to save tokens: %w", err)
	}

	// Gmail Watch APIを登録（メールボックスの変更を監視）
	if s.pubsubTopic != "" {
		if err := s.gmailRepo.WatchMailbox(ctx, token, s.pubsubTopic); err != nil {
			slog.Warn("failed to setup Gmail watch, push notifications may not work",
				"user_id", userID,
				"error", err,
			)
			// Watch APIの失敗は致命的ではないので、処理を続行
		} else {
			slog.Info("Gmail watch setup successfully", "user_id", userID, "topic", s.pubsubTopic)
		}
	}

	// 認証待ち状態を解除
	delete(s.pendingAuth, userID)

	// 成功メッセージ
	successMsg := "✅ Gmail連携が完了しました！\n\n新着メールが届くと自動で通知されます。\n\n手動確認: 「未読mail」または「mail一覧」を送信"
	if err := s.lineRepo.PushMessage(ctx, userID, successMsg); err != nil {
		return fmt.Errorf("failed to send success message: %w", err)
	}

	slog.Info("Gmail auth completed", "user_id", userID)
	return nil
}

func (s *Service) IsAuthPending(userID string) bool {
	return s.pendingAuth[userID]
}

func (s *Service) ProcessGmailPushNotification(ctx context.Context) error {
	// Get all active users
	users, err := s.userRepo.GetAllActiveUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active users: %w", err)
	}

	for _, user := range users {
		if user.GmailAccessToken == nil || *user.GmailAccessToken == "" {
			continue // Skip users without Gmail authentication
		}

		token := s.getUserToken(&user)

		// Get new unread messages
		messages, err := s.gmailRepo.GetUnreadMessages(ctx, token, 5)
		if err != nil {
			slog.Error("failed to get unread messages",
				"user_id", user.LineUserID,
				"error", err,
			)
			continue
		}

		// Send notifications for new unread messages
		for _, message := range messages {
			notificationText := fmt.Sprintf(
				"📧 新着メール\n\n差出人: %s\n件名: %s\n\n%s",
				message.From,
				message.Subject,
				message.Snippet,
			)

			if err := s.lineRepo.PushMessage(ctx, user.LineUserID, notificationText); err != nil {
				slog.Error("failed to send LINE notification",
					"user_id", user.LineUserID,
					"message_id", message.ID,
					"error", err,
				)
				continue
			}

			slog.Info("push notification sent successfully",
				"user_id", user.LineUserID,
				"message_id", message.ID,
				"subject", message.Subject,
			)
		}
	}

	return nil
}
