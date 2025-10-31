package gmail

import (
	"context"
	"time"

	"golang.org/x/oauth2"
)

type Message struct {
	ID       string
	ThreadID string
	From     string
	To       string
	Subject  string
	Snippet  string
	Date     time.Time
}

type GmailRepo interface {
	GetLatestMessages(ctx context.Context, token *oauth2.Token, maxResults int64) ([]*Message, error)
	GetUnreadMessages(ctx context.Context, token *oauth2.Token, maxResults int64) ([]*Message, error)
	WatchMailbox(ctx context.Context, token *oauth2.Token, topicName string) error
	GetMessage(ctx context.Context, token *oauth2.Token, messageID string) (*Message, error)
	GetHistoryMessages(ctx context.Context, token *oauth2.Token, startHistoryID uint64) ([]*Message, error)
	GetAuthURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error)
}
