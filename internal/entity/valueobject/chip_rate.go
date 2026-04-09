package valueobject

import "errors"

var ErrInvalidChips = errors.New("invalid chips")
var ErrInvalidMoney = errors.New("invalid money")
var ErrInvalidConversion = errors.New("chips not divisible by rate")

// сколько фишек можно купить за 1 рубль
type ChipRate struct {
	value int64
}

func NewChipRate(v int64) (ChipRate, error) {
	if v <= 0 {
		return ChipRate{}, ErrInvalidChips
	}
	return ChipRate{value: v}, nil
}

func (r ChipRate) ToChips(money Money) (int64, error) {
	if money.Amount() <= 0 {
		return 0, ErrInvalidMoney
	}

	return money.Amount() * r.value, nil
}

func (r ChipRate) ChipsToMoney(chips int64) (Money, error) {
	if err := r.ValidateCashOut(chips); err != nil {
		return Money{}, err
	}

	return NewMoney(chips / r.value)
}

func (r ChipRate) ValidateCashOut(chips int64) error {
	if chips <= 0 {
		return ErrInvalidChips
	}

	if chips%r.value != 0 {
		return ErrInvalidConversion
	}

	return nil
}

func (r ChipRate) Value() int64 {
	return r.value
}
