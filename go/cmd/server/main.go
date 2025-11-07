package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/huavcjj/flux/internal/di"
	"github.com/huavcjj/flux/internal/handler/oauth"
	"github.com/huavcjj/flux/internal/handler/webhook"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Warn("no .env file found")
	}

	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	if err := run(ctx); err != nil {
		slog.Error("application error", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	port := getEnv("PORT", "8080")

	cfg := di.Config{
		LineChannelToken:     os.Getenv("LINE_CHANNEL_TOKEN"),
		GmailCredentialsPath: os.Getenv("GMAIL_CREDENTIALS_PATH"),
		DBHost:               os.Getenv("DB_HOST"),
		DBPort:               os.Getenv("DB_PORT"),
		DBUser:               os.Getenv("DB_USER"),
		DBPassword:           os.Getenv("DB_PASSWORD"),
		DBName:               os.Getenv("DB_NAME"),
	}

	container, err := di.NewContainer(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize container: %w", err)
	}
	defer container.Close()

	lineWebhookHandler := webhook.NewLineWebhookHandler(container.NotificationService)
	pubsubWebhookHandler := webhook.NewPubSubWebhookHandler(container.NotificationService)
	gmailOAuthHandler := oauth.NewGmailOAuthHandler(container.NotificationService)

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook/line", lineWebhookHandler.HandleWebhook)
	mux.HandleFunc("/webhook/pubsub", pubsubWebhookHandler.HandlePubSub)
	mux.HandleFunc("/oauth/gmail/callback", gmailOAuthHandler.HandleCallback)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("starting server", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutting down...")
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown error: %w", err)
	}

	slog.Info("shutdown completed")
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
