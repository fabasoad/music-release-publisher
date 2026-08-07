package publisher

import (
	"fmt"
	"strings"
	"time"
)

func formatMessageSingle(release MusicRelease) string {
	title := fmt.Sprintf("🔥 New %s Release 🔥", time.Now().AddDate(0, 0, -1).Weekday().String())
	body := fmt.Sprintf("[%s] %s — %s (%s)", release.Type, release.Artist, release.Title, release.Genre)
	return fmt.Sprintf("%s\n\n%s", title, body)
}

func formatMessageMultiple(releases []MusicRelease) string {
	title := fmt.Sprintf("🔥 New %s Releases 🔥", time.Now().AddDate(0, 0, -1).Weekday().String())
	var body []string
	for _, r := range releases {
		body = append(body, fmt.Sprintf("- [%s] %s — %s (%s)", r.Type, r.Artist, r.Title, r.Genre))
	}
	return fmt.Sprintf("%s\n\n%s", title, strings.Join(body, "\n"))
}

func truncateText(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit-3]) + "..."
}
