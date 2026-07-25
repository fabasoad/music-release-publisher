package publisher

import (
	"fmt"
	"strings"
	"time"
)

func formatMessage(releases []MusicRelease) string {
	title := fmt.Sprintf("🔥 New %s Release 🔥", time.Now().AddDate(0, 0, -1).Weekday().String())
	var body []string
	for _, r := range releases {
		body = append(body, fmt.Sprintf("- [%s] %s — %s (%s)", r.Type, r.Artist, r.Title, r.Genre))
	}
	return fmt.Sprintf("%s\n\n%s", title, strings.Join(body, "\n"))
}
