package subscriber

import (
	"context"
	"log/slog"

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

const CATEGORY_QUERY = `
	SELECT DISTINCT u.id, ft.token
	FROM users u
	JOIN fcm_tokens ft ON ft.user_id = u.id
	JOIN user_categories uc ON uc.user_id = u.id
	WHERE uc.category_id = $1
	  AND ft.token IS NOT NULL
`

const LOCATION_QUERY = `
		SELECT DISTINCT u.id, ft.token
		FROM users u
		JOIN fcm_tokens ft ON ft.user_id = u.id
		JOIN user_locations ul ON ul.user_id = u.id
		JOIN user_categories uc ON uc.user_id = u.id
		WHERE uc.category_id = $1
		AND ft.token IS NOT NULL
		AND ST_DWithin(
			ul.location::geography,
			ST_MakePoint($2, $3)::geography,
			$4
		)
`

func (r *Repository) FindByCategory(
	ctx context.Context,
	categoryID uuid.UUID,
) ([]Subscriber, error) {
	rows, err := r.db.Query(ctx, CATEGORY_QUERY, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Subscriber])

	if err != nil {
		slog.ErrorContext(ctx, "subscribers couldn't fetch", "err", err)
		return nil, err
	}

	slog.InfoContext(ctx, "subscribers found", "will be notified", res)

	return res, nil
}

func (r *Repository) FindByCategoryAndLocation(
	ctx context.Context,
	categoryID uuid.UUID,
	lat, lng float64,
) ([]Subscriber, error) {
	rows, err := r.db.Query(
		ctx,
		LOCATION_QUERY,
		categoryID,
		lng,
		lat,
		subscriberRadiusMeters,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Subscriber])

	if err != nil {
		slog.ErrorContext(ctx, "subscribers couldn't fetch", "err", err)
		return nil, err
	}

	slog.InfoContext(ctx, "subscribers found", "will be notified", res)

	return res, nil
}
