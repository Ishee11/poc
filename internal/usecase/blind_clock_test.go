package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/ishee11/poc/internal/entity"
)

func TestValidateBlindClockLevelUpdateAllowsFutureLevelWhileRunning(t *testing.T) {
	now := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	clock := newTestBlindClock(t, now)

	if err := clock.Start(now); err != nil {
		t.Fatalf("start clock: %v", err)
	}

	next := clock.Levels()
	next[1].SmallBlind = 30
	next[1].BigBlind = 60

	if err := validateBlindClockLevelUpdate(clock, next, now); err != nil {
		t.Fatalf("validate future level update while running: %v", err)
	}
}

func TestValidateBlindClockLevelUpdateLocksCurrentLevelWhileRunning(t *testing.T) {
	now := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	clock := newTestBlindClock(t, now)

	if err := clock.Start(now); err != nil {
		t.Fatalf("start clock: %v", err)
	}

	next := clock.Levels()
	next[0].SmallBlind = 15

	err := validateBlindClockLevelUpdate(clock, next, now)
	if !errors.Is(err, entity.ErrBlindClockLevelsLocked) {
		t.Fatalf("validate current level update while running = %v, want %v", err, entity.ErrBlindClockLevelsLocked)
	}
}

func TestValidateBlindClockLevelUpdateAllowsCurrentLevelWhilePaused(t *testing.T) {
	now := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	clock := newTestBlindClock(t, now)

	if err := clock.Start(now); err != nil {
		t.Fatalf("start clock: %v", err)
	}
	if err := clock.Pause(now.Add(5 * time.Minute)); err != nil {
		t.Fatalf("pause clock: %v", err)
	}

	next := clock.Levels()
	next[0].SmallBlind = 15
	next[0].BigBlind = 30

	if err := validateBlindClockLevelUpdate(clock, next, now); err != nil {
		t.Fatalf("validate current level update while paused: %v", err)
	}
}

func TestValidateBlindClockLevelUpdateLocksCompletedLevelWhilePaused(t *testing.T) {
	now := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	clock := newTestBlindClock(t, now)

	if err := clock.Start(now); err != nil {
		t.Fatalf("start clock: %v", err)
	}
	if err := clock.MoveToNextLevel(now.Add(5 * time.Minute)); err != nil {
		t.Fatalf("move to next level: %v", err)
	}
	if err := clock.Pause(now.Add(5 * time.Minute)); err != nil {
		t.Fatalf("pause clock: %v", err)
	}

	next := clock.Levels()
	next[0].SmallBlind = 15

	err := validateBlindClockLevelUpdate(clock, next, now.Add(5*time.Minute))
	if !errors.Is(err, entity.ErrBlindClockLevelsLocked) {
		t.Fatalf("validate completed level update while paused = %v, want %v", err, entity.ErrBlindClockLevelsLocked)
	}
}

func TestDefaultBlindClockLevelsUseTournamentStructure(t *testing.T) {
	levels := defaultBlindClockLevels()
	wantBigBlinds := []int64{20, 40, 80, 100, 200, 300, 400, 600, 1000}
	wantDurations := []int64{40, 40, 40, 30, 30, 30, 20, 20, 20}

	if len(levels) != len(wantBigBlinds) {
		t.Fatalf("default levels len = %d, want %d", len(levels), len(wantBigBlinds))
	}

	for idx, level := range levels {
		if level.LevelIndex != idx {
			t.Fatalf("level %d index = %d, want %d", idx, level.LevelIndex, idx)
		}
		if level.BigBlind != wantBigBlinds[idx] {
			t.Fatalf("level %d big blind = %d, want %d", idx, level.BigBlind, wantBigBlinds[idx])
		}
		if level.SmallBlind != wantBigBlinds[idx]/2 {
			t.Fatalf("level %d small blind = %d, want %d", idx, level.SmallBlind, wantBigBlinds[idx]/2)
		}
		if level.DurationSeconds != wantDurations[idx]*60 {
			t.Fatalf("level %d duration = %d, want %d", idx, level.DurationSeconds, wantDurations[idx]*60)
		}
	}
}

func newTestBlindClock(t *testing.T, now time.Time) *entity.BlindClock {
	t.Helper()

	clock, err := entity.NewBlindClock("clock-1", []entity.BlindClockLevel{
		{LevelIndex: 0, SmallBlind: 10, BigBlind: 20, DurationSeconds: 600},
		{LevelIndex: 1, SmallBlind: 20, BigBlind: 40, DurationSeconds: 600},
		{LevelIndex: 2, SmallBlind: 40, BigBlind: 80, DurationSeconds: 600},
	}, now)
	if err != nil {
		t.Fatalf("new blind clock: %v", err)
	}

	return clock
}
