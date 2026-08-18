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
	log.Printf("total releases: %d", len(releases))

	releases = dedup(releases)
	log.Printf("total releases after dedup: %d", len(releases))

	if token := os.Getenv("DISCOGS_TOKEN"); token != "" {
		releases = enrich(ctx, releases, discogs.NewClient(token))
	}

	releases = filterByDate(releases, targetDate)
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

// enrich queries Discogs for every release and, when a match is found, overwrites
// the Date with the exact Discogs date and sets CoverURL if not already set.
func enrich(ctx context.Context, releases []publisher.MusicRelease, dc *discogs.Client) []publisher.MusicRelease {
	out := make([]publisher.MusicRelease, len(releases))
	totalEnriched := 0
	for i, r := range releases {
		info, err := dc.FetchReleaseInfo(ctx, r.Title, r.Artist)
		if err != nil {
			log.Printf("discogs enrich %q by %q: %v", r.Title, r.Artist, err)
		}
		if info != nil {
			enriched := false
			if info.Date != "" {
				r.Date = info.Date
				enriched = true
			}
			if r.CoverURL == "" && info.CoverURL != "" {
				r.CoverURL = info.CoverURL
				enriched = true
			}
			if enriched {
				totalEnriched++
			}
		}
		out[i] = r
	}
	log.Printf("total enriched releases: %d/%d", totalEnriched, len(releases))
	return out
}

// filterByDate keeps only releases whose Date exactly matches targetDate.
// Releases with a missing, partial, or non-matching date are dropped.
func filterByDate(releases []publisher.MusicRelease, targetDate string) []publisher.MusicRelease {
	out := releases[:0:0]
	for _, r := range releases {
		if r.Date == targetDate {
			out = append(out, r)
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
