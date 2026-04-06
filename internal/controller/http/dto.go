package http

import "github.com/ishee11/poc/internal/entity"

type SessionDTO struct {
	ID      string      `json:"id"`
	Status  string      `json:"status"`
	Rate    int64       `json:"rate"`
	Players []PlayerDTO `json:"players"`
}

type PlayerDTO struct {
	ID        string `json:"id"`
	Chips     int64  `json:"chips"`
	Spent     int64  `json:"spent"`
	CashedOut int64  `json:"cashed_out"`
	Result    int64  `json:"result"`
}

func mapSession(s *entity.Session) SessionDTO {
	dto := SessionDTO{
		ID:     s.ID(),
		Status: string(s.Status()),
		Rate:   s.Rate().Value(), // 👈 добавим ниже
	}

	for _, p := range s.Players() {
		result, _ := p.TotalMoneyCashedOut().Sub(p.TotalMoneySpent())

		dto.Players = append(dto.Players, PlayerDTO{
			ID:        p.PlayerID(),
			Chips:     p.CurrentChips(),
			Spent:     p.TotalMoneySpent().Amount(),
			CashedOut: p.TotalMoneyCashedOut().Amount(),
			Result:    result.Amount(),
		})
	}

	return dto
}
