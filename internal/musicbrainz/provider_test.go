package musicbrainz

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func noSleep(_ time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	ch <- time.Time{}
	return ch
}

func newTestProvider(srv *httptest.Server) *Provider {
	return &Provider{
		client:  srv.Client(),
		baseURL: srv.URL,
		sleep:   noSleep,
	}
}

func mbReleaseFull(id, status, country string, credits []mbArtistCredit, tags []struct {
	Count int    `json:"count"`
	Name  string `json:"name"`
}) mbRelease {
	r := mbRelease{
		ID:           id,
		Title:        "Test Album",
		Date:         time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02"),
		Status:       status,
		ArtistCredit: credits,
		Tags:         tags,
	}
	r.Country = country
	r.ReleaseGroup.PrimaryType = "Album"
	return r
}

func officialRelease(id string) mbRelease {
	return mbReleaseFull(id, "Official", "US",
		[]mbArtistCredit{{Artist: mbArtist{Name: "Artist"}}},
		[]struct {
			Count int    `json:"count"`
			Name  string `json:"name"`
		}{{Count: 1, Name: "rock"}},
	)
}

func serveJSON(t *testing.T, payload any) *httptest.Server {
	t.Helper()
	data, err := json.Marshal(payload)
	require.NoError(t, err)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}))
}

// ---------------------------------------------------------------------------
// NewProvider
// ---------------------------------------------------------------------------

func TestNewProvider(t *testing.T) {
	p := NewProvider()
	assert.NotNil(t, p.client)
	assert.Equal(t, 30*time.Second, p.client.Timeout)
	assert.Equal(t, baseURL, p.baseURL)
	assert.NotNil(t, p.sleep)
}

// ---------------------------------------------------------------------------
// joinArtists
// ---------------------------------------------------------------------------

func TestJoinArtists_SingleArtist(t *testing.T) {
	credits := []mbArtistCredit{{Artist: mbArtist{Name: "Metallica"}}}
	assert.Equal(t, "Metallica", joinArtists(credits))
}

func TestJoinArtists_MultipleArtists(t *testing.T) {
	credits := []mbArtistCredit{
		{Artist: mbArtist{Name: "Artist A"}},
		{Artist: mbArtist{Name: "Artist B"}},
	}
	assert.Equal(t, "Artist A & Artist B", joinArtists(credits))
}

func TestJoinArtists_FallsBackToCreditName(t *testing.T) {
	credits := []mbArtistCredit{
		{Name: "Fallback", Artist: mbArtist{Name: ""}},
	}
	assert.Equal(t, "Fallback", joinArtists(credits))
}

func TestJoinArtists_SkipsEmptyNames(t *testing.T) {
	credits := []mbArtistCredit{
		{Name: "", Artist: mbArtist{Name: ""}},
		{Artist: mbArtist{Name: "Real"}},
	}
	assert.Equal(t, "Real", joinArtists(credits))
}

func TestJoinArtists_Empty(t *testing.T) {
	assert.Equal(t, "", joinArtists(nil))
}

// ---------------------------------------------------------------------------
// joinGenres
// ---------------------------------------------------------------------------

func makeTags(names ...string) []struct {
	Count int    `json:"count"`
	Name  string `json:"name"`
} {
	tags := make([]struct {
		Count int    `json:"count"`
		Name  string `json:"name"`
	}, len(names))
	for i, n := range names {
		tags[i] = struct {
			Count int    `json:"count"`
			Name  string `json:"name"`
		}{Count: 1, Name: n}
	}
	return tags
}

func TestJoinGenres_Empty(t *testing.T) {
	assert.Equal(t, "", joinGenres(makeTags()))
}

func TestJoinGenres_Single(t *testing.T) {
	assert.Equal(t, "Rock", joinGenres(makeTags("rock")))
}

func TestJoinGenres_TitleCases(t *testing.T) {
	assert.Equal(t, "Heavy Metal", joinGenres(makeTags("heavy metal")))
}

func TestJoinGenres_MultipleJoined(t *testing.T) {
	got := joinGenres(makeTags("rock", "metal"))
	assert.Equal(t, "Rock / Metal", got)
}

func TestJoinGenres_CapAt5(t *testing.T) {
	tags := makeTags("a", "b", "c", "d", "e", "f", "g")
	got := joinGenres(tags)
	parts := strings.Split(got, " / ")
	assert.Len(t, parts, 5)
}

func TestJoinGenres_ExactlyFive(t *testing.T) {
	tags := makeTags("a", "b", "c", "d", "e")
	got := joinGenres(tags)
	parts := strings.Split(got, " / ")
	assert.Len(t, parts, 5)
}

// ---------------------------------------------------------------------------
// FetchReleases – happy path
// ---------------------------------------------------------------------------

func TestFetchReleases_Success(t *testing.T) {
	srv := serveJSON(t, mbResponse{Releases: []mbRelease{officialRelease("abc-123")}})
	defer srv.Close()

	p := newTestProvider(srv)
	releases, err := p.FetchReleases(context.Background())

	require.NoError(t, err)
	require.Len(t, releases, 1)
	assert.Equal(t, "Artist", releases[0].Artist)
	assert.Equal(t, "Test Album", releases[0].Title)
	assert.Equal(t, "Test Album", releases[0].Album)
	assert.Equal(t, "Album", releases[0].Type)
	assert.Equal(t, "US", releases[0].Country)
	assert.Equal(t, "https://coverartarchive.org/release/abc-123/front", releases[0].CoverURL)
}

func TestFetchReleases_Empty(t *testing.T) {
	srv := serveJSON(t, mbResponse{Releases: []mbRelease{}})
	defer srv.Close()

	p := newTestProvider(srv)
	releases, err := p.FetchReleases(context.Background())

	require.NoError(t, err)
	assert.Nil(t, releases)
}

func TestFetchReleases_MultipleReleases(t *testing.T) {
	r1 := officialRelease("id-1")
	r1.ArtistCredit[0].Artist.Name = "Alice"
	r2 := officialRelease("id-2")
	r2.ArtistCredit[0].Artist.Name = "Bob"

	srv := serveJSON(t, mbResponse{Releases: []mbRelease{r1, r2}})
	defer srv.Close()

	p := newTestProvider(srv)
	releases, err := p.FetchReleases(context.Background())

	require.NoError(t, err)
	require.Len(t, releases, 2)
	assert.Equal(t, "Alice", releases[0].Artist)
	assert.Equal(t, "Bob", releases[1].Artist)
}

// ---------------------------------------------------------------------------
// FetchReleases – filtering
// ---------------------------------------------------------------------------

func TestFetchReleases_SkipsNoArtistCredit(t *testing.T) {
	r := officialRelease("x")
	r.ArtistCredit = nil

	srv := serveJSON(t, mbResponse{Releases: []mbRelease{r}})
	defer srv.Close()

	releases, err := newTestProvider(srv).FetchReleases(context.Background())
	require.NoError(t, err)
	assert.Empty(t, releases)
}

func TestFetchReleases_PopulatesCountryFromRU(t *testing.T) {
	r := mbReleaseFull("id", "Official", "RU",
		[]mbArtistCredit{{Artist: mbArtist{Name: "Artist"}}}, makeTags("rock"))

	srv := serveJSON(t, mbResponse{Releases: []mbRelease{r}})
	defer srv.Close()

	releases, err := newTestProvider(srv).FetchReleases(context.Background())
	require.NoError(t, err)
	require.Len(t, releases, 1)
	assert.Equal(t, "RU", releases[0].Country)
}

func TestFetchReleases_SkipsNonOfficialStatus(t *testing.T) {
	r := mbReleaseFull("id", "Bootleg", "US",
		[]mbArtistCredit{{Artist: mbArtist{Name: "Artist"}}}, makeTags("rock"))

	srv := serveJSON(t, mbResponse{Releases: []mbRelease{r}})
	defer srv.Close()

	releases, err := newTestProvider(srv).FetchReleases(context.Background())
	require.NoError(t, err)
	assert.Empty(t, releases)
}

func TestFetchReleases_PassesEnglishNonRu(t *testing.T) {
	r := mbReleaseFull("id", "Official", "DE",
		[]mbArtistCredit{{Artist: mbArtist{Name: "Artist"}}}, makeTags("rock"))

	srv := serveJSON(t, mbResponse{Releases: []mbRelease{r}})
	defer srv.Close()

	releases, err := newTestProvider(srv).FetchReleases(context.Background())
	require.NoError(t, err)
	require.Len(t, releases, 1)
}

// ---------------------------------------------------------------------------
// FetchReleases – request details
// ---------------------------------------------------------------------------

func TestFetchReleases_SetsUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mbResponse{})
	}))
	defer srv.Close()

	_, err := newTestProvider(srv).FetchReleases(context.Background())
	require.NoError(t, err)
	assert.Equal(t, userAgent, gotUA)
}

func TestFetchReleases_QueryContainsFmtJSON(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mbResponse{})
	}))
	defer srv.Close()

	_, err := newTestProvider(srv).FetchReleases(context.Background())
	require.NoError(t, err)
	assert.Contains(t, gotQuery, "fmt=json")
}

func TestFetchReleases_QueryContainsYesterdaysDate(t *testing.T) {
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	var gotRawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mbResponse{})
	}))
	defer srv.Close()

	_, err := newTestProvider(srv).FetchReleases(context.Background())
	require.NoError(t, err)

	decoded, err := url.QueryUnescape(gotRawQuery)
	require.NoError(t, err)
	assert.Contains(t, decoded, "date:"+yesterday)
}

// ---------------------------------------------------------------------------
// FetchReleases – error handling
// ---------------------------------------------------------------------------

func TestFetchReleases_HTTPRequestError(t *testing.T) {
	p := &Provider{
		client:  &http.Client{Timeout: 1 * time.Millisecond},
		baseURL: "http://127.0.0.1:1", // nothing listening
		sleep:   noSleep,
	}
	_, err := p.FetchReleases(context.Background())
	require.Error(t, err)
	assert.ErrorContains(t, err, "musicbrainz: request")
}

func TestFetchReleases_BuildRequestError(t *testing.T) {
	// A URL with a control character causes http.NewRequestWithContext to fail.
	p := &Provider{
		client:  &http.Client{},
		baseURL: "http://invalid host\x00",
		sleep:   noSleep,
	}
	_, err := p.FetchReleases(context.Background())
	require.Error(t, err)
	assert.ErrorContains(t, err, "musicbrainz: build request")
}

func TestFetchReleases_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	_, err := newTestProvider(srv).FetchReleases(context.Background())
	require.Error(t, err)
	assert.ErrorContains(t, err, "musicbrainz: decode response")
}

func TestFetchReleases_Non200Status(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	_, err := newTestProvider(srv).FetchReleases(context.Background())
	require.Error(t, err)
	assert.ErrorContains(t, err, "musicbrainz: unexpected status 401")
}

func TestFetchReleases_ServerErrorAllRetries(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := newTestProvider(srv).FetchReleases(context.Background())
	require.Error(t, err)
	assert.ErrorContains(t, err, "musicbrainz: service unavailable after 3 attempts")
	assert.Equal(t, maxRetries, attempts)
}

func TestFetchReleases_RetrySucceedsOnSecondAttempt(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mbResponse{Releases: []mbRelease{officialRelease("ok")}})
	}))
	defer srv.Close()

	releases, err := newTestProvider(srv).FetchReleases(context.Background())
	require.NoError(t, err)
	assert.Len(t, releases, 1)
	assert.Equal(t, 2, attempts)
}

func TestFetchReleases_ContextCancelledDuringRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	// sleep that blocks until context is cancelled
	blockingSleep := func(_ time.Duration) <-chan time.Time {
		cancel()
		return make(chan time.Time) // never fires
	}

	p := &Provider{
		client:  srv.Client(),
		baseURL: srv.URL,
		sleep:   blockingSleep,
	}

	_, err := p.FetchReleases(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestFetchReleases_MapsGenresFromTags(t *testing.T) {
	r := officialRelease("id")
	r.Tags = makeTags("doom metal", "sludge metal", "post-metal")

	srv := serveJSON(t, mbResponse{Releases: []mbRelease{r}})
	defer srv.Close()

	releases, err := newTestProvider(srv).FetchReleases(context.Background())
	require.NoError(t, err)
	require.Len(t, releases, 1)
	assert.Equal(t, "Doom Metal / Sludge Metal / Post-Metal", releases[0].Genre)
}

func TestFetchReleases_MapsReleaseType(t *testing.T) {
	r := officialRelease("id")
	r.ReleaseGroup.PrimaryType = "EP"

	srv := serveJSON(t, mbResponse{Releases: []mbRelease{r}})
	defer srv.Close()

	releases, err := newTestProvider(srv).FetchReleases(context.Background())
	require.NoError(t, err)
	require.Len(t, releases, 1)
	assert.Equal(t, "EP", releases[0].Type)
}

func TestFetchReleases_MapsDate(t *testing.T) {
	r := officialRelease("id")
	r.Date = "2025-06-15"

	srv := serveJSON(t, mbResponse{Releases: []mbRelease{r}})
	defer srv.Close()

	releases, err := newTestProvider(srv).FetchReleases(context.Background())
	require.NoError(t, err)
	require.Len(t, releases, 1)
	assert.Equal(t, "2025-06-15", releases[0].Date)
}
