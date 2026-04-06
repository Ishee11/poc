package valueobject

type ChipRate struct {
	value int64
}

func NewChipRate(v int64) ChipRate {
	if v <= 0 {
		panic("invalid chip rate")
	}
	return ChipRate{value: v}
}

func (r ChipRate) ToMoney(chips int64) int64 {
	return r.value * chips
}
