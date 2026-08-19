package filter_test

import (
	"testing"

	"music-release-publisher/internal/filter"
	"music-release-publisher/internal/publisher"

	"github.com/stretchr/testify/assert"
)

func release(country string) publisher.MusicRelease {
	return publisher.MusicRelease{
		Artist:  "Artist",
		Title:   "Album",
		Country: country,
	}
}

func TestByCountry_ExcludesSingleMatch(t *testing.T) {
	releases := []publisher.MusicRelease{release("Russia"), release("US")}
	got := filter.ByCountry(releases, "Russia")
	assert.Equal(t, []publisher.MusicRelease{release("US")}, got)
}

func TestByCountry_ExcludesMultipleValues(t *testing.T) {
	releases := []publisher.MusicRelease{release("Russia"), release("RU"), release("DE")}
	got := filter.ByCountry(releases, "Russia", "RU")
	assert.Equal(t, []publisher.MusicRelease{release("DE")}, got)
}

func TestByCountry_CaseInsensitive(t *testing.T) {
	releases := []publisher.MusicRelease{release("russia"), release("RUSSIA"), release("GB")}
	got := filter.ByCountry(releases, "Russia")
	assert.Equal(t, []publisher.MusicRelease{release("GB")}, got)
}

func TestByCountry_EmptyCountryPassesThrough(t *testing.T) {
	releases := []publisher.MusicRelease{release(""), release("US")}
	got := filter.ByCountry(releases, "Russia", "RU")
	assert.Equal(t, releases, got)
}

func TestByCountry_NoExclusions(t *testing.T) {
	releases := []publisher.MusicRelease{release("US"), release("DE")}
	got := filter.ByCountry(releases)
	assert.Equal(t, releases, got)
}

func TestByCountry_AllExcluded(t *testing.T) {
	releases := []publisher.MusicRelease{release("Russia"), release("RU")}
	got := filter.ByCountry(releases, "Russia", "RU")
	assert.Empty(t, got)
}

func TestByCountry_EmptyInput(t *testing.T) {
	got := filter.ByCountry(nil, "Russia")
	assert.Empty(t, got)
}
