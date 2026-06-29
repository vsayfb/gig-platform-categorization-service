package extractor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/vsayfb/gig-platform-categorization-service/internal/config"
	"github.com/vsayfb/gig-platform-categorization-service/internal/prompter"
)

type ExtractedCategory struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
}

type EmbeddingClient interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type Extractor interface {
	Extract(ctx context.Context, title, description string) (*ExtractedCategory, error)
}

type GroqExtractor struct {
	cfg *config.Config
}

func New(embeddingClient EmbeddingClient, cfg *config.Config) *GroqExtractor {
	return &GroqExtractor{cfg: cfg}
}

type aiResponse struct {
	Name string `json:"name"`
}

func (s *GroqExtractor) Extract(ctx context.Context, title, description string) (*aiResponse, error) {
	prompt := prompter.BuildProfessionPrompt(title, description)

	payload := map[string]any{
		"model": "groq/compound",
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": prompt,
			},
			{
				"role": "user",
				"content": fmt.Sprintf(
					"Gig Title:\n%s\n\nGig Description:\n%s",
					title,
					description,
				),
			},
		},
		"temperature": 0,
	}

	body, err := json.Marshal(payload)

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.groq.com/openai/v1/chat/completions",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+s.cfg.AI_API_KEY)
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)

		log.Printf("Status: %d", resp.StatusCode)
		log.Printf("Body: %s", string(b))

		return nil, fmt.Errorf("groq API error (%d): %s", resp.StatusCode, string(b))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Choices) == 0 {
		return nil, errors.New("no choices returned from Groq")
	}

	var extracted aiResponse
	if err := json.Unmarshal([]byte(result.Choices[0].Message.Content), &extracted); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w\nresponse=%s",
			err,
			result.Choices[0].Message.Content,
		)
	}

	return &extracted, nil
}
