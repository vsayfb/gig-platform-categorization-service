package category

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/vsayfb/gig-platform-categorization-service/internal/config"
	"github.com/vsayfb/gig-platform-categorization-service/internal/prompter"
)

const similarityThreshold = 0.85

type EmbeddingClient interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type Service struct {
	repo            *Repository
	embeddingClient EmbeddingClient
	cfg             *config.Config
}

func NewService(repo *Repository, embeddingClient EmbeddingClient, cfg *config.Config) *Service {
	return &Service{repo: repo, embeddingClient: embeddingClient, cfg: cfg}
}

func (s *Service) Resolve(ctx context.Context, title, description string) (*Category, error) {
	// Extract profession via AI
	extracted, err := s.extractProfession(ctx, title, description)
	if err != nil {
		return nil, fmt.Errorf("extract profession: %w", err)
	}

	// Generate embedding
	embedding, err := s.embeddingClient.Embed(ctx, extracted.Name)

	if err != nil {
		return nil, fmt.Errorf("generate embedding: %w", err)
	}

	// Find similar existing category
	existing, err := s.repo.FindSimilar(ctx, embedding, similarityThreshold)

	if err != nil {
		return nil, fmt.Errorf("find similar category: %w", err)
	}

	if existing != nil {
		return existing, nil
	}

	// Create new category
	cat, err := s.repo.Create(ctx, extracted.Name, extracted.Slug, embedding)

	if err != nil {
		return nil, fmt.Errorf("create category: %w", err)
	}

	return cat, nil
}

type aiResponse struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (s *Service) extractProfession(ctx context.Context, title, description string) (*aiResponse, error) {
	prompt := prompter.BuildProfessionPrompt(title, description)

	body, _ := json.Marshal(map[string]any{
		"model": "llama3-8b-8192",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.groq.com/openai/v1/chat/completions",
		strings.NewReader(string(body)),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.AI_API_KEY)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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
		return nil, fmt.Errorf("no response from groq")
	}

	var extracted aiResponse
	if err := json.Unmarshal([]byte(result.Choices[0].Message.Content), &extracted); err != nil {
		return nil, fmt.Errorf("parse ai response: %w", err)
	}

	return &extracted, nil
}
