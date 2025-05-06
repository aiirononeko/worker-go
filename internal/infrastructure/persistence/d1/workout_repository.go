package d1

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/aiirononeko/bulktrack-api/internal/domain/workout"
	db "github.com/aiirononeko/bulktrack-api/internal/infrastructure/persistence/d1/sql"
)

// workoutRepository implements the workout.Repository interface using sqlc.
type workoutRepository struct {
	// Hold the sqlc DBTX interface, which can be *sql.DB or *sql.Tx.
	db db.DBTX // Use alias 'db'
}

// NewWorkoutRepository creates a new WorkoutRepository implementation.
// It expects the sqlc.DBTX interface (typically *sql.DB initially).
func NewWorkoutRepository(database db.DBTX) workout.Repository {
	return &workoutRepository{db: database}
}

// SaveWorkout implements the corresponding method from the workout.Repository interface.
func (r *workoutRepository) SaveWorkout(ctx context.Context, w *workout.Workout) error {
	// Create a Queries object specific to the current DBTX (could be DB or Tx).
	q := db.New(r.db) // Use alias 'db'

	// Prepare parameters
	params := db.CreateWorkoutParams{ // Use alias 'db'
		ID:          w.ID().String(),
		DeviceID:    w.DeviceID().String(),                    // Assumes placeholder has String()
		MenuID:      w.MenuID().String(),                      // Assumes placeholder has String()
		PerformedAt: w.PerformedAt().Format(sqliteTimeFormat), // Format time to string
		CreatedAt:   w.CreatedAt().Format(sqliteTimeFormat),   // Format time to string
		UpdatedAt:   w.UpdatedAt().Format(sqliteTimeFormat),   // Format time to string
	}

	// Execute the query using the Queries object.
	err := q.CreateWorkout(ctx, params)
	if err != nil {
		log.Printf("ERROR: Failed to execute CreateWorkout query for Workout ID %s: %v", w.ID().String(), err)
		// Wrap the error for context.
		return fmt.Errorf("d1 query CreateWorkout failed: %w", err)
	}
	log.Printf("INFO: Successfully saved Workout ID %s", w.ID().String())
	return nil
}

// SaveWorkoutSets implements the corresponding method from the workout.Repository interface.
func (r *workoutRepository) SaveWorkoutSets(ctx context.Context, sets []*workout.WorkoutSet) error {
	// Create a Queries object specific to the current DBTX.
	q := db.New(r.db) // Use alias 'db'

	// Iterate and save each set.
	for i, s := range sets { // Add index for logging
		params := db.CreateWorkoutSetParams{ // Use alias 'db'
			ID:         s.ID().String(),
			WorkoutID:  s.WorkoutID().String(),
			ExerciseID: s.ExerciseID().String(),
			Weight:     s.Weight().Float64(),
			Reps:       int64(s.Reps().Int()),
			// Volume is generated column
			Rpe: sql.NullFloat64{Valid: false},
			Rir: sql.NullInt64{Valid: false},
			// CreatedAt, UpdatedAt not in workout_sets table
		}
		if rpe := s.RPE(); rpe != nil {
			params.Rpe.Float64 = (*rpe).Float64()
			params.Rpe.Valid = true
		}
		if rir := s.RIR(); rir != nil {
			params.Rir.Int64 = int64((*rir).Int())
			params.Rir.Valid = true
		}

		// Execute the query.
		err := q.CreateWorkoutSet(ctx, params)
		if err != nil {
			log.Printf("ERROR: Failed to execute CreateWorkoutSet query for Set ID %s (index %d) in Workout ID %s: %v",
				s.ID().String(), i, s.WorkoutID().String(), err)
			// If one set fails, the transaction (if active) should be rolled back by the caller (CommandHandler).
			return fmt.Errorf("d1 query CreateWorkoutSet failed for set index %d: %w", i, err)
		}
	}
	// Log WorkoutID from the first set if sets is not empty
	if len(sets) > 0 {
		log.Printf("INFO: Successfully saved %d sets for Workout ID %s", len(sets), sets[0].WorkoutID().String())
	}
	return nil
}
