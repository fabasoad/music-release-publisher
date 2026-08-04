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
	ID             string `json:"id"`
	Score          int    `json:"score"`
	Count          int    `json:"count"`
	StatusID       string `json:"status-id"`
	Status         string `json:"status"`
	PackagingID    string `json:"packaging-id"`
	Packaging      string `json:"packaging"`
	ArtistCreditID string `json:"artist-credit-id"`
	Title          string `json:"title"`
	Date           string `json:"date"`
	Country        string `json:"country"`
	Barcode        string `json:"barcode"`
	TrackCount     int    `json:"track-count"`
	ArtistCredit   []struct {
		Name   string `json:"name"`
		Artist struct {
			ID             string `json:"id"`
			Name           string `json:"name"`
			SortName       string `json:"sort-name"`
			Disambiguation string `json:"disambiguation"`
		} `json:"artist"`
	} `json:"artist-credit"`
	ReleaseGroup struct {
		ID            string `json:"id"`
		TypeID        string `json:"type-id"`
		PrimaryTypeID string `json:"primary-type-id"`
		Title         string `json:"title"`
		PrimaryType   string `json:"primary-type"`
	} `json:"release-group"`
	TextRepresentation struct {
		Language string `json:"language"`
		Script   string `json:"script"`
	} `json:"text-representation"`
	ReleaseEvents []struct {
		Date string `json:"date"`
		Area struct {
			ID            string   `json:"id"`
			Name          string   `json:"name"`
			SortName      string   `json:"sort-name"`
			Iso31661Codes []string `json:"iso-3166-1-codes"`
		} `json:"area"`
	} `json:"release-events"`
	LabelInfo []struct {
		CatalogNumber string `json:"catalog-number"`
		Label         struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"label"`
	} `json:"label-info"`
	Media []struct {
		ID         string `json:"id"`
		Format     string `json:"format"`
		DiscCount  int    `json:"disc-count"`
		TrackCount int    `json:"track-count"`
	} `json:"media"`
	Tags []struct {
		Count int    `json:"count"`
		Name  string `json:"name"`
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

const maxRetries = 3

func (p *Provider) FetchReleases(ctx context.Context) ([]publisher.MusicRelease, error) {
	date := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")

	var tagParts []string
	for _, g := range genres.All {
		tagParts = append(tagParts, fmt.Sprintf("tag:%q", g))
	}
	query := fmt.Sprintf("date:%s AND (%s)", date, strings.Join(tagParts, " OR "))

	reqURL := fmt.Sprintf("%s/release?query=%s&fmt=json", baseURL, url.QueryEscape(query))

	var resp *http.Response
	for attempt := range maxRetries {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, fmt.Errorf("musicbrainz: build request: %w", err)
		}
		req.Header.Set("User-Agent", userAgent)

		resp, err = p.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("musicbrainz: request: %w", err)
		}

		if resp.StatusCode < 500 {
			break
		}

		_ = resp.Body.Close()
		resp = nil

		if attempt < maxRetries-1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(1<<uint(attempt)) * time.Second):
			}
		}
	}

	if resp == nil {
		return nil, fmt.Errorf("musicbrainz: service unavailable after %d attempts", maxRetries)
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
		// skip releases with no credited artist, wrong release date, Russian-language releases, releases from Russia, or non-official releases
		if len(r.ArtistCredit) == 0 || r.Date != date || r.TextRepresentation.Language == "rus" || r.Country == "RU" || r.Status != "Official" {
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
	Count int    `json:"count"`
	Name  string `json:"name"`
}) string {
	parts := make([]string, 0, len(tags))
	for _, t := range tags {
		parts = append(parts, cases.Title(language.Und).String(t.Name))
	}
	return strings.Join(parts, " / ")
}
