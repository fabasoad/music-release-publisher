package main

import (
	"context"
	"log"
	"os"

	"music-release-publisher/internal/ai"
	"music-release-publisher/internal/musicbrainz"
	"music-release-publisher/internal/publisher"
)

func main() {
	ctx := context.Background()

	provider := buildReleaseProvider(ctx)

	publishers := buildPublishers()
	if len(publishers) == 0 {
		log.Fatal("no publishers configured — set at least one platform's env vars")
	}

	releases, err := provider.FetchReleases(ctx)
	if err != nil {
		log.Fatalf("fetch releases: %v", err)
	}
	log.Printf("fetched %d releases", len(releases))

	chunkSize := 4
	for i := 0; i < len(releases); i += chunkSize {
		// Ensure the index does not go out of bounds on the last chunk
		end := min(i+chunkSize, len(releases))

		chunk := releases[i:end]

		for _, p := range publishers {
			if err := p.Publish(ctx, chunk); err != nil {
				log.Printf("[%s] publish failed: %v", p.Name(), err)
			} else {
				log.Printf("[%s] published successfully", p.Name())
			}
		}
	}
}

func buildReleaseProvider(ctx context.Context) publisher.ReleaseProvider {
	name := os.Getenv("RELEASE_PROVIDER")
	if name == "" {
		name = "gemini"
	}
	switch name {
	case "gemini":
		p, err := ai.NewCurator(ctx, mustEnv("GEMINI_API_KEY"))
		if err != nil {
			log.Fatalf("init gemini provider: %v", err)
		}
		return p
	case "musicbrainz":
		return musicbrainz.NewProvider()
	default:
		log.Fatalf("unknown RELEASE_PROVIDER %q", name)
		return nil
	}
}

func buildPublishers() []publisher.Publisher {
	var publishers []publisher.Publisher

	if url := os.Getenv("DISCORD_WEBHOOK_URL"); url != "" {
		publishers = append(publishers, publisher.NewDiscordPublisher(url))
	}

	if token, chatID := os.Getenv("TELEGRAM_BOT_TOKEN"), os.Getenv("TELEGRAM_CHAT_ID"); token != "" && chatID != "" {
		publishers = append(publishers, publisher.NewTelegramPublisher(token, chatID))
	}

	clientID := os.Getenv("THREADS_CLIENT_ID")
	clientSecret := os.Getenv("THREADS_CLIENT_SECRET")
	redirectURI := os.Getenv("THREADS_REDIRECT_URI")
	accessToken := os.Getenv("THREADS_ACCESS_TOKEN")
	if clientID != "" && clientSecret != "" && redirectURI != "" && accessToken != "" {
		tp, err := publisher.NewThreadsPublisher(clientID, clientSecret, redirectURI, accessToken)
		if err != nil {
			log.Printf("threads: init failed: %v", err)
		} else {
			publishers = append(publishers, tp)
		}
	}

	return publishers
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}
