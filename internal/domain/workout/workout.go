package workout

import (
	"time"

	"github.com/aiirononeko/bulktrack-api/internal/domain/entity"
)

// Workout represents a workout session aggregate root.
type Workout struct {
	ID          entity.WorkoutID
	DeviceID    entity.DeviceID
	MenuID      entity.MenuID // Reference Menu entity ID
	PerformedAt time.Time     // When the workout was actually done
	Notes       *string       // Optional notes for the workout
	Sets        []*WorkoutSet // Slice of pointers to WorkoutSet
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// WorkoutSet represents a single set within a workout session.
type WorkoutSet struct {
	ID        entity.WorkoutSetID
	WorkoutID entity.WorkoutID // Back-reference to the aggregate root
	SetOrder  int              // 1-based order of the set
	Weight    float64
	Reps      int
	Interval  *int // Rest interval *after* this set in seconds (nullable). Renamed from RestDuration.
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AddSet adds a workout set to the workout aggregate.
// Modifies the passed set pointer.
func (w *Workout) AddSet(set *WorkoutSet) { // Takes pointer to WorkoutSet
	set.WorkoutID = w.ID
	set.SetOrder = len(w.Sets) + 1 // Assign 1-based order
	w.Sets = append(w.Sets, set)
	// Optionally update the Workout's UpdatedAt timestamp here
	w.UpdatedAt = time.Now().UTC()
}

// NewWorkout creates a new workout session instance.
// Sets are added separately via AddSet.
func NewWorkout(deviceID entity.DeviceID, menuID entity.MenuID, performedAt time.Time, notes *string) *Workout {
	now := time.Now().UTC()
	return &Workout{
		ID:          entity.NewWorkoutID(),
		DeviceID:    deviceID,
		MenuID:      menuID,
		PerformedAt: performedAt.UTC(), // Ensure UTC
		Notes:       notes,
		Sets:        make([]*WorkoutSet, 0), // Initialize slice of pointers
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// NewWorkoutSet creates a new workout set instance.
// WorkoutID and SetOrder are typically set when added to a Workout aggregate.
func NewWorkoutSet(weight float64, reps int, interval *int) *WorkoutSet {
	now := time.Now().UTC()
	return &WorkoutSet{
		ID:        entity.NewWorkoutSetID(),
		Weight:    weight,
		Reps:      reps,
		Interval:  interval,
		CreatedAt: now,
		UpdatedAt: now,
		// WorkoutID and SetOrder are set later by AddSet
	}
}
