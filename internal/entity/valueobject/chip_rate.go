package valueobject

import "errors"

var ErrInvalidChips = errors.New("invalid chips")
var ErrInvalidMoney = errors.New("invalid money")
var ErrInvalidConversion = errors.New("chips not divisible by rate")

type ChipRate struct {
	value int64
}

func NewChipRate(v int64) ChipRate {
	if v <= 0 {
		panic("invalid chip rate")
	}
	return ChipRate{value: v}
}

func (r ChipRate) ToChips(money Money) (int64, error) {
	if money.Amount() <= 0 {
		return 0, ErrInvalidMoney
	}

	return money.Amount() * r.value, nil
}

func (r ChipRate) ChipsToMoney(chips int64) (Money, error) {
	if chips <= 0 {
		return Money{}, ErrInvalidChips
	}

	if chips%r.value != 0 {
		return Money{}, ErrInvalidConversion
	}

	return NewMoney(chips / r.value)
}

func (r ChipRate) Value() int64 {
	return r.value
}
