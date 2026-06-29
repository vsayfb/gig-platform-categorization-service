package subscriber

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Subscriber struct {
	ID       uuid.UUID
	FCMToken string
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindByCategoryAndLocation(ctx context.Context, categoryID uuid.UUID, lat, lng float64) ([]Subscriber, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT p.id, p.fcm_token
		FROM users p
		LEFT JOIN user_categories pc ON pc.user_id = p.id
		WHERE (
			pc.category_id = $1
		)
		AND p.fcm_token IS NOT NULL
		AND ST_DWithin(
			p.location::geography,
			ST_MakePoint($2, $3)::geography,
			p.radius_km * 1000
		)
	`, categoryID, lng, lat)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subscribers []Subscriber

	for rows.Next() {
		var s Subscriber

		if err := rows.Scan(&s.ID, &s.FCMToken); err != nil {
			return nil, err
		}

		subscribers = append(subscribers, s)
	}

	return subscribers, nil
}
