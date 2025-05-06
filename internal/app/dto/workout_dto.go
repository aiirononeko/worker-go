package dto

import "time"

// CreateWorkoutRequest represents the request body for creating a new workout.
type CreateWorkoutRequest struct {
	MenuID      string            `json:"menuId" validate:"required,uuid4"`
	PerformedAt string            `json:"performedAt" validate:"required,datetime=2006-01-02T15:04:05Z07:00"` // ISO8601
	Notes       *string           `json:"notes"`
	Sets        []WorkoutSetInput `json:"sets" validate:"required,min=1,dive"`
}

// WorkoutSetInput represents the input for a single workout set.
type WorkoutSetInput struct {
	ExerciseID string  `json:"exerciseId" validate:"required,uuid4"`
	Weight     float64 `json:"weight" validate:"required,gte=0"`
	Reps       int     `json:"reps" validate:"required,gte=0"`
	Interval   *int    `json:"interval" validate:"omitempty,gte=0"` // Optional, in seconds. Renamed from restDuration.
}

// WorkoutDTO represents the workout data returned to the client.
type WorkoutDTO struct {
	ID          string          `json:"id"`
	MenuID      string          `json:"menuId"`
	PerformedAt time.Time       `json:"performedAt"`
	Notes       *string         `json:"notes,omitempty"`
	Sets        []WorkoutSetDTO `json:"sets"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

// WorkoutSetDTO represents a single workout set returned to the client.
type WorkoutSetDTO struct {
	ID         string  `json:"id"`
	ExerciseID string  `json:"exerciseId"`
	SetOrder   int     `json:"setOrder"`
	Weight     float64 `json:"weight"`
	Reps       int     `json:"reps"`
	Interval   *int    `json:"interval,omitempty"` // Renamed from restDuration.
}
