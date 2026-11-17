package model

import (
	"time"

	"github.com/google/uuid"
)

type Coordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Position struct {
	ID          int64       `json:"id"`
	TripID      uuid.UUID   `json:"trip_id"`
	Coordinates Coordinates `json:"coordinates"`
	RecordedAt  time.Time   `json:"recorded_at"`
	CreatedAt   time.Time   `json:"-"`
}
