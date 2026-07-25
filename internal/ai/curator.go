package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"music-release-publisher/internal/genres"
	"music-release-publisher/internal/publisher"

	"google.golang.org/genai"
)

const model = "gemini-2.5-flash-lite"

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
	now := time.Now().UTC()
	start := now.AddDate(0, 0, -1).Format("2006/01/02")
	end := now.Format("2006/01/02")
	prompt := fmt.Sprintf(
		"List notable new music releases between %s 00:00 UTC and %s 00:00 UTC"+
			" including but not limited to the following genres: %v. Return only real,"+
			" verifiable releases. For each release include the artist name,"+
			" release title, release type (Album, EP, or Single), album title"+
			" and release genre. If no new music releases were released in that time range"+
			" — do not return older releases.",
		start,
		end,
		strings.Join(genres.All, ", "),
	)

	schema := &genai.Schema{
		Type: genai.TypeArray,
		Items: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"artist": {Type: genai.TypeString},
				"title":  {Type: genai.TypeString},
				"album":  {Type: genai.TypeString},
				"type":   {Type: genai.TypeString, Enum: []string{"Album", "EP", "Single"}},
				"genre":  {Type: genai.TypeString},
			},
			Required: []string{"artist", "title", "album", "type", "genre"},
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
