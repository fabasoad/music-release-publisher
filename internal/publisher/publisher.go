package publisher

import "context"

type MusicRelease struct {
	Artist   string `json:"artist"`
	Title    string `json:"title"`
	Album    string `json:"album"`
	Type     string `json:"type"` // "Album", "EP", or "Single"
	Genre    string `json:"genre"`
	CoverURL string `json:"cover_url,omitempty"`
}

type Publisher interface {
	Name() string
	Publish(ctx context.Context, releases []MusicRelease) error
}

type ReleaseProvider interface {
	FetchReleases(ctx context.Context) ([]MusicRelease, error)
}
