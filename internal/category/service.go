package category

import (
	"context"
	"fmt"

	"github.com/gosimple/slug"
	"github.com/vsayfb/gig-platform-categorization-service/internal/config"
	"github.com/vsayfb/gig-platform-categorization-service/internal/extractor"
)

const similarityThreshold = 0.85

type EmbeddingClient interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type Service struct {
	repo            *Repository
	embeddingClient EmbeddingClient
	extractor       extractor.Extractor
	cfg             *config.Config
}

func NewService(repo *Repository, embeddingClient EmbeddingClient, extractor extractor.Extractor, cfg *config.Config) *Service {
	return &Service{repo: repo, embeddingClient: embeddingClient, extractor: extractor, cfg: cfg}
}

func (s *Service) Resolve(ctx context.Context, title, description string) (*Category, error) {
	extracted, err := s.extractor.Extract(ctx, title, description)

	if err != nil {
		return nil, fmt.Errorf("extract profession: %w", err)
	}

	embedding, err := s.embeddingClient.Embed(ctx, extracted.Name)

	if err != nil {
		return nil, fmt.Errorf("generate embedding: %w", err)
	}

	existing, err := s.repo.FindSimilar(ctx, embedding, similarityThreshold)

	if err != nil {
		return nil, fmt.Errorf("find similar category: %w", err)
	}

	if existing != nil {
		return existing, nil
	}

	cat, err := s.repo.Create(ctx, extracted.Name, slug.Make(extracted.Name), embedding)

	if err != nil {
		return nil, fmt.Errorf("create category: %w", err)
	}

	return cat, nil
}
