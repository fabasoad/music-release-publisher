package ai

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"text/template"
	"time"

	"music-release-publisher/internal/publisher"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/genai"
)

type mockContentGenerator struct {
	mock.Mock
}

func (m *mockContentGenerator) GenerateContent(
	ctx context.Context,
	model string,
	contents []*genai.Content,
	config *genai.GenerateContentConfig,
) (*genai.GenerateContentResponse, error) {
	args := m.Called(ctx, model, contents, config)
	resp, _ := args.Get(0).(*genai.GenerateContentResponse)
	return resp, args.Error(1)
}

func makeResponse(text string) *genai.GenerateContentResponse {
	return &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Parts: []*genai.Part{{Text: text}},
				},
			},
		},
	}
}

func TestFetchReleases_Success(t *testing.T) {
	body := `[{"artist":"Burial","title":"Antidawn","type":"EP","genre":"electronic","country":"GB"}]`

	gen := new(mockContentGenerator)
	gen.On("GenerateContent", mock.Anything, model, mock.Anything, mock.Anything).
		Return(makeResponse(body), nil)

	c := &Curator{models: gen, tmpl: prompt}
	releases, err := c.FetchReleases(context.Background())

	year := time.Now().UTC().AddDate(0, 0, -1).Format("2006")
	require.NoError(t, err)
	assert.Equal(t, []publisher.MusicRelease{
		{Artist: "Burial", Title: "Antidawn", Type: "EP", Genre: "electronic", Country: "GB", Date: year},
	}, releases)
	gen.AssertExpectations(t)
}

func TestFetchReleases_MultipleReleases(t *testing.T) {
	body := `[
		{"artist":"Portishead","title":"Third","type":"album","genre":"ambient","country":"GB"},
		{"artist":"Radiohead","title":"OK Computer","type":"album","genre":"indie","country":"GB"}
	]`

	gen := new(mockContentGenerator)
	gen.On("GenerateContent", mock.Anything, model, mock.Anything, mock.Anything).
		Return(makeResponse(body), nil)

	c := &Curator{models: gen, tmpl: prompt}
	releases, err := c.FetchReleases(context.Background())

	require.NoError(t, err)
	require.Len(t, releases, 2)
	assert.Equal(t, "Portishead", releases[0].Artist)
	assert.Equal(t, "Radiohead", releases[1].Artist)
	gen.AssertExpectations(t)
}

func TestFetchReleases_EmptyList(t *testing.T) {
	gen := new(mockContentGenerator)
	gen.On("GenerateContent", mock.Anything, model, mock.Anything, mock.Anything).
		Return(makeResponse(`[]`), nil)

	c := &Curator{models: gen, tmpl: prompt}
	releases, err := c.FetchReleases(context.Background())

	require.NoError(t, err)
	assert.Empty(t, releases)
	gen.AssertExpectations(t)
}

func TestFetchReleases_GenerateContentError(t *testing.T) {
	gen := new(mockContentGenerator)
	gen.On("GenerateContent", mock.Anything, model, mock.Anything, mock.Anything).
		Return(nil, errors.New("network error"))

	c := &Curator{models: gen, tmpl: prompt}
	releases, err := c.FetchReleases(context.Background())

	require.Error(t, err)
	assert.ErrorContains(t, err, "ai: generate content")
	assert.ErrorContains(t, err, "network error")
	assert.Nil(t, releases)
	gen.AssertExpectations(t)
}

func TestFetchReleases_InvalidJSON(t *testing.T) {
	gen := new(mockContentGenerator)
	gen.On("GenerateContent", mock.Anything, model, mock.Anything, mock.Anything).
		Return(makeResponse(`not valid json`), nil)

	c := &Curator{models: gen, tmpl: prompt}
	releases, err := c.FetchReleases(context.Background())

	require.Error(t, err)
	assert.ErrorContains(t, err, "ai: unmarshal releases")
	assert.Nil(t, releases)
	gen.AssertExpectations(t)
}

func TestFetchReleases_UsesCorrectModel(t *testing.T) {
	gen := new(mockContentGenerator)
	gen.On("GenerateContent", mock.Anything, "gemini-2.5-flash-lite", mock.Anything, mock.Anything).
		Return(makeResponse(`[]`), nil)

	c := &Curator{models: gen, tmpl: prompt}
	_, err := c.FetchReleases(context.Background())

	require.NoError(t, err)
	gen.AssertExpectations(t)
}

func TestFetchReleases_ConfigHasJSONMimeType(t *testing.T) {
	gen := new(mockContentGenerator)
	gen.On("GenerateContent", mock.Anything, mock.Anything, mock.Anything,
		mock.MatchedBy(func(cfg *genai.GenerateContentConfig) bool {
			return cfg != nil && cfg.ResponseMIMEType == "application/json"
		}),
	).Return(makeResponse(`[]`), nil)

	c := &Curator{models: gen, tmpl: prompt}
	_, err := c.FetchReleases(context.Background())

	require.NoError(t, err)
	gen.AssertExpectations(t)
}

func TestFetchReleases_ConfigHasResponseSchema(t *testing.T) {
	gen := new(mockContentGenerator)
	gen.On("GenerateContent", mock.Anything, mock.Anything, mock.Anything,
		mock.MatchedBy(func(cfg *genai.GenerateContentConfig) bool {
			return cfg != nil && cfg.ResponseSchema != nil &&
				cfg.ResponseSchema.Type == genai.TypeArray
		}),
	).Return(makeResponse(`[]`), nil)

	c := &Curator{models: gen, tmpl: prompt}
	_, err := c.FetchReleases(context.Background())

	require.NoError(t, err)
	gen.AssertExpectations(t)
}

func TestFetchReleases_TemplateExecuteError(t *testing.T) {
	// Template that always fails execution via a bad function call.
	badTmpl := template.Must(template.New("bad").Funcs(template.FuncMap{
		"fail": func() (string, error) { return "", fmt.Errorf("template exec error") },
	}).Parse(`{{fail}}`))

	gen := new(mockContentGenerator)
	c := &Curator{models: gen, tmpl: badTmpl}
	_, err := c.FetchReleases(context.Background())

	require.Error(t, err)
	assert.ErrorContains(t, err, "ai: render prompt")
}

func TestNewCurator_Success(t *testing.T) {
	c, err := NewCurator(context.Background(), "fake-api-key")
	require.NoError(t, err)
	assert.NotNil(t, c)
	assert.NotNil(t, c.tmpl)
}

func TestNewCurator_ClientError(t *testing.T) {
	orig := newClientFn
	newClientFn = func(_ context.Context, _ string) (*genai.Client, error) {
		return nil, fmt.Errorf("injected client error")
	}
	defer func() { newClientFn = orig }()

	c, err := NewCurator(context.Background(), "any-key")
	require.Error(t, err)
	assert.ErrorContains(t, err, "ai: create client")
	assert.ErrorContains(t, err, "injected client error")
	assert.Nil(t, c)
}
