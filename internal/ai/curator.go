package ai

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"

	"music-release-publisher/internal/genres"
	"music-release-publisher/internal/publisher"

	"google.golang.org/genai"
)

//go:embed prompt.tmpl
var promptTmpl string

var prompt = template.Must(template.New("prompt").Parse(promptTmpl))

const model = "gemini-2.5-flash-lite"

type contentGenerator interface {
	GenerateContent(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error)
}

type Curator struct {
	models contentGenerator
	tmpl   *template.Template
}

// newClientFn is the factory for the underlying genai client. Replaced in tests.
var newClientFn = func(ctx context.Context, apiKey string) (*genai.Client, error) {
	return genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
}

func NewCurator(ctx context.Context, apiKey string) (*Curator, error) {
	client, err := newClientFn(ctx, apiKey)
	if err != nil {
		return nil, fmt.Errorf("ai: create client: %w", err)
	}
	return &Curator{models: client.Models, tmpl: prompt}, nil
}

func (c *Curator) FetchReleases(ctx context.Context) ([]publisher.MusicRelease, error) {
	now := time.Now().UTC()
	var buf bytes.Buffer
	if err := c.tmpl.Execute(&buf, map[string]string{
		"Start":  now.AddDate(0, 0, -1).Format("2006/01/02"),
		"End":    now.Format("2006/01/02"),
		"Genres": strings.Join(genres.All, ", "),
	}); err != nil {
		return nil, fmt.Errorf("ai: render prompt: %w", err)
	}

	schema := &genai.Schema{
		Type: genai.TypeArray,
		Items: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"artist":  {Type: genai.TypeString},
				"title":   {Type: genai.TypeString},
				"album":   {Type: genai.TypeString},
				"type":    {Type: genai.TypeString, Enum: []string{"Album", "EP", "Single", "Broadcast", "Other", "Compilation", "Soundtrack", "Spokenword", "Interview", "Audiobook", "Audio drama", "Live", "Remix", "DJ-mix", "Mixtape/Street"}},
				"genre":   {Type: genai.TypeString},
				"country": {Type: genai.TypeString},
			},
			Required: []string{"artist", "title", "album", "type", "genre", "country"},
		},
	}

	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   schema,
	}

	result, err := c.models.GenerateContent(ctx, model, genai.Text(buf.String()), config)
	if err != nil {
		return nil, fmt.Errorf("ai: generate content: %w", err)
	}

	var releases []publisher.MusicRelease
	if err := json.Unmarshal([]byte(result.Text()), &releases); err != nil {
		return nil, fmt.Errorf("ai: unmarshal releases: %w", err)
	}

	// AI date accuracy cannot be trusted beyond the year, so force a partial
	// date of just the target year. The pipeline will verify the exact date via
	// Discogs before including these releases.
	year := now.AddDate(0, 0, -1).Format("2006")
	for i := range releases {
		releases[i].Date = year
	}

	return releases, nil
}
