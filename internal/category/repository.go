package category

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Category struct {
	ID        uuid.UUID
	Name      string
	Slug      string
	CreatedAt time.Time
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindSimilar(ctx context.Context, embedding []float32, threshold float64) (*Category, error) {
	var storedEmbedding []float32
	var cat Category

	row := r.db.QueryRow(ctx, `
		SELECT id, name, slug, created_at, embedding
		FROM categories
		ORDER BY embedding <=> $1
		LIMIT 1
	`, pgvectorLiteral(embedding))

	if err := row.Scan(&cat.ID, &cat.Name, &cat.Slug, &cat.CreatedAt, &storedEmbedding); err != nil {
		return nil, nil // no categories yet
	}

	if cosineSimilarity(embedding, storedEmbedding) < threshold {
		return nil, nil
	}

	return &cat, nil
}

func (r *Repository) Create(ctx context.Context, name, slug string, embedding []float32) (*Category, error) {
	cat := &Category{
		ID:        uuid.New(),
		Name:      name,
		Slug:      slug,
		CreatedAt: time.Now(),
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO categories (id, name, slug, embedding, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, cat.ID, cat.Name, cat.Slug, pgvectorLiteral(embedding), cat.CreatedAt)
	if err != nil {
		return nil, err
	}

	return cat, nil
}

func cosineSimilarity(a, b []float32) float64 {
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func pgvectorLiteral(v []float32) string {
	b, _ := json.Marshal(v)
	return string(b)
}
