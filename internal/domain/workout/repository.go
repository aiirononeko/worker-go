package workout

import (
	"context"
)

// Repository defines the persistence operations for Workouts and WorkoutSets
// within the workout domain context.
type Repository interface {
	// SaveWorkout saves the main workout record.
	// Expects context to potentially contain a repository.DBTX via repository.TxKey.
	SaveWorkout(ctx context.Context, workout *Workout) error

	// SaveWorkoutSets saves multiple workout sets.
	// Expects context to potentially contain a repository.DBTX via repository.TxKey.
	SaveWorkoutSets(ctx context.Context, sets []*WorkoutSet) error

	// FindWorkoutByID retrieves a workout by its ID.
	// FindWorkoutByID(ctx context.Context, id WorkoutID) (*Workout, error)

	// FindWorkoutSetsByWorkoutID retrieves sets for a given workout ID.
	// FindWorkoutSetsByWorkoutID(ctx context.Context, workoutID WorkoutID) ([]*WorkoutSet, error)
}
