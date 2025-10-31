package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/huavcjj/flux/internal/domain/gmail"
	"github.com/huavcjj/flux/internal/handler/oauth"
	"github.com/huavcjj/flux/internal/handler/webhook"
	gmail_repo "github.com/huavcjj/flux/internal/infrastructure/repository/gmail"
	line_repo "github.com/huavcjj/flux/internal/infrastructure/repository/line"
	user_repo "github.com/huavcjj/flux/internal/infrastructure/repository/user"
	"github.com/huavcjj/flux/internal/service/notification"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Warn("no .env file found")
	}

	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer cancel()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	lineChannelToken := os.Getenv("LINE_CHANNEL_TOKEN")
	lineChannelSecret := os.Getenv("LINE_CHANNEL_SECRET")
	gmailCredentialsPath := os.Getenv("GMAIL_CREDENTIALS_PATH")
	pubsubTopic := os.Getenv("PUBSUB_TOPIC")

	if pubsubTopic == "" {
		pubsubTopic = "projects/line-gmail-bot/topics/gmail-notifications"
	}

	slog.Info("environment variables loaded",
		"line_token_set", lineChannelToken != "",
		"line_secret_set", lineChannelSecret != "",
		"pubsub_topic", pubsubTopic,
	)

	var gmailRepo gmail.GmailRepo
	if gmailCredentialsPath != "" {
		var err error
		gmailRepo, err = gmail_repo.NewGmailRepo(ctx, gmailCredentialsPath)
		if err != nil {
			slog.Warn("failed to initialize Gmail repository, continuing without Gmail", "error", err)
		} else {
			slog.Info("Gmail repository initialized successfully")
		}
	} else {
		slog.Warn("Gmail credentials not configured, Gmail features will be disabled")
	}

	lineRepo, err := line_repo.NewLineRepo(lineChannelToken)
	if err != nil {
		slog.Error("failed to initialize LINE repository", "error", err)
		os.Exit(1)
	}

	// Database接続
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", dbUser, dbPassword, dbHost, dbPort, dbName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		slog.Warn("database connection check failed, continuing without DB features", "error", err)
	} else {
		slog.Info("database connected successfully")
	}

	userRepo := user_repo.NewUserRepo(db)

	notificationService := notification.NewService(gmailRepo, lineRepo, userRepo, pubsubTopic)

	lineWebhookHandler := webhook.NewLineWebhookHandler(notificationService, lineChannelSecret)
	pubsubWebhookHandler := webhook.NewPubSubWebhookHandler(notificationService)
	gmailOAuthHandler := oauth.NewGmailOAuthHandler(notificationService)

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook/line", lineWebhookHandler.HandleWebhook)
	mux.HandleFunc("/webhook/pubsub", pubsubWebhookHandler.HandlePubSub)
	mux.HandleFunc("/oauth/gmail/callback", gmailOAuthHandler.HandleCallback)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		slog.Info(fmt.Sprintf("starting Gmail-LINE bot server on port %s...", port))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("failed to start server", "error", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	slog.Info("shutting down server gracefully...")
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("failed to shutdown server gracefully", "error", err)
		os.Exit(1)
	}
	slog.Info("server shutdown completed")
}
