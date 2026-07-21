package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"music-release-publisher/internal/publisher"

	"google.golang.org/genai"
)

const model = "gemini-2.5-flash"

var genres = []string{
	"electronic", "ambient", "post-rock", "indie", "jazz",
}

type contentGenerator interface {
	GenerateContent(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error)
}

type Curator struct {
	models contentGenerator
}

func NewCurator(ctx context.Context, apiKey string) (*Curator, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("ai: create client: %w", err)
	}
	return &Curator{models: client.Models}, nil
}

func (c *Curator) FetchReleases(ctx context.Context) ([]publisher.MusicRelease, error) {
	prompt := fmt.Sprintf(
		"List notable music releases from the past 24 hours across these genres: %v. "+
			"Return only real, verifiable releases. For each release include the artist name, "+
			"release title, type (album, EP, or single), and genre.",
		genres,
	)

	schema := &genai.Schema{
		Type: genai.TypeArray,
		Items: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"artist": {Type: genai.TypeString},
				"title":  {Type: genai.TypeString},
				"type":   {Type: genai.TypeString, Enum: []string{"album", "EP", "single"}},
				"genre":  {Type: genai.TypeString},
			},
			Required: []string{"artist", "title", "type", "genre"},
		},
	}

	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   schema,
	}

	result, err := c.models.GenerateContent(ctx, model, genai.Text(prompt), config)
	if err != nil {
		return nil, fmt.Errorf("ai: generate content: %w", err)
	}

	var releases []publisher.MusicRelease
	if err := json.Unmarshal([]byte(result.Text()), &releases); err != nil {
		return nil, fmt.Errorf("ai: unmarshal releases: %w", err)
	}
	return releases, nil
}
