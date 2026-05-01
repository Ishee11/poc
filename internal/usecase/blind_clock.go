package usecase

import (
	"context"
	"time"

	"github.com/ishee11/poc/internal/entity"
)

type BlindClockLevelInput struct {
	SmallBlind      int64 `json:"small_blind"`
	BigBlind        int64 `json:"big_blind"`
	DurationMinutes int64 `json:"duration_minutes"`
}

type BlindClockLevelDTO struct {
	Level           int   `json:"level"`
	SmallBlind      int64 `json:"small_blind"`
	BigBlind        int64 `json:"big_blind"`
	DurationMinutes int64 `json:"duration_minutes"`
}

type BlindClockBlindsDTO struct {
	SmallBlind int64 `json:"small_blind"`
	BigBlind   int64 `json:"big_blind"`
}

type BlindClockResponse struct {
	ID                entity.BlindClockID     `json:"id"`
	Status            entity.BlindClockStatus `json:"status"`
	Levels            []BlindClockLevelDTO    `json:"levels"`
	CurrentLevelIndex int                     `json:"current_level_index"`
	CurrentLevel      int                     `json:"current_level"`
	RemainingSeconds  int64                   `json:"remaining_seconds"`
	TotalLevels       int                     `json:"total_levels"`
	UpcomingLevels    int                     `json:"upcoming_levels"`
	CurrentBlinds     *BlindClockBlindsDTO    `json:"current_blinds,omitempty"`
	NextBlinds        *BlindClockBlindsDTO    `json:"next_blinds,omitempty"`
	SyncedAt          string                  `json:"synced_at"`
}

type BlindClockService struct {
	repo      BlindClockRepository
	txManager TxManager
	idGen     BlindClockIDGenerator
}

func NewBlindClockService(
	repo BlindClockRepository,
	txManager TxManager,
	idGen BlindClockIDGenerator,
) *BlindClockService {
	return &BlindClockService{
		repo:      repo,
		txManager: txManager,
		idGen:     idGen,
	}
}

func (s *BlindClockService) GetActive(ctx context.Context) (*BlindClockResponse, error) {
	var result *BlindClockResponse

	err := s.txManager.RunInTx(ctx, func(tx Tx) error {
		clock, now, err := s.ensureClock(tx, false)
		if err != nil {
			return err
		}
		result = buildBlindClockResponse(clock, now)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *BlindClockService) Start(ctx context.Context) (*BlindClockResponse, error) {
	return s.mutate(ctx, func(tx Tx, clock *entity.BlindClock, now time.Time) error {
		return clock.Start(now)
	})
}

func (s *BlindClockService) Pause(ctx context.Context) (*BlindClockResponse, error) {
	return s.mutate(ctx, func(tx Tx, clock *entity.BlindClock, now time.Time) error {
		return clock.Pause(now)
	})
}

func (s *BlindClockService) Resume(ctx context.Context) (*BlindClockResponse, error) {
	return s.mutate(ctx, func(tx Tx, clock *entity.BlindClock, now time.Time) error {
		return clock.Resume(now)
	})
}

func (s *BlindClockService) Reset(ctx context.Context) (*BlindClockResponse, error) {
	return s.mutate(ctx, func(tx Tx, clock *entity.BlindClock, now time.Time) error {
		clock.Reset(now)
		return nil
	})
}

func (s *BlindClockService) PreviousLevel(ctx context.Context) (*BlindClockResponse, error) {
	return s.mutate(ctx, func(tx Tx, clock *entity.BlindClock, now time.Time) error {
		return clock.MoveToPreviousLevel(now)
	})
}

func (s *BlindClockService) NextLevel(ctx context.Context) (*BlindClockResponse, error) {
	return s.mutate(ctx, func(tx Tx, clock *entity.BlindClock, now time.Time) error {
		return clock.MoveToNextLevel(now)
	})
}

func (s *BlindClockService) UpdateLevels(ctx context.Context, levelInputs []BlindClockLevelInput) (*BlindClockResponse, error) {
	return s.mutate(ctx, func(tx Tx, clock *entity.BlindClock, now time.Time) error {
		levels := make([]entity.BlindClockLevel, 0, len(levelInputs))
		for idx, level := range levelInputs {
			levels = append(levels, entity.BlindClockLevel{
				LevelIndex:      idx,
				SmallBlind:      level.SmallBlind,
				BigBlind:        level.BigBlind,
				DurationSeconds: level.DurationMinutes * 60,
			})
		}

		if err := validateBlindClockLevelUpdate(clock, levels, now); err != nil {
			return err
		}

		return clock.ReplaceLevels(levels, now)
	})
}

func (s *BlindClockService) mutate(
	ctx context.Context,
	fn func(tx Tx, clock *entity.BlindClock, now time.Time) error,
) (*BlindClockResponse, error) {
	var result *BlindClockResponse

	err := s.txManager.RunInTx(ctx, func(tx Tx) error {
		clock, now, err := s.ensureClock(tx, true)
		if err != nil {
			return err
		}

		if err := fn(tx, clock, now); err != nil {
			return err
		}

		if err := s.repo.Save(tx, clock); err != nil {
			return err
		}

		result = buildBlindClockResponse(clock, now)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *BlindClockService) ensureClock(tx Tx, forUpdate bool) (*entity.BlindClock, time.Time, error) {
	now := time.Now()

	var (
		clock *entity.BlindClock
		err   error
	)

	if forUpdate {
		clock, err = s.repo.FindLatestForUpdate(tx)
	} else {
		clock, err = s.repo.FindLatest(tx)
	}

	if err == entity.ErrBlindClockNotFound {
		clock, err = entity.NewBlindClock(s.idGen.New(), defaultBlindClockLevels(), now)
		if err != nil {
			return nil, time.Time{}, err
		}
		if err := s.repo.Save(tx, clock); err != nil {
			return nil, time.Time{}, err
		}
		return clock, now, nil
	}
	if err != nil {
		return nil, time.Time{}, err
	}

	if clock.Sync(now) {
		if err := s.repo.Save(tx, clock); err != nil {
			return nil, time.Time{}, err
		}
	}

	return clock, now, nil
}

func defaultBlindClockLevels() []entity.BlindClockLevel {
	input := []BlindClockLevelInput{
		{SmallBlind: 10, BigBlind: 20, DurationMinutes: 30},
		{SmallBlind: 20, BigBlind: 40, DurationMinutes: 30},
		{SmallBlind: 50, BigBlind: 100, DurationMinutes: 30},
		{SmallBlind: 100, BigBlind: 200, DurationMinutes: 30},
		{SmallBlind: 250, BigBlind: 500, DurationMinutes: 30},
	}

	levels := make([]entity.BlindClockLevel, 0, len(input))
	for idx, level := range input {
		levels = append(levels, entity.BlindClockLevel{
			LevelIndex:      idx,
			SmallBlind:      level.SmallBlind,
			BigBlind:        level.BigBlind,
			DurationSeconds: level.DurationMinutes * 60,
		})
	}

	return levels
}

func validateBlindClockLevelUpdate(clock *entity.BlindClock, next []entity.BlindClockLevel, now time.Time) error {
	if clock.Status() == entity.BlindClockStatusRunning {
		return entity.ErrBlindClockLevelsLocked
	}
	if clock.Status() == entity.BlindClockStatusIdle {
		return nil
	}

	snapshot := clock.Snapshot(now)
	currentIndex := snapshot.CurrentLevelIndex
	existing := clock.Levels()

	if currentIndex < 0 || len(next) <= currentIndex || len(existing) <= currentIndex {
		return entity.ErrBlindClockLevelsLocked
	}

	for idx := 0; idx <= currentIndex; idx++ {
		if existing[idx].SmallBlind != next[idx].SmallBlind ||
			existing[idx].BigBlind != next[idx].BigBlind ||
			existing[idx].DurationSeconds != next[idx].DurationSeconds {
			return entity.ErrBlindClockLevelsLocked
		}
	}

	return nil
}

func buildBlindClockResponse(clock *entity.BlindClock, now time.Time) *BlindClockResponse {
	snapshot := clock.Snapshot(now)
	levels := clock.Levels()

	resp := &BlindClockResponse{
		ID:                clock.ID(),
		Status:            snapshot.Status,
		Levels:            make([]BlindClockLevelDTO, 0, len(levels)),
		CurrentLevelIndex: snapshot.CurrentLevelIndex,
		CurrentLevel:      snapshot.CurrentLevelIndex + 1,
		RemainingSeconds:  snapshot.RemainingSeconds,
		TotalLevels:       len(levels),
		UpcomingLevels:    max(len(levels)-snapshot.CurrentLevelIndex-1, 0),
		SyncedAt:          now.Format(time.RFC3339),
	}

	if snapshot.CurrentLevelIndex < 0 {
		resp.CurrentLevel = 0
	}

	for _, level := range levels {
		resp.Levels = append(resp.Levels, BlindClockLevelDTO{
			Level:           level.LevelIndex + 1,
			SmallBlind:      level.SmallBlind,
			BigBlind:        level.BigBlind,
			DurationMinutes: level.DurationSeconds / 60,
		})
	}

	if snapshot.CurrentLevelIndex >= 0 && snapshot.CurrentLevelIndex < len(levels) {
		level := levels[snapshot.CurrentLevelIndex]
		resp.CurrentBlinds = &BlindClockBlindsDTO{
			SmallBlind: level.SmallBlind,
			BigBlind:   level.BigBlind,
		}
	}

	nextIndex := snapshot.CurrentLevelIndex + 1
	if nextIndex >= 0 && nextIndex < len(levels) {
		level := levels[nextIndex]
		resp.NextBlinds = &BlindClockBlindsDTO{
			SmallBlind: level.SmallBlind,
			BigBlind:   level.BigBlind,
		}
	}

	return resp
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
