package oauth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/huavcjj/flux/internal/service/notification"
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

	if code == "" {
		slog.Error("missing authorization code")
		http.Error(w, "認証コードが見つかりません", http.StatusBadRequest)
		return
	}

	if state == "" {
		slog.Error("missing state (user ID)")
		http.Error(w, "ユーザー情報が見つかりません", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	if err := h.notificationService.CompleteGmailAuth(ctx, state, code); err != nil {
		slog.Error("failed to complete Gmail auth", "user_id", state, "error", err)
		h.renderError(w, err)
		return
	}

	h.renderSuccess(w)
}

func (h *GmailOAuthHandler) renderError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>認証エラー</title>
<style>
body { font-family: Arial, sans-serif; text-align: center; padding: 50px; }
.error { color: #d32f2f; }
</style>
</head>
<body>
<h1 class="error">❌ 認証に失敗しました</h1>
<p>LINEで「Gmail連携」を送信して、もう一度やり直してください。</p>
<p>エラー: %s</p>
</body>
</html>`, err.Error())
}

func (h *GmailOAuthHandler) renderSuccess(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>認証完了</title>
<style>
body { font-family: Arial, sans-serif; text-align: center; padding: 50px; }
.success { color: #388e3c; }
</style>
</head>
<body>
<h1 class="success">✅ Gmail連携が完了しました！</h1>
<p>このページを閉じて、LINEに戻ってください。</p>
<p>「未読mail」または「mail一覧」を送信して、メールを確認できます。</p>
</body>
</html>`)
}
