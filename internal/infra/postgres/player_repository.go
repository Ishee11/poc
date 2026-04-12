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
