package entity

import "time"

type BlindClockID string
type BlindClockStatus string

const (
	BlindClockStatusIdle     BlindClockStatus = "idle"
	BlindClockStatusRunning  BlindClockStatus = "running"
	BlindClockStatusPaused   BlindClockStatus = "paused"
	BlindClockStatusFinished BlindClockStatus = "finished"
)

type BlindClockLevel struct {
	LevelIndex      int
	SmallBlind      int64
	BigBlind        int64
	DurationSeconds int64
}

type BlindClock struct {
	id                      BlindClockID
	status                  BlindClockStatus
	levels                  []BlindClockLevel
	startedAt               *time.Time
	pausedAt                *time.Time
	finishedAt              *time.Time
	accumulatedPauseSeconds int64
	createdAt               time.Time
	updatedAt               time.Time
}

type BlindClockSnapshot struct {
	Status            BlindClockStatus
	CurrentLevelIndex int
	RemainingSeconds  int64
}

func NewBlindClock(id BlindClockID, levels []BlindClockLevel, now time.Time) (*BlindClock, error) {
	if err := validateBlindClockLevels(levels); err != nil {
		return nil, err
	}

	copied := cloneBlindClockLevels(levels)
	return &BlindClock{
		id:        id,
		status:    BlindClockStatusIdle,
		levels:    copied,
		createdAt: now,
		updatedAt: now,
	}, nil
}

func RestoreBlindClock(
	id BlindClockID,
	status BlindClockStatus,
	levels []BlindClockLevel,
	startedAt *time.Time,
	pausedAt *time.Time,
	finishedAt *time.Time,
	accumulatedPauseSeconds int64,
	createdAt time.Time,
	updatedAt time.Time,
) (*BlindClock, error) {
	if err := validateBlindClockLevels(levels); err != nil {
		return nil, err
	}

	return &BlindClock{
		id:                      id,
		status:                  status,
		levels:                  cloneBlindClockLevels(levels),
		startedAt:               cloneTimePtr(startedAt),
		pausedAt:                cloneTimePtr(pausedAt),
		finishedAt:              cloneTimePtr(finishedAt),
		accumulatedPauseSeconds: accumulatedPauseSeconds,
		createdAt:               createdAt,
		updatedAt:               updatedAt,
	}, nil
}

func (c *BlindClock) ID() BlindClockID               { return c.id }
func (c *BlindClock) Status() BlindClockStatus       { return c.status }
func (c *BlindClock) Levels() []BlindClockLevel      { return cloneBlindClockLevels(c.levels) }
func (c *BlindClock) StartedAt() *time.Time          { return cloneTimePtr(c.startedAt) }
func (c *BlindClock) PausedAt() *time.Time           { return cloneTimePtr(c.pausedAt) }
func (c *BlindClock) FinishedAt() *time.Time         { return cloneTimePtr(c.finishedAt) }
func (c *BlindClock) AccumulatedPauseSeconds() int64 { return c.accumulatedPauseSeconds }
func (c *BlindClock) CreatedAt() time.Time           { return c.createdAt }
func (c *BlindClock) UpdatedAt() time.Time           { return c.updatedAt }

func (c *BlindClock) Start(now time.Time) error {
	if len(c.levels) == 0 {
		return ErrBlindClockHasNoLevels
	}
	if c.status == BlindClockStatusRunning {
		return ErrBlindClockAlreadyRunning
	}
	if c.status == BlindClockStatusPaused {
		return ErrBlindClockNotPaused
	}
	if c.status == BlindClockStatusFinished {
		return ErrBlindClockFinished
	}

	c.status = BlindClockStatusRunning
	c.startedAt = &now
	c.pausedAt = nil
	c.finishedAt = nil
	c.accumulatedPauseSeconds = 0
	c.updatedAt = now
	return nil
}

func (c *BlindClock) Pause(now time.Time) error {
	if c.status != BlindClockStatusRunning {
		return ErrBlindClockNotRunning
	}
	c.status = BlindClockStatusPaused
	c.pausedAt = &now
	c.updatedAt = now
	return nil
}

func (c *BlindClock) Resume(now time.Time) error {
	if c.status != BlindClockStatusPaused {
		return ErrBlindClockNotPaused
	}
	if c.pausedAt == nil {
		return ErrBlindClockNotPaused
	}

	c.accumulatedPauseSeconds += int64(now.Sub(*c.pausedAt).Seconds())
	c.status = BlindClockStatusRunning
	c.pausedAt = nil
	c.updatedAt = now
	return nil
}

func (c *BlindClock) Reset(now time.Time) {
	c.status = BlindClockStatusIdle
	c.startedAt = nil
	c.pausedAt = nil
	c.finishedAt = nil
	c.accumulatedPauseSeconds = 0
	c.updatedAt = now
}

func (c *BlindClock) ReplaceLevels(levels []BlindClockLevel, now time.Time) error {
	if err := validateBlindClockLevels(levels); err != nil {
		return err
	}
	c.levels = cloneBlindClockLevels(levels)
	c.updatedAt = now
	return nil
}

func (c *BlindClock) MoveToPreviousLevel(now time.Time) error {
	return c.moveToLevelOffset(now, -1)
}

func (c *BlindClock) MoveToNextLevel(now time.Time) error {
	return c.moveToLevelOffset(now, 1)
}

func (c *BlindClock) Sync(now time.Time) bool {
	if c.status != BlindClockStatusRunning && c.status != BlindClockStatusPaused {
		return false
	}

	snapshot := c.Snapshot(now)
	if snapshot.Status != BlindClockStatusFinished {
		return false
	}

	c.status = BlindClockStatusFinished
	c.pausedAt = nil
	c.finishedAt = &now
	c.updatedAt = now
	return true
}

func (c *BlindClock) moveToLevelOffset(now time.Time, offset int) error {
	if len(c.levels) == 0 {
		return ErrBlindClockHasNoLevels
	}

	snapshot := c.Snapshot(now)
	currentIndex := snapshot.CurrentLevelIndex
	if currentIndex < 0 {
		currentIndex = 0
	}

	target := currentIndex + offset
	if target < 0 || target >= len(c.levels) {
		return ErrInvalidOperation
	}

	var elapsedBeforeTarget int64
	for idx := 0; idx < target; idx++ {
		elapsedBeforeTarget += c.levels[idx].DurationSeconds
	}

	startedAt := now.Add(-time.Duration(elapsedBeforeTarget) * time.Second)
	c.startedAt = &startedAt
	c.finishedAt = nil
	c.accumulatedPauseSeconds = 0
	c.updatedAt = now

	switch c.status {
	case BlindClockStatusRunning:
		c.pausedAt = nil
	case BlindClockStatusIdle, BlindClockStatusPaused, BlindClockStatusFinished:
		c.status = BlindClockStatusPaused
		c.pausedAt = &now
	default:
		c.pausedAt = nil
	}

	return nil
}

func (c *BlindClock) Snapshot(now time.Time) BlindClockSnapshot {
	if len(c.levels) == 0 {
		return BlindClockSnapshot{
			Status:            BlindClockStatusIdle,
			CurrentLevelIndex: -1,
			RemainingSeconds:  0,
		}
	}

	if c.status == BlindClockStatusIdle || c.startedAt == nil {
		return BlindClockSnapshot{
			Status:            BlindClockStatusIdle,
			CurrentLevelIndex: 0,
			RemainingSeconds:  c.levels[0].DurationSeconds,
		}
	}

	elapsed := c.elapsedSeconds(now)
	total := c.TotalDurationSeconds()
	if elapsed >= total {
		return BlindClockSnapshot{
			Status:            BlindClockStatusFinished,
			CurrentLevelIndex: len(c.levels) - 1,
			RemainingSeconds:  0,
		}
	}

	var cumulative int64
	for idx, level := range c.levels {
		next := cumulative + level.DurationSeconds
		if elapsed < next {
			return BlindClockSnapshot{
				Status:            c.status,
				CurrentLevelIndex: idx,
				RemainingSeconds:  next - elapsed,
			}
		}
		cumulative = next
	}

	return BlindClockSnapshot{
		Status:            BlindClockStatusFinished,
		CurrentLevelIndex: len(c.levels) - 1,
		RemainingSeconds:  0,
	}
}

func (c *BlindClock) TotalDurationSeconds() int64 {
	var total int64
	for _, level := range c.levels {
		total += level.DurationSeconds
	}
	return total
}

func (c *BlindClock) elapsedSeconds(now time.Time) int64 {
	if c.startedAt == nil {
		return 0
	}

	var end time.Time
	switch c.status {
	case BlindClockStatusPaused:
		if c.pausedAt == nil {
			return 0
		}
		end = *c.pausedAt
	case BlindClockStatusFinished:
		if c.finishedAt == nil {
			end = now
		} else {
			end = *c.finishedAt
		}
	default:
		end = now
	}

	elapsed := int64(end.Sub(*c.startedAt).Seconds()) - c.accumulatedPauseSeconds
	if elapsed < 0 {
		return 0
	}
	return elapsed
}

func validateBlindClockLevels(levels []BlindClockLevel) error {
	for idx, level := range levels {
		if level.SmallBlind <= 0 || level.BigBlind <= 0 || level.DurationSeconds <= 0 {
			return ErrInvalidBlindClockLevel
		}
		if level.BigBlind < level.SmallBlind {
			return ErrInvalidBlindClockLevel
		}
		if level.LevelIndex != idx {
			return ErrInvalidBlindClockLevel
		}
	}
	return nil
}

func cloneBlindClockLevels(levels []BlindClockLevel) []BlindClockLevel {
	if len(levels) == 0 {
		return nil
	}
	copied := make([]BlindClockLevel, len(levels))
	copy(copied, levels)
	return copied
}

func cloneTimePtr(v *time.Time) *time.Time {
	if v == nil {
		return nil
	}
	t := *v
	return &t
}
