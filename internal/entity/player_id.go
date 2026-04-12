package entity

type PlayerID string

type Player struct {
    id   PlayerID
    name string
}

func NewPlayer(id PlayerID, name string) (*Player, error) {
    if id == "" {
        return nil, ErrInvalidPlayerID
    }

    if name == "" {
        return nil, ErrInvalidPlayerName
    }

    return &Player{
        id:   id,
        name: name,
    }, nil
}

func (p *Player) ID() PlayerID {
    return p.id
}

func (p *Player) Name() string {
    return p.name
}
