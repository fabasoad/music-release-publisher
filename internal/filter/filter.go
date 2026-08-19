package filter

import (
	"music-release-publisher/internal/publisher"
	"strings"
)

func ByCountry(releases []publisher.MusicRelease, excluded ...string) []publisher.MusicRelease {
	out := releases[:0:0]
	for _, r := range releases {
		drop := false
		for _, c := range excluded {
			if strings.EqualFold(r.Country, c) {
				drop = true
				break
			}
		}
		if !drop {
			out = append(out, r)
		}
	}
	return out
}

func ByDate(releases []publisher.MusicRelease, targetDate string) []publisher.MusicRelease {
	out := releases[:0:0]
	for _, r := range releases {
		if r.Date == targetDate {
			out = append(out, r)
		}
	}
	return out
}

func Dedup(releases []publisher.MusicRelease) []publisher.MusicRelease {
	type entry struct {
		r     publisher.MusicRelease
		count int
	}
	best := make(map[string]entry, len(releases))
	for _, r := range releases {
		key := releaseKey(r)
		c := countNonEmptyFields(r)
		e, ok := best[key]
		if !ok || c > e.count || (c == e.count && preferRelease(r, e.r)) {
			best[key] = entry{r: r, count: c}
		}
	}
	seen := make(map[string]struct{}, len(best))
	out := releases[:0:0]
	for _, r := range releases {
		key := releaseKey(r)
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			out = append(out, best[key].r)
		}
	}
	return out
}

func releaseKey(r publisher.MusicRelease) string {
	return strings.ToLower(strings.TrimSpace(r.Artist)) + "\x00" + strings.ToLower(strings.TrimSpace(r.Title))
}

func countNonEmptyFields(r publisher.MusicRelease) int {
	count := 0
	for _, v := range []string{r.Artist, r.Title, r.Date, r.Genre, r.CoverURL, r.Type, r.Country, r.Album} {
		if v != "" {
			count++
		}
	}
	return count
}

func preferRelease(a, b publisher.MusicRelease) bool {
	pairs := [][2]string{
		{a.Artist, b.Artist},
		{a.Title, b.Title},
		{a.Date, b.Date},
		{a.Genre, b.Genre},
		{a.CoverURL, b.CoverURL},
		{a.Type, b.Type},
		{a.Country, b.Country},
		{a.Album, b.Album},
	}
	for _, p := range pairs {
		if p[0] != "" && p[1] == "" {
			return true
		}
		if p[0] == "" && p[1] != "" {
			return false
		}
	}
	return false
}
