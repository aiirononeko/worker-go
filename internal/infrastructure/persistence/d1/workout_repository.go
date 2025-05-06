package d1

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/aiirononeko/bulktrack-api/internal/app/apperror"
	"github.com/aiirononeko/bulktrack-api/internal/domain/entity"
	"github.com/aiirononeko/bulktrack-api/internal/domain/workout"
	db "github.com/aiirononeko/bulktrack-api/internal/infrastructure/persistence/d1/sql"
)

// D1WorkoutRepository implements the workout.WorkoutRepository interface using Cloudflare D1.
// It relies on sqlc-generated query implementations.
type D1WorkoutRepository struct {
	db      *sql.DB // Changed to *sql.DB to allow BeginTx directly
	queries *db.Queries
}

// NewD1WorkoutRepository creates a new D1WorkoutRepository.
func NewD1WorkoutRepository(database *sql.DB) *D1WorkoutRepository {
	return &D1WorkoutRepository{
		db:      database,
		queries: db.New(database), // db.New can take *sql.DB
	}
}

// CreateWorkout saves the main workout session details and its associated sets.
// NOTE: D1 transactions (db.BeginTx) are not supported in the current environment/driver.
// Operations are performed sequentially without a transaction.
func (r *D1WorkoutRepository) CreateWorkout(ctx context.Context, wk *workout.Workout) error {
	// Transaction removed due to D1 limitations in the current environment.
	// qtx := r.queries // Use r.queries directly instead of qtx from a transaction

	// Persist the main workout record
	_, err := r.queries.CreateWorkout(ctx, db.CreateWorkoutParams{
		ID:          wk.ID.String(),
		DeviceID:    wk.DeviceID.String(),
		MenuID:      wk.MenuID.String(),
		PerformedAt: wk.PerformedAt.Format(time.RFC3339),
		CreatedAt:   wk.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   wk.UpdatedAt.Format(time.RFC3339),
	})
	if err != nil {
		return apperror.NewErrInternal("failed to create workout record in DB (no-tx)", err)
	}

	// Persist each workout set
	for _, set := range wk.Sets {
		params := db.CreateWorkoutSetParams{
			ID:         set.ID.String(),
			WorkoutID:  wk.ID.String(),
			ExerciseID: set.ExerciseID.String(),
			SetOrder:   int64(set.SetOrder),
			Weight:     set.Weight,
			Reps:       int64(set.Reps),
			Interval:   sql.NullInt64{Int64: int64OrZero(set.Interval), Valid: set.Interval != nil},
			CreatedAt:  set.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  set.UpdatedAt.Format(time.RFC3339),
		}
		_, err = r.queries.CreateWorkoutSet(ctx, params)
		if err != nil {
			// If a set fails, the workout record is already created. This is not atomic.
			// Consider manual cleanup or accept potential inconsistency if atomicity is critical and transactions are unavailable.
			return apperror.NewErrInternal(fmt.Sprintf("failed to create workout set record in DB (no-tx, set_id: %s)", set.ID.String()), err)
		}
	}

	// Commit removed as transaction is not used.

	return nil
}

// CreateWorkoutSet saves a single workout set.
// This method might be used if sets are created independently or added to an existing workout.
func (r *D1WorkoutRepository) CreateWorkoutSet(ctx context.Context, set *workout.WorkoutSet) error {
	params := db.CreateWorkoutSetParams{
		ID:         set.ID.String(),
		WorkoutID:  set.WorkoutID.String(),
		ExerciseID: set.ExerciseID.String(),
		SetOrder:   int64(set.SetOrder),
		Weight:     set.Weight,
		Reps:       int64(set.Reps),
		Interval:   sql.NullInt64{Int64: int64OrZero(set.Interval), Valid: set.Interval != nil},
		CreatedAt:  set.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  set.UpdatedAt.Format(time.RFC3339),
	}
	_, err := r.queries.CreateWorkoutSet(ctx, params)
	if err != nil {
		return apperror.NewErrInternal("failed to create workout set record", err)
	}
	return nil
}

// FindWorkoutByID retrieves a workout by its ID including its sets.
func (r *D1WorkoutRepository) FindWorkoutByID(ctx context.Context, id entity.WorkoutID) (*workout.Workout, error) {
	dbWorkout, err := r.queries.GetWorkout(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.NewErrNotFound("Workout", id.String())
		}
		return nil, apperror.NewErrInternal(fmt.Sprintf("failed to get workout by id %s from DB", id.String()), err)
	}

	dbSets, err := r.queries.ListWorkoutSetsByWorkoutId(ctx, id.String())
	if err != nil {
		return nil, apperror.NewErrInternal(fmt.Sprintf("failed to list workout sets by workout id %s from DB", id.String()), err)
	}

	return r.mapDBWorkoutToDomain(dbWorkout, dbSets)
}

// ListWorkoutsByDeviceID retrieves a list of workouts for a device, including their sets.
func (r *D1WorkoutRepository) ListWorkoutsByDeviceID(ctx context.Context, deviceID entity.DeviceID) ([]workout.Workout, error) {
	dbWorkouts, err := r.queries.ListWorkoutsByDeviceId(ctx, deviceID.String())
	if err != nil {
		if err == sql.ErrNoRows { // Should return empty slice, not error
			return []workout.Workout{}, nil
		}
		return nil, apperror.NewErrInternal(fmt.Sprintf("failed to list workouts by device id %s from DB", deviceID.String()), err)
	}

	workouts := make([]workout.Workout, 0, len(dbWorkouts))
	for _, dbWk := range dbWorkouts {
		dbSets, err := r.queries.ListWorkoutSetsByWorkoutId(ctx, dbWk.ID)
		if err != nil {
			return nil, apperror.NewErrInternal(fmt.Sprintf("failed to list sets for workout id %s from DB", dbWk.ID), err)
		}
		domainWorkout, err := r.mapDBWorkoutToDomain(dbWk, dbSets)
		if err != nil {
			return nil, err // Error during mapping, should already be an apperror type or wrapped internal
		}
		workouts = append(workouts, *domainWorkout)
	}

	return workouts, nil
}

func (r *D1WorkoutRepository) mapDBWorkoutToDomain(dbWk db.Workout, dbSets []db.WorkoutSet) (*workout.Workout, error) {
	workoutID, err := entity.NewWorkoutIDFromString(dbWk.ID)
	if err != nil {
		return nil, apperror.NewErrInternal(fmt.Sprintf("invalid workout ID from DB '%s'", dbWk.ID), err)
	}
	parsedDeviceID, err := entity.NewDeviceID(dbWk.DeviceID)
	if err != nil {
		return nil, apperror.NewErrInternal(fmt.Sprintf("failed to parse device ID from DB '%s'", dbWk.DeviceID), err)
	}

	menuID, err := entity.NewMenuIDFromString(dbWk.MenuID)
	if err != nil {
		return nil, apperror.NewErrInternal(fmt.Sprintf("invalid menu ID from DB '%s'", dbWk.MenuID), err)
	}
	performedAt, err := parseTime(dbWk.PerformedAt)
	if err != nil {
		return nil, apperror.NewErrInternal(fmt.Sprintf("invalid performed_at from DB '%s'", dbWk.PerformedAt), err)
	}
	createdAt, err := parseTime(dbWk.CreatedAt)
	if err != nil {
		return nil, apperror.NewErrInternal(fmt.Sprintf("invalid created_at from DB '%s'", dbWk.CreatedAt), err)
	}
	updatedAt, err := parseTime(dbWk.UpdatedAt)
	if err != nil {
		return nil, apperror.NewErrInternal(fmt.Sprintf("invalid updated_at from DB '%s'", dbWk.UpdatedAt), err)
	}

	var notes *string
	// if dbWk.Notes.Valid { // Uncomment if/when Notes field exists in db.Workout
	// 	notes = &dbWk.Notes.String
	// }

	domainWorkout := &workout.Workout{
		ID:          workoutID,
		DeviceID:    parsedDeviceID,
		MenuID:      menuID,
		PerformedAt: performedAt,
		Notes:       notes,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		Sets:        make([]*workout.WorkoutSet, len(dbSets)),
	}

	for i, dbSet := range dbSets {
		setID, err := entity.NewWorkoutSetIDFromString(dbSet.ID)
		if err != nil {
			return nil, apperror.NewErrInternal(fmt.Sprintf("invalid workout set ID from DB '%s'", dbSet.ID), err)
		}
		setExerciseID, err := entity.NewExerciseIDFromString(dbSet.ExerciseID)
		if err != nil {
			return nil, apperror.NewErrInternal(fmt.Sprintf("invalid exercise_id for set from DB '%s'", dbSet.ExerciseID), err)
		}
		setWorkoutID, err := entity.NewWorkoutIDFromString(dbSet.WorkoutID)
		if err != nil {
			return nil, apperror.NewErrInternal(fmt.Sprintf("invalid workout_id for set from DB '%s'", dbSet.WorkoutID), err)
		}
		if setWorkoutID != domainWorkout.ID {
			return nil, apperror.NewErrInternal(fmt.Sprintf("inconsistent workout_id for set '%s', expected '%s', got '%s'", setID.String(), domainWorkout.ID.String(), setWorkoutID.String()), nil)
		}

		setCreatedAt, err := parseTime(dbSet.CreatedAt)
		if err != nil {
			return nil, apperror.NewErrInternal(fmt.Sprintf("invalid created_at for set from DB '%s'", dbSet.CreatedAt), err)
		}
		setUpdatedAt, err := parseTime(dbSet.UpdatedAt)
		if err != nil {
			return nil, apperror.NewErrInternal(fmt.Sprintf("invalid updated_at for set from DB '%s'", dbSet.UpdatedAt), err)
		}

		var interval *int
		if dbSet.Interval.Valid {
			val := int(dbSet.Interval.Int64)
			interval = &val
		}

		domainWorkout.Sets[i] = &workout.WorkoutSet{
			ID:         setID,
			ExerciseID: setExerciseID,
			WorkoutID:  domainWorkout.ID,
			SetOrder:   int(dbSet.SetOrder),
			Weight:     dbSet.Weight,
			Reps:       int(dbSet.Reps),
			Interval:   interval,
			CreatedAt:  setCreatedAt,
			UpdatedAt:  setUpdatedAt,
		}
	}
	return domainWorkout, nil
}

// Helper function to convert *int to int64, returning 0 if nil.
func int64OrZero(val *int) int64 {
	if val == nil {
		return 0
	}
	return int64(*val)
}

// Helper function to parse time strings from DB (assuming ISO8601 like format)
func parseTime(timeStr string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("failed to parse time string '%s': %w", timeStr, err)
}
