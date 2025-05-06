package entity

import (
	"fmt"

	"github.com/google/uuid"
)

// WorkoutID はワークアウトセッションの一意な識別子を表します。
type WorkoutID uuid.UUID

// NewWorkoutID は新しい WorkoutID (UUID v4) を生成します。
func NewWorkoutID() WorkoutID {
	return WorkoutID(uuid.New())
}

// NewWorkoutIDFromString は文字列から WorkoutID を生成します。
func NewWorkoutIDFromString(id string) (WorkoutID, error) {
	u, err := uuid.Parse(id)
	if err != nil {
		return WorkoutID(uuid.Nil), fmt.Errorf("invalid workout ID format '%s': %w", id, err)
	}
	return WorkoutID(u), nil
}

// String は WorkoutID を文字列として返します。
func (wid WorkoutID) String() string {
	return uuid.UUID(wid).String()
}

// WorkoutSetID はワークアウトセットの一意な識別子を表します。
type WorkoutSetID uuid.UUID

// NewWorkoutSetID は新しい WorkoutSetID (UUID v4) を生成します。
func NewWorkoutSetID() WorkoutSetID {
	return WorkoutSetID(uuid.New())
}

// NewWorkoutSetIDFromString は文字列から WorkoutSetID を生成します。
func NewWorkoutSetIDFromString(id string) (WorkoutSetID, error) {
	u, err := uuid.Parse(id)
	if err != nil {
		return WorkoutSetID(uuid.Nil), fmt.Errorf("invalid workout set ID format '%s': %w", id, err)
	}
	return WorkoutSetID(u), nil
}

// String は WorkoutSetID を文字列として返します。
func (wsid WorkoutSetID) String() string {
	return uuid.UUID(wsid).String()
}
