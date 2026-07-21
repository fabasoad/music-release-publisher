package main

import (
	"context"
	"log"
	"os"

	"music-release-publisher/internal/ai"
	"music-release-publisher/internal/publisher"
)

func main() {
	ctx := context.Background()

	geminiKey := mustEnv("GEMINI_API_KEY")
	curator, err := ai.NewCurator(ctx, geminiKey)
	if err != nil {
		log.Fatalf("init curator: %v", err)
	}

	publishers := buildPublishers()
	if len(publishers) == 0 {
		log.Fatal("no publishers configured — set at least one platform's env vars")
	}

	releases, err := curator.FetchReleases(ctx)
	if err != nil {
		log.Fatalf("fetch releases: %v", err)
	}
	log.Printf("fetched %d releases", len(releases))

	for _, release := range releases {
		for _, p := range publishers {
			if err := p.Publish(ctx, release); err != nil {
				log.Printf("[%s] publish %q by %s: %v", p.Name(), release.Title, release.Artist, err)
			} else {
				log.Printf("[%s] published %q by %s", p.Name(), release.Title, release.Artist)
			}
		}
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
