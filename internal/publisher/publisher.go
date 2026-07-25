package publisher

import "context"

type MusicRelease struct {
	Artist string `json:"artist"`
	Title  string `json:"title"`
	Album  string `json:"album"`
	Type   string `json:"type"` // "album", "EP", or "single"
	Genre  string `json:"genre"`
}

type Publisher interface {
	Name() string
	Publish(ctx context.Context, release MusicRelease) error
}

type ReleaseProvider interface {
	FetchReleases(ctx context.Context) ([]MusicRelease, error)
}
