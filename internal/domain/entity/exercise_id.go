package entity

import (
	"fmt"

	"github.com/google/uuid"
)

// ExerciseID represents a unique identifier for an Exercise.
type ExerciseID struct {
	value uuid.UUID
}

// NewExerciseID generates a new random ExerciseID.
func NewExerciseID() ExerciseID {
	return ExerciseID{value: uuid.New()}
}

// NewExerciseIDFromString parses a string into an ExerciseID.
// Returns an error if the string is not a valid UUID.
func NewExerciseIDFromString(s string) (ExerciseID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return ExerciseID{}, fmt.Errorf("invalid ExerciseID format '%s': %w", s, err)
	}
	return ExerciseID{value: id}, nil
}

// String returns the string representation of the ExerciseID.
func (id ExerciseID) String() string {
	return id.value.String()
}

// IsZero checks if the ExerciseID is the zero value.
func (id ExerciseID) IsZero() bool {
	return id.value == uuid.Nil
}

// Equals checks if two ExerciseIDs are equal.
func (id ExerciseID) Equals(other ExerciseID) bool {
	return id.value == other.value
}
