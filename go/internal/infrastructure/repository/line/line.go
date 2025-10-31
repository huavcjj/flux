package line

import (
	"context"
	"fmt"

	line_repo "github.com/huavcjj/flux/internal/domain/line"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

type lineRepo struct {
	bot *messaging_api.MessagingApiAPI
}

var _ line_repo.LineRepo = (*lineRepo)(nil)

func NewLineRepo(channelToken string) (line_repo.LineRepo, error) {
	if channelToken == "" {
		return nil, fmt.Errorf("line channel token is empty")
	}

	bot, err := messaging_api.NewMessagingApiAPI(channelToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create messaging API: %w", err)
	}

	return &lineRepo{
		bot: bot,
	}, nil
}

func (r *lineRepo) SendTextMessage(ctx context.Context, userID, message string) error {
	if userID == "" {
		return fmt.Errorf("user ID is empty")
	}

	_, err := r.bot.PushMessage(
		&messaging_api.PushMessageRequest{
			To: userID,
			Messages: []messaging_api.MessageInterface{
				messaging_api.TextMessage{
					Text: message,
				},
			},
		},
		"",
	)
	if err != nil {
		return fmt.Errorf("failed to send text message: %w", err)
	}

	return nil
}

func (r *lineRepo) PushMessage(ctx context.Context, userID, message string) error {
	return r.SendTextMessage(ctx, userID, message)
}

func (r *lineRepo) SendButtonMessage(ctx context.Context, userID, text, buttonText, buttonURL string) error {
	if userID == "" {
		return fmt.Errorf("user ID is empty")
	}

	_, err := r.bot.PushMessage(
		&messaging_api.PushMessageRequest{
			To: userID,
			Messages: []messaging_api.MessageInterface{
				&messaging_api.TemplateMessage{
					AltText: text,
					Template: &messaging_api.ButtonsTemplate{
						Text: text,
						Actions: []messaging_api.ActionInterface{
							&messaging_api.UriAction{
								Label: buttonText,
								Uri:   buttonURL,
							},
						},
					},
				},
			},
		},
		"",
	)
	if err != nil {
		return fmt.Errorf("failed to send button message: %w", err)
	}

	return nil
}
