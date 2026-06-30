package subscriber

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const subscriberRadiusMeters = 150_000 // 150km

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
		SELECT DISTINCT u.id, ua.fcm_token
		FROM users u
		JOIN user_auth ua ON ua.user_id = u.id
		JOIN user_locations ul ON ul.user_id = u.id
		JOIN user_categories uc ON uc.user_id = u.id
		WHERE uc.category_id = $1
		AND ua.fcm_token IS NOT NULL
		AND ST_DWithin(
			ul.location::geography,
			ST_MakePoint($2, $3)::geography,
			$4
		)
	`, categoryID, lng, lat, subscriberRadiusMeters)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	subscribers, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Subscriber])

	if err != nil {
		return nil, err
	}

	return subscribers, nil
}
