package valueobject

import er "github.com/ishee11/poc/internal/entity/errors"

type ChipRate struct {
	value int64
}

func NewChipRate(v int64) ChipRate {
	if v <= 0 {
		panic("invalid chip rate")
	}
	return ChipRate{value: v}
}

func (r ChipRate) ToMoney(chips int64) (Money, error) {
	if chips <= 0 {
		return Money{}, er.ErrInvalidChips
	}
	return NewMoney(r.value * chips)
}
