package gmail

import (
	"context"
	"fmt"
	"os"
	"time"

	gmail_repo "github.com/huavcjj/flux/internal/domain/gmail"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type gmailRepo struct {
	config *oauth2.Config
	ctx    context.Context
}

var _ gmail_repo.GmailRepo = (*gmailRepo)(nil)

func NewGmailRepo(ctx context.Context, credentialsPath string) (gmail_repo.GmailRepo, error) {
	b, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read credentials file: %w", err)
	}

	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse credentials: %w", err)
	}

	return &gmailRepo{
		config: config,
		ctx:    ctx,
	}, nil
}

// getServiceWithToken creates a Gmail service using the provided OAuth token
func (r *gmailRepo) getServiceWithToken(token *oauth2.Token) (*gmail.Service, error) {
	client := r.config.Client(r.ctx, token)
	srv, err := gmail.NewService(r.ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to create gmail service: %w", err)
	}
	return srv, nil
}

func (r *gmailRepo) GetLatestMessages(ctx context.Context, token *oauth2.Token, maxResults int64) ([]*gmail_repo.Message, error) {
	service, err := r.getServiceWithToken(token)
	if err != nil {
		return nil, err
	}

	user := "me"
	msgs, err := service.Users.Messages.List(user).MaxResults(maxResults).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve messages: %w", err)
	}

	var messages []*gmail_repo.Message
	for _, m := range msgs.Messages {
		msg, err := r.GetMessage(ctx, token, m.Id)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func (r *gmailRepo) GetUnreadMessages(ctx context.Context, token *oauth2.Token, maxResults int64) ([]*gmail_repo.Message, error) {
	service, err := r.getServiceWithToken(token)
	if err != nil {
		return nil, err
	}

	user := "me"
	// Use label filtering instead of query to get only unread messages
	msgs, err := service.Users.Messages.List(user).LabelIds("UNREAD").MaxResults(maxResults).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve unread messages: %w", err)
	}

	var messages []*gmail_repo.Message
	for _, m := range msgs.Messages {
		// Get message with minimal format to verify labels
		minimalMsg, err := service.Users.Messages.Get(user, m.Id).Format("minimal").Do()
		if err != nil {
			continue
		}

		// Debug: Check if message actually has UNREAD label
		hasUnread := false
		for _, label := range minimalMsg.LabelIds {
			if label == "UNREAD" {
				hasUnread = true
				break
			}
		}

		if !hasUnread {
			continue // Skip if not actually unread
		}

		fullMsg, err := r.GetMessage(ctx, token, m.Id)
		if err != nil {
			continue // Skip messages we can't retrieve
		}
		messages = append(messages, fullMsg)
	}

	return messages, nil
}

func (r *gmailRepo) GetMessage(ctx context.Context, token *oauth2.Token, messageID string) (*gmail_repo.Message, error) {
	service, err := r.getServiceWithToken(token)
	if err != nil {
		return nil, err
	}

	user := "me"
	msg, err := service.Users.Messages.Get(user, messageID).Format("full").Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve message: %w", err)
	}

	var from, to, subject string
	var date time.Time

	for _, header := range msg.Payload.Headers {
		switch header.Name {
		case "From":
			from = header.Value
		case "To":
			to = header.Value
		case "Subject":
			subject = header.Value
		case "Date":
			parsedDate, err := time.Parse(time.RFC1123Z, header.Value)
			if err != nil {
				parsedDate = time.Now()
			}
			date = parsedDate
		}
	}

	snippet := msg.Snippet
	if len(snippet) > 100 {
		snippet = snippet[:100] + "..."
	}

	return &gmail_repo.Message{
		ID:       msg.Id,
		ThreadID: msg.ThreadId,
		From:     from,
		To:       to,
		Subject:  subject,
		Snippet:  snippet,
		Date:     date,
	}, nil
}

func (r *gmailRepo) WatchMailbox(ctx context.Context, token *oauth2.Token, topicName string) error {
	service, err := r.getServiceWithToken(token)
	if err != nil {
		return err
	}

	user := "me"
	watchRequest := &gmail.WatchRequest{
		TopicName:         topicName,
		LabelIds:          []string{"INBOX"},
		LabelFilterAction: "include",
	}

	_, err = service.Users.Watch(user, watchRequest).Do()
	if err != nil {
		return fmt.Errorf("unable to watch mailbox: %w", err)
	}

	return nil
}

func (r *gmailRepo) GetAuthURL(state string) string {
	return r.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (r *gmailRepo) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := r.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	return token, nil
}

func (r *gmailRepo) GetHistoryMessages(ctx context.Context, token *oauth2.Token, startHistoryID uint64) ([]*gmail_repo.Message, error) {
	service, err := r.getServiceWithToken(token)
	if err != nil {
		return nil, err
	}

	user := "me"

	// Get history since the given history ID
	historyList, err := service.Users.History.List(user).StartHistoryId(startHistoryID).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve history: %w", err)
	}

	// Extract unique message IDs from history
	messageIDs := make(map[string]bool)
	for _, history := range historyList.History {
		// Check for messages added
		for _, msgAdded := range history.MessagesAdded {
			// Only include messages with UNREAD label
			hasUnread := false
			for _, labelID := range msgAdded.Message.LabelIds {
				if labelID == "UNREAD" {
					hasUnread = true
					break
				}
			}
			if hasUnread {
				messageIDs[msgAdded.Message.Id] = true
			}
		}
	}

	// Fetch full message details
	var messages []*gmail_repo.Message
	for msgID := range messageIDs {
		msg, err := r.GetMessage(ctx, token, msgID)
		if err != nil {
			continue // Skip messages we can't retrieve
		}
		messages = append(messages, msg)
	}

	return messages, nil
}
