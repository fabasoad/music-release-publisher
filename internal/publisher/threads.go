package publisher

import (
	"context"
	"fmt"

	"github.com/tirthpatell/threads-go"
)

type ThreadsPublisher struct {
	client *threads.Client
}

// NewThreadsPublisher creates a publisher using a pre-obtained long-lived access token.
// clientID and clientSecret are still required by the SDK config validation.
func NewThreadsPublisher(clientID, clientSecret, redirectURI, accessToken string) (*ThreadsPublisher, error) {
	cfg := &threads.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  redirectURI,
		Scopes:       []string{"threads_basic", "threads_content_publish"},
	}
	client, err := threads.NewClientWithToken(accessToken, cfg)
	if err != nil {
		return nil, fmt.Errorf("threads: create client: %w", err)
	}
	return &ThreadsPublisher{client: client}, nil
}

func (t *ThreadsPublisher) Name() string { return "Threads" }

func (t *ThreadsPublisher) Publish(ctx context.Context, release MusicRelease) error {
	content := &threads.TextPostContent{
		Text: formatMessage(release),
	}
	_, err := t.client.CreateTextPost(ctx, content)
	if err != nil {
		return fmt.Errorf("threads: create post: %w", err)
	}
	return nil
}
