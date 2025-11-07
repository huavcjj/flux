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

const (
	cmdGmailAuth  = "Gmail連携"
	cmdUnreadMail = "未読mail"
	cmdMailList   = "mail一覧"
	mailListLimit = 10
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
	if r.Method == http.MethodGet {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cb, err := webhook.ParseRequest(h.channelSecret, r)
	if err != nil {
		slog.Error("failed to parse webhook request", "error", err)
		http.Error(w, "Failed to parse request", http.StatusBadRequest)
		return
	}

	for _, event := range cb.Events {
		if msgEvent, ok := event.(webhook.MessageEvent); ok {
			h.handleMessageEvent(r.Context(), msgEvent)
		}
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
}

func (h *LineWebhookHandler) handleMessageEvent(ctx context.Context, event webhook.MessageEvent) {
	textMsg, ok := event.Message.(webhook.TextMessageContent)
	if !ok {
		return
	}

	userID := h.extractUserID(event.Source)
	if userID == "" {
		slog.Error("could not extract user ID from source")
		return
	}

	h.processTextMessage(ctx, userID, textMsg.Text)
}

func (h *LineWebhookHandler) extractUserID(source webhook.SourceInterface) string {
	sourceData, _ := json.Marshal(source)
	var sourceMap map[string]interface{}
	if err := json.Unmarshal(sourceData, &sourceMap); err != nil {
		return ""
	}

	userID, _ := sourceMap["userId"].(string)
	return userID
}

func (h *LineWebhookHandler) processTextMessage(ctx context.Context, userID, text string) {
	text = strings.TrimSpace(text)
	slog.Info("received text message", "user_id", userID, "text", text)

	if h.notificationService.IsAuthPending(userID) {
		h.handleAuthCode(ctx, userID, text)
		return
	}

	var err error
	switch text {
	case cmdGmailAuth:
		err = h.notificationService.StartGmailAuth(ctx, userID)
	case cmdUnreadMail:
		err = h.notificationService.SendUnreadEmailList(ctx, userID)
	case cmdMailList:
		err = h.notificationService.SendEmailList(ctx, userID, mailListLimit)
	}

	if err != nil {
		slog.Error("failed to process text message", "user_id", userID, "text", text, "error", err)
	}
}

func (h *LineWebhookHandler) handleAuthCode(ctx context.Context, userID, code string) {
	if err := h.notificationService.CompleteGmailAuth(ctx, userID, code); err != nil {
		slog.Error("failed to complete Gmail auth", "user_id", userID, "error", err)
	}
}
