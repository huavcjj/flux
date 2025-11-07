package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/huavcjj/flux/internal/service/notification"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

type LineWebhookHandler struct {
	notificationService *notification.Service
	channelSecret       string
}

func NewLineWebhookHandler(notificationService *notification.Service) *LineWebhookHandler {
	return &LineWebhookHandler{
		notificationService: notificationService,
		channelSecret:       os.Getenv("LINE_CHANNEL_SECRET"),
	}
}

func (h *LineWebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Allow GET for verification
	if r.Method == http.MethodGet {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse webhook request (includes signature validation)
	cb, err := webhook.ParseRequest(h.channelSecret, r)
	if err != nil {
		slog.Error("failed to parse webhook request", "error", err)
		http.Error(w, "Failed to parse request", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Process each event
	for _, event := range cb.Events {
		switch e := event.(type) {
		case webhook.MessageEvent:
			h.handleMessageEvent(ctx, e)
		default:
			slog.Info("received unhandled event", "type", fmt.Sprintf("%T", event))
		}
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

func (h *LineWebhookHandler) handleMessageEvent(ctx context.Context, event webhook.MessageEvent) {
	switch message := event.Message.(type) {
	case webhook.TextMessageContent:
		h.handleTextMessage(ctx, event.Source, message.Text)
	default:
		slog.Info("received unhandled message type", "type", fmt.Sprintf("%T", message))
	}
}

func (h *LineWebhookHandler) handleTextMessage(ctx context.Context, source webhook.SourceInterface, text string) {
	// Extract user ID
	var userID string
	sourceData, _ := json.Marshal(source)
	var sourceMap map[string]interface{}
	if err := json.Unmarshal(sourceData, &sourceMap); err == nil {
		if uid, ok := sourceMap["userId"].(string); ok {
			userID = uid
		}
	}

	if userID == "" {
		slog.Error("could not extract user ID from source")
		return
	}

	slog.Info("received text message",
		"user_id", userID,
		"text", text,
	)

	// Process text message
	text = strings.TrimSpace(text)
	var err error

	// 認証待ちの場合は認証コードとして処理
	if h.notificationService.IsAuthPending(userID) {
		err = h.notificationService.CompleteGmailAuth(ctx, userID, text)
		if err != nil {
			errorMsg := "認証に失敗しました。もう一度「Gmail連携」を送信してやり直してください。\n\nエラー: " + err.Error()
			h.notificationService.SendHelpMessage(ctx, userID, errorMsg)
		}
		return
	}

	switch text {
	case "Gmail連携":
		err = h.notificationService.StartGmailAuth(ctx, userID)
	case "未読mail":
		err = h.notificationService.SendUnreadEmailList(ctx, userID)
	case "mail一覧":
		err = h.notificationService.SendEmailList(ctx, userID, 10)
	default:
		// Send help message
		helpMessage := "使用可能なコマンド:\n• Gmail連携 - Gmailアカウントと連携\n• 未読mail - 未読メールの一覧を表示\n• mail一覧 - 最新メール10件を表示"
		err = h.notificationService.SendHelpMessage(ctx, userID, helpMessage)
	}

	if err != nil {
		slog.Error("failed to process text message",
			"user_id", userID,
			"text", text,
			"error", err,
		)
	}
}
