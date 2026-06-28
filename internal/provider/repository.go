package provider

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Provider struct {
	ID       uuid.UUID
	FCMToken string
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindByGig(ctx context.Context, gigID, categoryID uuid.UUID) ([]Provider, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT p.id, p.fcm_token
		FROM providers p
		LEFT JOIN provider_categories pc ON pc.provider_id = p.id
		JOIN gig_locations gl ON gl.gig_id = $1
		WHERE (
			pc.category_id = $2
		)
		AND p.fcm_token IS NOT NULL
		AND ST_DWithin(
			p.location::geography,
			gl.location::geography,
			p.radius_km * 1000
		)
	`, gigID, categoryID)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var providers []Provider

	for rows.Next() {
		var p Provider

		if err := rows.Scan(&p.ID, &p.FCMToken); err != nil {
			return nil, err
		}
		providers = append(providers, p)
	}

	return providers, nil
}
