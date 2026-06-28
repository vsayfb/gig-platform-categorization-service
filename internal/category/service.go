package category

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gosimple/slug"
	pgvector "github.com/pgvector/pgvector-go"
	"github.com/vsayfb/gig-platform-categorization-service/internal/config"
	"github.com/vsayfb/gig-platform-categorization-service/internal/prompter"
)

const similarityThreshold = 0.75

type AIResponse struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type Service struct {
	repo            *Repository
	embeddingClient EmbeddingClient
	cfg             *config.Config
}

type EmbeddingClient interface {
	Embed(ctx context.Context, text string) (pgvector.Vector, error)
}

func NewService(repo *Repository, embeddingClient EmbeddingClient, cfg *config.Config) *Service {
	return &Service{repo: repo, embeddingClient: embeddingClient, cfg: cfg}
}

// ResolveForGig is the main entry point:
// 1. Fetch gig title + description
// 2. Extract profession via AI
// 3. Match or create category via embeddings
// 4. Link gig to category
func (s *Service) ResolveForGig(ctx context.Context, gigID uuid.UUID) (*Category, error) {
	// 1. Fetch gig
	title, description, err := s.repo.GetGig(ctx, gigID)

	if err != nil {
		return nil, fmt.Errorf("fetch gig: %w", err)
	}

	// 2. Extract profession via AI (Groq/Gemini)
	extracted, err := s.ExtractProfession(ctx, title, description)

	if err != nil {
		return nil, fmt.Errorf("extract profession: %w", err)
	}

	log.Printf("extracted profession: %s (%s)", extracted.Name, extracted.Slug)

	// 3. Generate embedding for extracted profession
	embedding, err := s.embeddingClient.Embed(ctx, extracted.Name)

	if err != nil {
		return nil, fmt.Errorf("generate embedding: %w", err)
	}

	// 4. Find similar existing category
	existing, err := s.repo.FindSimilar(ctx, embedding, similarityThreshold)

	if err != nil {
		return nil, fmt.Errorf("find similar category: %w", err)
	}

	var cat *Category

	if existing != nil {
		log.Printf("matched existing category: %s", existing.Slug)

		cat = existing
	} else {
		// 5. Create new category

		log.Printf("creating new category: %s", extracted.Slug)

		cat, err = s.repo.Create(ctx, extracted.Name, extracted.Slug, embedding)

		if err != nil {
			return nil, fmt.Errorf("create category: %w", err)
		}
	}

	// 6. Link gig to category
	if err := s.repo.LinkGig(ctx, gigID, cat.ID); err != nil {
		return nil, fmt.Errorf("link gig to category: %w", err)
	}

	return cat, nil
}

// extractProfession calls Groq API to extract profession name.
func (s *Service) ExtractProfession(ctx context.Context, title, description string) (*AIResponse, error) {
	prompt := prompter.BuildProfessionPrompt(title, description)

	body, err := json.Marshal(map[string]any{
		"model": s.cfg.AI_MODEL,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.cfg.AI_API_ENDPOINT,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.cfg.AI_API_KEY)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("groq returned %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned")
	}

	var extracted AIResponse

	if err := json.Unmarshal([]byte(result.Choices[0].Message.Content), &extracted); err != nil {
		return nil, fmt.Errorf("parse ai response: %w", err)
	}

	if extracted.Name == "" {
		return nil, fmt.Errorf("ai could not determine profession")
	}

	extracted.Slug = slug.Make(extracted.Name)

	return &extracted, nil
}

// keep math import happy
var _ = math.Sqrt
