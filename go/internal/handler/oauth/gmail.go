package oauth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/huavcjj/flux/internal/service/notification"
)

const (
	htmlError   = `<html><body><h1>❌ 認証失敗</h1></body></html>`
	htmlSuccess = `<html><body><h1>✅ 認証完了</h1></body></html>`
)

type GmailOAuthHandler struct {
	notificationService *notification.Service
}

func NewGmailOAuthHandler(notificationService *notification.Service) *GmailOAuthHandler {
	return &GmailOAuthHandler{
		notificationService: notificationService,
	}
}

func (h *GmailOAuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		slog.Error("missing code or state", "code", code, "state", state)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if err := h.notificationService.CompleteGmailAuth(context.Background(), state, code); err != nil {
		slog.Error("failed to complete Gmail auth", "user_id", state, "error", err)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, htmlError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, htmlSuccess)
}
