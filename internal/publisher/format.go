package publisher

import "fmt"

func formatMessage(r MusicRelease) string {
	return fmt.Sprintf("🎵 New %s release!\n\n%s — %s\nGenre: %s", r.Type, r.Artist, r.Title, r.Genre)
}
