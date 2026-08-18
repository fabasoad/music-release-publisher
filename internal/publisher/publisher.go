package publisher

import "context"

type MusicRelease struct {
	Artist   string `json:"artist"`
	Title    string `json:"title"`
	Album    string `json:"album"`
	Type     string `json:"type"`
	Genre    string `json:"genre"`
	Country  string `json:"country,omitempty"`
	Date     string `json:"date,omitempty"`
	CoverURL string `json:"cover_url,omitempty"`
}

type Publisher interface {
	Name() string
	Publish(ctx context.Context, releases []MusicRelease) error
}

type ReleaseProvider interface {
	FetchReleases(ctx context.Context) ([]MusicRelease, error)
}
