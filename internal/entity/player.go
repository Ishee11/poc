package entity

type SessionPlayer struct {
	playerID string

	totalChipsBought int64
	totalMoneySpent  int64

	totalChipsCashedOut int64
	totalMoneyCashedOut int64
}
