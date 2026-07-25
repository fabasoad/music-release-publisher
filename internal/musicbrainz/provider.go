package musicbrainz

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"music-release-publisher/internal/genres"
	"music-release-publisher/internal/publisher"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	baseURL   = "https://musicbrainz.org/ws/2"
	userAgent = "music-release-publisher/1.0.0-beta ( fabasoad@gmail.com )"
)

type mbRelease struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Date         string `json:"date"`
	ArtistCredit []struct {
		Artist struct {
			Name string `json:"name"`
		} `json:"artist"`
	} `json:"artist-credit"`
	ReleaseGroup struct {
		PrimaryType string `json:"primary-type"`
	} `json:"release-group"`
	Tags []struct {
		Name string `json:"name"`
	} `json:"tags"`
}

type mbResponse struct {
	Releases []mbRelease `json:"releases"`
}

type Provider struct {
	client *http.Client
}

func NewProvider() *Provider {
	return &Provider{client: &http.Client{Timeout: 30 * time.Second}}
}

func (p *Provider) FetchReleases(ctx context.Context) ([]publisher.MusicRelease, error) {
	date := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")

	var tagParts []string
	for _, g := range genres.All {
		tagParts = append(tagParts, fmt.Sprintf("tag:%q", g))
	}
	query := fmt.Sprintf("date:%s AND (%s)", date, strings.Join(tagParts, " OR "))

	reqURL := fmt.Sprintf("%s/release?query=%s&fmt=json", baseURL, url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("musicbrainz: build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("musicbrainz: request: %w", err)
	}
	defer func(b io.ReadCloser) {
		_ = b.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("musicbrainz: unexpected status %d", resp.StatusCode)
	}

	var mbResp mbResponse
	if err := json.NewDecoder(resp.Body).Decode(&mbResp); err != nil {
		return nil, fmt.Errorf("musicbrainz: decode response: %w", err)
	}

	var releases []publisher.MusicRelease
	for _, r := range mbResp.Releases {
		if len(r.ArtistCredit) == 0 || r.Date != date {
			continue
		}
		releases = append(releases, publisher.MusicRelease{
			Artist:   r.ArtistCredit[0].Artist.Name,
			Title:    r.Title,
			Album:    r.Title,
			Type:     r.ReleaseGroup.PrimaryType,
			Genre:    joinGenres(r.Tags),
			CoverURL: fmt.Sprintf("https://coverartarchive.org/release/%s/front", r.ID),
		})
	}
	return releases, nil
}

func joinGenres(tags []struct {
	Name string `json:"name"`
}) string {
	parts := make([]string, 0, len(tags))
	for _, t := range tags {
		parts = append(parts, cases.Title(language.Und).String(t.Name))
	}
	return strings.Join(parts, " / ")
}
