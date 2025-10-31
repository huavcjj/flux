package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/huavcjj/flux/internal/service/notification"
)

// PubSubMessage represents a message from Google Cloud Pub/Sub
type PubSubMessage struct {
	Message struct {
		Data        string            `json:"data"`
		MessageID   string            `json:"messageId"`
		Attributes  map[string]string `json:"attributes"`
		PublishTime string            `json:"publishTime"`
	} `json:"message"`
	Subscription string `json:"subscription"`
}

// GmailNotification represents the notification data from Gmail
type GmailNotification struct {
	EmailAddress string `json:"emailAddress"`
	HistoryID    uint64 `json:"historyId"`
}

type PubSubWebhookHandler struct {
	notificationService *notification.Service
}

func NewPubSubWebhookHandler(notificationService *notification.Service) *PubSubWebhookHandler {
	return &PubSubWebhookHandler{
		notificationService: notificationService,
	}
}

func (h *PubSubWebhookHandler) HandlePubSub(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var pubsubMsg PubSubMessage
	if err := json.NewDecoder(r.Body).Decode(&pubsubMsg); err != nil {
		slog.Error("failed to decode pubsub message", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	// Process the Gmail notification
	if err := h.processGmailNotification(ctx, pubsubMsg); err != nil {
		slog.Error("failed to process Gmail notification", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

func (h *PubSubWebhookHandler) processGmailNotification(ctx context.Context, msg PubSubMessage) error {
	slog.Info("received Gmail notification",
		"message_id", msg.Message.MessageID,
		"publish_time", msg.Message.PublishTime,
	)

	// Decode the base64 encoded data
	// The data contains email address and history ID
	// For now, we'll process notifications for all users
	// In production, you should decode the data and match the email address to a user

	// Process notification for all users
	if err := h.notificationService.ProcessGmailPushNotification(ctx); err != nil {
		return fmt.Errorf("failed to process push notification: %w", err)
	}

	return nil
}
