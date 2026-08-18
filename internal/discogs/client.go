package discogs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const userAgent = "music-release-publisher/1.0.0-beta ( fabasoad@gmail.com )"

type searchResult struct {
	ResourceURL string `json:"resource_url"`
}

type searchResponse struct {
	Results []searchResult `json:"results"`
}

type releaseImage struct {
	Type string `json:"type"`
	URI  string `json:"uri"`
}

type release struct {
	Released string         `json:"released"`
	Images   []releaseImage `json:"images"`
}

// Client is a minimal Discogs API client.
type Client struct {
	http  *http.Client
	token string
}

// NewClient returns a Client authenticated with the given personal access token.
func NewClient(token string) *Client {
	return &Client{
		http:  &http.Client{Timeout: 30 * time.Second},
		token: token,
	}
}

// ReleaseInfo holds the Discogs details for a release.
type ReleaseInfo struct {
	Date     string
	CoverURL string
}

// FetchReleaseInfo searches Discogs for the release with the given title and
// artist and returns its exact release date and primary cover image URL. It
// performs two requests:
//  1. Search for the release to obtain its resource URL.
//  2. Fetch the release detail to read the full "released" date and cover image.
//
// Returns (nil, nil) when no matching release is found.
func (c *Client) FetchReleaseInfo(ctx context.Context, title, artist string) (*ReleaseInfo, error) {
	resourceURL, err := c.searchResourceURL(ctx, title, artist)
	if err != nil {
		return nil, err
	}
	if resourceURL == "" {
		return nil, nil
	}

	r, err := c.fetchRelease(ctx, resourceURL)
	if err != nil {
		return nil, err
	}

	info := &ReleaseInfo{Date: r.Released}
	for _, img := range r.Images {
		if img.Type == "primary" && img.URI != "" {
			info.CoverURL = img.URI
			break
		}
	}
	return info, nil
}

func (c *Client) searchResourceURL(ctx context.Context, title, artist string) (string, error) {
	searchURL := fmt.Sprintf(
		"https://api.discogs.com/database/search?release_title=%s&artist=%s",
		url.QueryEscape(title),
		url.QueryEscape(artist),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("discogs: build search request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("discogs: search request: %w", err)
	}
	defer func(b io.ReadCloser) { _ = b.Close() }(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("discogs: unexpected search status %d", resp.StatusCode)
	}

	var sr searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return "", fmt.Errorf("discogs: decode search response: %w", err)
	}

	if len(sr.Results) == 0 {
		return "", nil
	}
	return sr.Results[0].ResourceURL, nil
}

func (c *Client) fetchRelease(ctx context.Context, resourceURL string) (*release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resourceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("discogs: build release request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("discogs: release request: %w", err)
	}
	defer func(b io.ReadCloser) { _ = b.Close() }(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discogs: unexpected release status %d", resp.StatusCode)
	}

	var r release
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("discogs: decode release response: %w", err)
	}
	return &r, nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Discogs token="+c.token)
	req.Header.Set("User-Agent", userAgent)
}
