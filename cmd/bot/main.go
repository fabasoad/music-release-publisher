package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"music-release-publisher/internal/ai"
	"music-release-publisher/internal/discogs"
	"music-release-publisher/internal/musicbrainz"
	"music-release-publisher/internal/publisher"
)

func main() {
	ctx := context.Background()

	providers := buildReleaseProviders(ctx)
	if len(providers) == 0 {
		log.Fatal("no release providers configured — set GEMINI_API_KEY or MUSICBRAINZ_ENABLED=true")
	}

	publishers := buildPublishers()
	if len(publishers) == 0 {
		log.Fatal("no publishers configured — set at least one platform's env vars")
	}

	var dc *discogs.Client
	if token := os.Getenv("DISCOGS_TOKEN"); token != "" {
		dc = discogs.NewClient(token)
	}

	targetDate := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")

	var releases []publisher.MusicRelease
	for _, rp := range providers {
		r, err := rp.FetchReleases(ctx)
		if err != nil {
			log.Printf("fetch releases: %v", err)
			continue
		}
		log.Printf("fetched %d releases from %T", len(r), rp)
		releases = append(releases, r...)
	}

	releases = dedup(releases)
	log.Printf("total releases after dedup: %d", len(releases))

	releases = filterByDate(ctx, releases, targetDate, dc)
	log.Printf("total releases after date filter: %d", len(releases))

	chunkSize := 4
	for i := 0; i < len(releases); i += chunkSize {
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

// filterByDate partitions releases into exact matches (Date == targetDate) and
// partial matches (Date is a non-empty prefix of targetDate, e.g. "2026" or
// "2026-08"). Exact matches pass through directly. Partial matches are verified
// against Discogs; when no Discogs client is configured they are dropped.
// Releases with an empty or unrelated Date are dropped unconditionally.
func filterByDate(ctx context.Context, releases []publisher.MusicRelease, targetDate string, dc *discogs.Client) []publisher.MusicRelease {
	out := releases[:0:0]
	for _, r := range releases {
		switch {
		case r.Date == targetDate:
			out = append(out, r)
		case r.Date != "" && strings.HasPrefix(targetDate, r.Date):
			if dc == nil {
				log.Printf("skipping partial-date release %q by %q (no Discogs client)", r.Title, r.Artist)
				continue
			}
			info, err := dc.VerifyReleaseDate(ctx, r.Title, r.Artist, targetDate)
			if err != nil {
				log.Printf("discogs verify %q by %q: %v", r.Title, r.Artist, err)
			}
			if info != nil {
				r.Date = targetDate
				if r.CoverURL == "" && info.CoverURL != "" {
					r.CoverURL = info.CoverURL
				}
				out = append(out, r)
			}
		}
	}
	return out
}

func dedup(releases []publisher.MusicRelease) []publisher.MusicRelease {
	seen := make(map[string]struct{}, len(releases))
	out := releases[:0:0]
	for _, r := range releases {
		key := strings.ToLower(strings.TrimSpace(r.Artist)) + "\x00" + strings.ToLower(strings.TrimSpace(r.Title))
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			out = append(out, r)
		}
	}
	return out
}

func buildReleaseProviders(ctx context.Context) []publisher.ReleaseProvider {
	var providers []publisher.ReleaseProvider

	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		p, err := ai.NewCurator(ctx, key)
		if err != nil {
			log.Printf("gemini: init failed: %v", err)
		} else {
			providers = append(providers, p)
		}
	}

	if os.Getenv("MUSICBRAINZ_ENABLED") == "true" {
		providers = append(providers, musicbrainz.NewProvider())
	}

	return providers
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
