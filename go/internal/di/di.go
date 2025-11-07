package di

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/go-sql-driver/mysql"
	emaildomain "github.com/huavcjj/flux/internal/domain/email"
	gmaildomain "github.com/huavcjj/flux/internal/domain/gmail"
	linedomain "github.com/huavcjj/flux/internal/domain/line"
	userdomain "github.com/huavcjj/flux/internal/domain/user"
	emailrepo "github.com/huavcjj/flux/internal/infrastructure/repository/email"
	gmailrepo "github.com/huavcjj/flux/internal/infrastructure/repository/gmail"
	linerepo "github.com/huavcjj/flux/internal/infrastructure/repository/line"
	userrepo "github.com/huavcjj/flux/internal/infrastructure/repository/user"
	"github.com/huavcjj/flux/internal/service/notification"
)

type Container struct {
	DB                  *sql.DB
	GmailRepo           gmaildomain.GmailRepo
	LineRepo            linedomain.LineRepo
	UserRepo            userdomain.UserRepo
	EmailRepo           emaildomain.EmailRepo
	NotificationService *notification.Service
}

type Config struct {
	LineChannelToken     string
	GmailCredentialsPath string
	DBHost               string
	DBPort               string
	DBUser               string
	DBPassword           string
	DBName               string
}

func NewContainer(ctx context.Context, cfg Config) (*Container, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		slog.Warn("database ping failed", "error", err)
	} else {
		slog.Info("database connected")
	}

	gmailRepo, err := gmailrepo.NewGmailRepo(ctx, cfg.GmailCredentialsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Gmail repository: %w", err)
	}

	lineRepo, err := linerepo.NewLineRepo(cfg.LineChannelToken)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize LINE repository: %w", err)
	}

	userRepo := userrepo.NewUserRepo(db)
	emailRepo := emailrepo.NewEmailRepo(db)

	notificationService := notification.NewService(
		gmailRepo,
		lineRepo,
		userRepo,
		emailRepo,
	)

	return &Container{
		DB:                  db,
		GmailRepo:           gmailRepo,
		LineRepo:            lineRepo,
		UserRepo:            userRepo,
		EmailRepo:           emailRepo,
		NotificationService: notificationService,
	}, nil
}

func (c *Container) Close() error {
	if c.DB != nil {
		return c.DB.Close()
	}
	return nil
}
