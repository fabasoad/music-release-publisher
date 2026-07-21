package ai

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/genai"
	"music-release-publisher/internal/publisher"
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
	body := `[{"artist":"Burial","title":"Antidawn","type":"EP","genre":"electronic"}]`

	gen := new(mockContentGenerator)
	gen.On("GenerateContent", mock.Anything, model, mock.Anything, mock.Anything).
		Return(makeResponse(body), nil)

	c := &Curator{models: gen}
	releases, err := c.FetchReleases(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []publisher.MusicRelease{
		{Artist: "Burial", Title: "Antidawn", Type: "EP", Genre: "electronic"},
	}, releases)
	gen.AssertExpectations(t)
}

func TestFetchReleases_MultipleReleases(t *testing.T) {
	body := `[
		{"artist":"Portishead","title":"Third","type":"album","genre":"ambient"},
		{"artist":"Radiohead","title":"OK Computer","type":"album","genre":"indie"}
	]`

	gen := new(mockContentGenerator)
	gen.On("GenerateContent", mock.Anything, model, mock.Anything, mock.Anything).
		Return(makeResponse(body), nil)

	c := &Curator{models: gen}
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

	c := &Curator{models: gen}
	releases, err := c.FetchReleases(context.Background())

	require.NoError(t, err)
	assert.Empty(t, releases)
	gen.AssertExpectations(t)
}

func TestFetchReleases_GenerateContentError(t *testing.T) {
	gen := new(mockContentGenerator)
	gen.On("GenerateContent", mock.Anything, model, mock.Anything, mock.Anything).
		Return(nil, errors.New("network error"))

	c := &Curator{models: gen}
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

	c := &Curator{models: gen}
	releases, err := c.FetchReleases(context.Background())

	require.Error(t, err)
	assert.ErrorContains(t, err, "ai: unmarshal releases")
	assert.Nil(t, releases)
	gen.AssertExpectations(t)
}

func TestFetchReleases_UsesCorrectModel(t *testing.T) {
	gen := new(mockContentGenerator)
	gen.On("GenerateContent", mock.Anything, "gemini-2.5-flash", mock.Anything, mock.Anything).
		Return(makeResponse(`[]`), nil)

	c := &Curator{models: gen}
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

	c := &Curator{models: gen}
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

	c := &Curator{models: gen}
	_, err := c.FetchReleases(context.Background())

	require.NoError(t, err)
	gen.AssertExpectations(t)
}
