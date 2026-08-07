package publisher

import (
	"context"
	"fmt"
	"strings"
	"time"

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
		Text:     truncateText(formatMessageSingle(release), 500),
		ImageURL: release.CoverURL,
		TopicTag: topicTag,
	}

	_, err := t.client.CreateImagePost(ctx, content)
	if err != nil {
		return fmt.Errorf("threads: create image post: %w", err)
	}
	return nil
}

func (t *ThreadsPublisher) waitForContainer(ctx context.Context, id threads.ContainerID) error {
	for range threads.DefaultContainerPollMaxAttempts {
		status, err := t.client.GetContainerStatus(ctx, id)
		if err != nil {
			return fmt.Errorf("get status: %w", err)
		}
		switch status.Status {
		case threads.ContainerStatusFinished:
			return nil
		case threads.ContainerStatusError:
			return fmt.Errorf("container processing failed: %s", status.ErrorMessage)
		case threads.ContainerStatusExpired:
			return fmt.Errorf("container expired before publish")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(threads.DefaultContainerPollInterval):
		}
	}
	return fmt.Errorf("container not ready after %d attempts", threads.DefaultContainerPollMaxAttempts)
}

func (t *ThreadsPublisher) publishMultiple(ctx context.Context, topicTag string, releases []MusicRelease) error {
	var containerIDs []string
	for _, r := range releases {
		c, err := t.client.CreateMediaContainer(ctx, threads.MediaTypeImage, r.CoverURL, r.Title)
		if err != nil {
			return fmt.Errorf("threads: create container: %w", err)
		}
		if err := t.waitForContainer(ctx, c); err != nil {
			return fmt.Errorf("threads: container %s not ready: %w", c, err)
		}
		containerIDs = append(containerIDs, c.String())
	}

	content := &threads.CarouselPostContent{
		Text:     truncateText(formatMessageMultiple(releases), 500),
		Children: containerIDs,
		TopicTag: topicTag,
	}

	_, err := t.client.CreateCarouselPost(ctx, content)
	if err != nil {
		return fmt.Errorf("threads: create carousel post: %w", err)
	}
	return nil
}
