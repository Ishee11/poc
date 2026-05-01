package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

type BlindClockRepository struct{}

func NewBlindClockRepository() *BlindClockRepository {
	return &BlindClockRepository{}
}

func (r *BlindClockRepository) FindLatest(tx usecase.Tx) (*entity.BlindClock, error) {
	return r.findLatest(tx, false)
}

func (r *BlindClockRepository) FindLatestForUpdate(tx usecase.Tx) (*entity.BlindClock, error) {
	return r.findLatest(tx, true)
}

func (r *BlindClockRepository) findLatest(tx usecase.Tx, forUpdate bool) (*entity.BlindClock, error) {
	query := `
		SELECT id, status, started_at, paused_at, finished_at, accumulated_pause_seconds, created_at, updated_at
		FROM blind_clocks
		ORDER BY updated_at DESC, created_at DESC
		LIMIT 1
	`
	if forUpdate {
		query += ` FOR UPDATE`
	}

	row := tx.QueryRow(context.Background(), query)

	var (
		id                      string
		status                  entity.BlindClockStatus
		startedAt               *time.Time
		pausedAt                *time.Time
		finishedAt              *time.Time
		accumulatedPauseSeconds int64
		createdAt               time.Time
		updatedAt               time.Time
	)

	if err := row.Scan(
		&id,
		&status,
		&startedAt,
		&pausedAt,
		&finishedAt,
		&accumulatedPauseSeconds,
		&createdAt,
		&updatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrBlindClockNotFound
		}
		return nil, err
	}

	levels, err := r.listLevels(tx, entity.BlindClockID(id))
	if err != nil {
		return nil, err
	}

	return entity.RestoreBlindClock(
		entity.BlindClockID(id),
		status,
		levels,
		startedAt,
		pausedAt,
		finishedAt,
		accumulatedPauseSeconds,
		createdAt,
		updatedAt,
	)
}

func (r *BlindClockRepository) Save(tx usecase.Tx, clock *entity.BlindClock) error {
	_, err := tx.Exec(context.Background(), `
		INSERT INTO blind_clocks (
			id, status, started_at, paused_at, finished_at, accumulated_pause_seconds, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			started_at = EXCLUDED.started_at,
			paused_at = EXCLUDED.paused_at,
			finished_at = EXCLUDED.finished_at,
			accumulated_pause_seconds = EXCLUDED.accumulated_pause_seconds,
			updated_at = EXCLUDED.updated_at
	`,
		clock.ID(),
		clock.Status(),
		clock.StartedAt(),
		clock.PausedAt(),
		clock.FinishedAt(),
		clock.AccumulatedPauseSeconds(),
		clock.CreatedAt(),
		clock.UpdatedAt(),
	)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(context.Background(), `DELETE FROM blind_clock_levels WHERE clock_id = $1`, clock.ID()); err != nil {
		return err
	}

	for _, level := range clock.Levels() {
		if _, err := tx.Exec(context.Background(), `
			INSERT INTO blind_clock_levels (clock_id, level_index, small_blind, big_blind, duration_seconds)
			VALUES ($1, $2, $3, $4, $5)
		`,
			clock.ID(),
			level.LevelIndex,
			level.SmallBlind,
			level.BigBlind,
			level.DurationSeconds,
		); err != nil {
			return err
		}
	}

	return nil
}

func (r *BlindClockRepository) listLevels(tx usecase.Tx, clockID entity.BlindClockID) ([]entity.BlindClockLevel, error) {
	rows, err := tx.Query(context.Background(), `
		SELECT level_index, small_blind, big_blind, duration_seconds
		FROM blind_clock_levels
		WHERE clock_id = $1
		ORDER BY level_index ASC
	`, clockID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var levels []entity.BlindClockLevel
	for rows.Next() {
		var level entity.BlindClockLevel
		if err := rows.Scan(
			&level.LevelIndex,
			&level.SmallBlind,
			&level.BigBlind,
			&level.DurationSeconds,
		); err != nil {
			return nil, err
		}
		levels = append(levels, level)
	}

	return levels, rows.Err()
}
