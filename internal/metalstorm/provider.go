package metalstorm

import (
	"context"
	"fmt"
	"io"
	"music-release-publisher/internal/publisher"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Provider struct {
	client  *http.Client
	baseURL string
}

func NewProvider() *Provider {
	return &Provider{
		client:  &http.Client{Timeout: 30 * time.Second},
		baseURL: "https://metalstorm.net",
	}
}

func (p *Provider) FetchReleases(ctx context.Context) ([]publisher.MusicRelease, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/events/new_releases.php", p.baseURL), nil)
	if err != nil {
		return nil, err
	}

	// Standard browser headers to avoid basic blocks
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/150.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Cookie", "ms_id=6f7f5c29ee2e95f5aea03fd9e522b00a; ms_settings=cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e; cf_clearance=u7bnVa54gA8GHjC.9qE6Z2jAl3jcI5HHbesPzGUrTSs-1787098884-1.2.1.1-8KLGUtTaPR5IgQ0zSb9PP0Rl0cRxd_i4t0rLO1oVD0YJSd4VTPJ.axQdlqSIwLnj_9pT_iu10KAJ7U0Gn8M1ZdsGNq6g7ONPBuWWaaEhDyATPedK0gxFdGsUT5lHPagJpssTUfB8nfz5drDHetXN4AHUWQkrAJTmJt0SOD_JQdfvi0QTfWP2D7sl6Dk3kjj5HBxQVdCjB4TIEbOqVLLrGKvMmBmg.mLit37CPWCx7Ib4psPMQbLcbgbxGP1KIRJahtBDFQ1GanVXIZ8QmKqHDDdQZxRMuaPhBgIxYKlbmj5dDq.onjCo0QAQigbG4xO7QFBZOlu8cwHv_XK0OB9Jc8UPzsGnRgnNG9JYzbHM8LQ")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(b io.ReadCloser) {
		_ = b.Close()
	}(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("metal storm returned HTTP status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var releases []publisher.MusicRelease
	monthYear := time.Now().Format("January-2006")
	doc.Find("div#" + monthYear + ">table.table.table-striped.align-middle tr").Each(func(i int, s *goquery.Selection) {
		// Extract details from table cells
		band := strings.TrimSpace(s.Find("a.ms-link[href^='/bands/band.php?band_id=']").First().Text())
		album := strings.TrimSpace(s.Find("a.ms-link[href^='/bands/album.php?album_id=']").First().Text())
		releaseType := strings.Trim(s.Find("div.col-hide-when-pane-open>span.dark:not([class*=' '])").Text(), " []")

		genre := normalizeGenre(strings.TrimSpace(s.Find("div.col-lg-3.col-hide-when-pane-open").Text()))

		countryRaw, _ := s.Find("a[href*='band.php']").First().Attr("data-content-1")
		country := strings.TrimSpace(countryRaw)
		if idx := strings.LastIndex(country, " "); idx != -1 {
			country = strings.TrimSpace(country[:idx])
		}

		date := normalizeDate(strings.TrimSpace(s.Find("span.dark.d-md-none.me-1").Text()))

		coverPath, _ := s.Find("a[href*='album.php']").First().Attr("data-image-url")
		coverURL := ""
		if coverPath != "" {
			coverURL = p.baseURL + coverPath
		}

		if band != "" && album != "" {
			releases = append(releases, publisher.MusicRelease{
				Artist:   band,
				Title:    album,
				Album:    album,
				Type:     releaseType,
				Genre:    genre,
				Country:  country,
				Date:     date,
				CoverURL: coverURL,
			})
		}
	})

	return releases, nil
}

func normalizeGenre(genre string) string {
	genres := strings.Split(genre, ",")
	if len(genres) == 0 {
		return ""
	}
	var out []string
	for _, g := range genres {
		out = append(out, strings.TrimSpace(g))
	}
	return strings.Join(out, " / ")
}

// normalizeDate converts MetalStorm's "DD.MM" format to "YYYY-MM-DD" using the current UTC year.
// Returns the input unchanged if it doesn't match the expected format.
func normalizeDate(raw string) string {
	t, err := time.Parse("02.01", raw)
	if err != nil {
		return raw
	}
	now := time.Now().UTC()
	return time.Date(now.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Format("2006-01-02")
}
