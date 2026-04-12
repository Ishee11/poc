package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

type PlayerRepository struct{}

func NewPlayerRepository() *PlayerRepository {
    return &PlayerRepository{}
}

func (r *PlayerRepository) Save(
    tx usecase.Tx,
    player *entity.Player,
) error {

    pgxTx, ok := tx.(pgx.Tx)
    if !ok {
        return errors.New("invalid tx type")
    }

    _, err := pgxTx.Exec(context.Background(), `
        INSERT INTO players (id, name)
        VALUES ($1, $2)
        ON CONFLICT (id) DO NOTHING
    `,
        player.ID(),
        player.Name(),
    )

    return err
}

func (r *PlayerRepository) GetByID(
    tx usecase.Tx,
    id entity.PlayerID,
) (*entity.Player, error) {

    pgxTx, ok := tx.(pgx.Tx)
    if !ok {
        return nil, errors.New("invalid tx type")
    }

    row := pgxTx.QueryRow(context.Background(), `
        SELECT id, name
        FROM players
        WHERE id = $1
    `, id)

    var (
        playerID string
        name     string
    )

    if err := row.Scan(&playerID, &name); err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, entity.ErrPlayerNotFound
        }
        return nil, err
    }

    return entity.NewPlayer(
        entity.PlayerID(playerID),
        name,
    )
}

func (r *PlayerRepository) List(
    tx usecase.Tx,
) ([]*entity.Player, error) {

    pgxTx, ok := tx.(pgx.Tx)
    if !ok {
        return nil, errors.New("invalid tx type")
    }

    rows, err := pgxTx.Query(context.Background(), `
        SELECT id, name
        FROM players
        ORDER BY name
    `)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var result []*entity.Player

    for rows.Next() {
        var id string
        var name string

        if err := rows.Scan(&id, &name); err != nil {
            return nil, err
        }

        p, err := entity.NewPlayer(entity.PlayerID(id), name)
        if err != nil {
            return nil, err
        }

        result = append(result, p)
    }

    return result, nil
}
