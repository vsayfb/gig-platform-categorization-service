package category

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	pgvector "github.com/pgvector/pgvector-go"
)

type Category struct {
	ID        uuid.UUID
	Name      string
	Slug      string
	Embedding pgvector.Vector
	CreatedAt time.Time
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// FindSimilar returns the most similar category above the threshold, or nil.
func (r *Repository) FindSimilar(ctx context.Context, embedding pgvector.Vector, threshold float64) (*Category, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, name, slug, embedding, created_at
		FROM categories
		ORDER BY embedding <=> $1
		LIMIT 1
	`, embedding)

	var cat Category

	if err := row.Scan(&cat.ID, &cat.Name, &cat.Slug, &cat.Embedding, &cat.CreatedAt); err != nil {
		return nil, nil // no categories yet
	}

	// Calculate cosine similarity
	similarity := cosineSimilarity(embedding.Slice(), cat.Embedding.Slice())

	if similarity < threshold {
		return nil, nil // not similar enough
	}

	return &cat, nil
}

func (r *Repository) Create(ctx context.Context, name, slug string, embedding pgvector.Vector) (*Category, error) {
	cat := &Category{
		ID:        uuid.New(),
		Name:      name,
		Slug:      slug,
		Embedding: embedding,
		CreatedAt: time.Now(),
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO categories (id, name, slug, embedding, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, cat.ID, cat.Name, cat.Slug, cat.Embedding, cat.CreatedAt)
	if err != nil {
		return nil, err
	}

	return cat, nil
}

func (r *Repository) LinkGig(ctx context.Context, gigID, categoryID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO gig_categories (gig_id, category_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, gigID, categoryID)
	return err
}

func (r *Repository) GetGig(ctx context.Context, gigID uuid.UUID) (title, description string, err error) {
	row := r.db.QueryRow(ctx, `
		SELECT title, description_raw FROM gigs WHERE id = $1
	`, gigID)
	err = row.Scan(&title, &description)
	return
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
	return dot / (sqrt(normA) * sqrt(normB))
}

func sqrt(x float64) float64 {
	return math.Sqrt(x)
}
