package publisher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type TelegramPublisher struct {
	botToken string
	chatID   string
	client   *http.Client
}

func NewTelegramPublisher(botToken, chatID string) *TelegramPublisher {
	return &TelegramPublisher{
		botToken: botToken,
		chatID:   chatID,
		client:   &http.Client{},
	}
}

func (t *TelegramPublisher) Name() string { return "Telegram" }

func (t *TelegramPublisher) Publish(ctx context.Context, releases []MusicRelease) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)
	payload := map[string]string{
		"chat_id": t.chatID,
		"text":    formatMessageMultiple(releases),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("telegram: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram: unexpected status %d", resp.StatusCode)
	}
	return nil
}
