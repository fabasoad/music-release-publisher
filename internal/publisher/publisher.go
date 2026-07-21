package publisher

import "context"

type MusicRelease struct {
	Artist string `json:"artist"`
	Title  string `json:"title"`
	Type   string `json:"type"` // "album", "EP", or "single"
	Genre  string `json:"genre"`
}

type Publisher interface {
	Name() string
	Publish(ctx context.Context, release MusicRelease) error
}
