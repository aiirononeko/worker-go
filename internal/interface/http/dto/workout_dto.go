package dto

import "time"

// CreateWorkoutRequestDTO maps to the OpenAPI schema for POST /v1/workouts request body.
type CreateWorkoutRequestDTO struct {
	MenuID      string               `json:"menu_id"`      // UUID string
	PerformedAt time.Time            `json:"performed_at"` // Expect ISO 8601 format from JSON unmarshal
	Sets        []WorkoutSetInputDTO `json:"sets"`
}

// WorkoutSetInputDTO maps to the OpenAPI schema for a single set in the request.
type WorkoutSetInputDTO struct {
	ExerciseID string   `json:"exercise_id"` // UUID string
	Weight     float64  `json:"weight"`
	Reps       int      `json:"reps"`
	RPE        *float64 `json:"rpe"` // Pointer for nullable float
	RIR        *int     `json:"rir"` // Pointer for nullable int
}

// WorkoutResponseDTO maps to the OpenAPI schema for the successful response.
type WorkoutResponseDTO struct {
	ID          string                      `json:"id"`        // UUID string
	DeviceID    string                      `json:"device_id"` // UUID string
	MenuID      string                      `json:"menu_id"`   // UUID string
	PerformedAt time.Time                   `json:"performed_at"`
	CreatedAt   time.Time                   `json:"created_at"`
	UpdatedAt   time.Time                   `json:"updated_at"`
	Sets        []WorkoutSetResponseItemDTO `json:"sets"`
}

// WorkoutSetResponseItemDTO maps to the OpenAPI schema for a single set in the response.
type WorkoutSetResponseItemDTO struct {
	ID         string   `json:"id"`          // UUID string
	ExerciseID string   `json:"exercise_id"` // UUID string
	Weight     float64  `json:"weight"`
	Reps       int      `json:"reps"`
	Volume     float64  `json:"volume"`
	RPE        *float64 `json:"rpe"` // Pointer for nullable float
	RIR        *int     `json:"rir"` // Pointer for nullable int
}
