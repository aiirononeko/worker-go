package workout

import (
	"errors"
	"time"
	// Import necessary value objects from other domains if placeholders are not used
	// For now, we use the placeholder interfaces defined in value.go
)

// Workout represents a single workout session.
type Workout struct {
	id          WorkoutID
	deviceID    DeviceIDType // Use placeholder interface
	userID      *UserIDType  // Use placeholder interface, optional
	menuID      MenuIDType   // Use placeholder interface
	performedAt time.Time
	createdAt   time.Time
	updatedAt   time.Time
}

// NewWorkout creates a new Workout entity.
func NewWorkout(
	id WorkoutID,
	deviceID DeviceIDType,
	menuID MenuIDType,
	performedAt time.Time,
	createdAt time.Time,
	updatedAt time.Time,
	// userID *UserIDType, // Add later if needed
) *Workout {
	return &Workout{
		id:          id,
		deviceID:    deviceID,
		menuID:      menuID,
		performedAt: performedAt,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
		// userID: userID,
	}
}

// Getters for Workout
func (w *Workout) ID() WorkoutID          { return w.id }
func (w *Workout) DeviceID() DeviceIDType { return w.deviceID }
func (w *Workout) UserID() *UserIDType    { return w.userID }
func (w *Workout) MenuID() MenuIDType     { return w.menuID }
func (w *Workout) PerformedAt() time.Time { return w.performedAt }
func (w *Workout) CreatedAt() time.Time   { return w.createdAt }
func (w *Workout) UpdatedAt() time.Time   { return w.updatedAt }

// --- WorkoutSet ---

// WorkoutSet represents a single set performed within a workout.
type WorkoutSet struct {
	id         SetID
	workoutID  WorkoutID // Link back to the parent workout
	exerciseID ExerciseID
	weight     Weight
	reps       Reps
	volume     Volume // Calculated value
	rpe        *RPE   // Optional
	rir        *RIR   // Optional
	createdAt  time.Time
	updatedAt  time.Time
}

// NewWorkoutSet creates a new WorkoutSet entity.
func NewWorkoutSet(
	id SetID,
	workoutID WorkoutID,
	exerciseID ExerciseID,
	weight Weight,
	reps Reps,
	rpe *RPE,
	rir *RIR,
	createdAt time.Time,
	updatedAt time.Time,
) (*WorkoutSet, error) {
	if rpe != nil && rir != nil {
		return nil, errors.New("cannot specify both RPE and RIR for a set")
	}
	volume := CalculateVolume(weight, reps)
	return &WorkoutSet{
		id:         id,
		workoutID:  workoutID,
		exerciseID: exerciseID,
		weight:     weight,
		reps:       reps,
		volume:     volume,
		rpe:        rpe,
		rir:        rir,
		createdAt:  createdAt,
		updatedAt:  updatedAt,
	}, nil
}

// Getters for WorkoutSet
func (s *WorkoutSet) ID() SetID              { return s.id }
func (s *WorkoutSet) WorkoutID() WorkoutID   { return s.workoutID }
func (s *WorkoutSet) ExerciseID() ExerciseID { return s.exerciseID }
func (s *WorkoutSet) Weight() Weight         { return s.weight }
func (s *WorkoutSet) Reps() Reps             { return s.reps }
func (s *WorkoutSet) Volume() Volume         { return s.volume }
func (s *WorkoutSet) RPE() *RPE              { return s.rpe }
func (s *WorkoutSet) RIR() *RIR              { return s.rir }
func (s *WorkoutSet) CreatedAt() time.Time   { return s.createdAt }
func (s *WorkoutSet) UpdatedAt() time.Time   { return s.updatedAt }
