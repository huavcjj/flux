package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/huavcjj/flux/internal/service/notification"
)

type PubSubMessage struct {
	Message struct {
		Data        string            `json:"data"`
		MessageID   string            `json:"messageId"`
		Attributes  map[string]string `json:"attributes"`
		PublishTime string            `json:"publishTime"`
	} `json:"message"`
	Subscription string `json:"subscription"`
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

	var msg PubSubMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		slog.Error("failed to decode pubsub message", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	slog.Info("received Gmail notification", "message_id", msg.Message.MessageID, "publish_time", msg.Message.PublishTime)

	if err := h.notificationService.ProcessGmailPushNotification(context.Background()); err != nil {
		slog.Error("failed to process Gmail notification", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
}
