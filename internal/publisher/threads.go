package publisher

import (
	"context"
	"fmt"
	"strings"

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

func (t *ThreadsPublisher) Publish(ctx context.Context, releases []MusicRelease) error {
	topicTag := "Rock Music"
	for _, r := range releases {
		if strings.Contains(r.Genre, "Metal") {
			topicTag = "Metal Threads"
			break
		}
	}

	if len(releases) == 1 {
		return t.publishSingle(ctx, topicTag, releases[0])
	}

	return t.publishMultiple(ctx, topicTag, releases)
}

func (t *ThreadsPublisher) publishSingle(ctx context.Context, topicTag string, release MusicRelease) error {
	content := &threads.ImagePostContent{
		Text:     formatMessageSingle(release),
		ImageURL: release.CoverURL,
		TopicTag: topicTag,
	}

	_, err := t.client.CreateImagePost(ctx, content)
	if err != nil {
		return fmt.Errorf("threads: create post: %w", err)
	}
	return nil
}

func (t *ThreadsPublisher) publishMultiple(ctx context.Context, topicTag string, releases []MusicRelease) error {
	var containerIDs []string
	for _, r := range releases {
		c, err := t.client.CreateMediaContainer(ctx, threads.MediaTypeImage, r.CoverURL, r.Title)
		if err != nil {
			return fmt.Errorf("threads: create container: %w", err)
		}
		containerIDs = append(containerIDs, c.String())
	}

	content := &threads.CarouselPostContent{
		Text:     formatMessageMultiple(releases),
		Children: containerIDs,
		TopicTag: topicTag,
	}

	_, err := t.client.CreateCarouselPost(ctx, content)
	if err != nil {
		return fmt.Errorf("threads: create post: %w", err)
	}
	return nil
}
