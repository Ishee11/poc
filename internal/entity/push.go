package entity

import "time"

type BlindClockPushSubscription struct {
	Endpoint        string
	KeyAuth         string
	KeyP256DH       string
	UserAgent       string
	NotifyWarning60 bool
	NotifyWarning10 bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type BlindClockPushEvent struct {
	ClockID   BlindClockID
	EventKey  string
	EventKind string
	CreatedAt time.Time
}
