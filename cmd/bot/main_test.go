package main

import (
	"testing"

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

func TestFilterByCountry_ExcludesSingleMatch(t *testing.T) {
	releases := []publisher.MusicRelease{release("Russia"), release("US")}
	got := filterByCountry(releases, "Russia")
	assert.Equal(t, []publisher.MusicRelease{release("US")}, got)
}

func TestFilterByCountry_ExcludesMultipleValues(t *testing.T) {
	releases := []publisher.MusicRelease{release("Russia"), release("RU"), release("DE")}
	got := filterByCountry(releases, "Russia", "RU")
	assert.Equal(t, []publisher.MusicRelease{release("DE")}, got)
}

func TestFilterByCountry_CaseInsensitive(t *testing.T) {
	releases := []publisher.MusicRelease{release("russia"), release("RUSSIA"), release("GB")}
	got := filterByCountry(releases, "Russia")
	assert.Equal(t, []publisher.MusicRelease{release("GB")}, got)
}

func TestFilterByCountry_EmptyCountryPassesThrough(t *testing.T) {
	releases := []publisher.MusicRelease{release(""), release("US")}
	got := filterByCountry(releases, "Russia", "RU")
	assert.Equal(t, releases, got)
}

func TestFilterByCountry_NoExclusions(t *testing.T) {
	releases := []publisher.MusicRelease{release("US"), release("DE")}
	got := filterByCountry(releases)
	assert.Equal(t, releases, got)
}

func TestFilterByCountry_AllExcluded(t *testing.T) {
	releases := []publisher.MusicRelease{release("Russia"), release("RU")}
	got := filterByCountry(releases, "Russia", "RU")
	assert.Empty(t, got)
}

func TestFilterByCountry_EmptyInput(t *testing.T) {
	got := filterByCountry(nil, "Russia")
	assert.Empty(t, got)
}