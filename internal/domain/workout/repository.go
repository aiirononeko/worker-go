package workout

import (
	"context"

	"github.com/aiirononeko/bulktrack-api/internal/domain/entity"
)

// WorkoutRepository defines the interface for workout data persistence.
//
//go:generate mockery --name WorkoutRepository --output ./mocks --inpackage
type WorkoutRepository interface {
	// CreateWorkout saves the main workout session details.
	CreateWorkout(ctx context.Context, workout *Workout) error

	// CreateWorkoutSet saves a single workout set associated with a workout.
	// Consider if a transaction context needs to be explicitly passed or handled.
	CreateWorkoutSet(ctx context.Context, set *WorkoutSet) error

	// FindWorkoutByID retrieves a workout by its ID, potentially including its sets.
	FindWorkoutByID(ctx context.Context, id entity.WorkoutID) (*Workout, error)

	// ListWorkoutsByDeviceID retrieves a list of workouts for a device.
	ListWorkoutsByDeviceID(ctx context.Context, deviceID entity.DeviceID) ([]Workout, error)
}
