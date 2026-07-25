package publisher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type DiscordPublisher struct {
	webhookURL string
	client     *http.Client
}

func NewDiscordPublisher(webhookURL string) *DiscordPublisher {
	return &DiscordPublisher{
		webhookURL: webhookURL,
		client:     &http.Client{},
	}
}

func (d *DiscordPublisher) Name() string { return "Discord" }

func (d *DiscordPublisher) Publish(ctx context.Context, releases []MusicRelease) error {
	payload := map[string]string{
		"content": formatMessage(releases),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("discord: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("discord: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("discord: send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord: unexpected status %d", resp.StatusCode)
	}
	return nil
}
