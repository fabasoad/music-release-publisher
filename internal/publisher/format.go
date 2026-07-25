package publisher

import (
	"fmt"
)

func formatMessage(r MusicRelease) string {
	title := fmt.Sprintf("🔥 New %s Release 🔥", r.Type)
	body := fmt.Sprintf("%s — %s (%s)", r.Artist, r.Title, r.Genre)
	return fmt.Sprintf("%s\n\n%s", title, body)
}
