package workout

import (
	"errors"

	"github.com/google/uuid"
)

// --- IDs specific to Workout domain ---

// WorkoutID represents a unique identifier for a workout session.
type WorkoutID uuid.UUID

func NewWorkoutID(id uuid.UUID) WorkoutID { return WorkoutID(id) }
func (id WorkoutID) UUID() uuid.UUID      { return uuid.UUID(id) }
func (id WorkoutID) String() string       { return id.UUID().String() }

// SetID represents a unique identifier for a workout set.
type SetID uuid.UUID

func NewSetID(id uuid.UUID) SetID { return SetID(id) }
func (id SetID) UUID() uuid.UUID  { return uuid.UUID(id) }
func (id SetID) String() string   { return id.UUID().String() }

// ExerciseID represents a unique identifier for an exercise type.
type ExerciseID uuid.UUID

func NewExerciseID(id uuid.UUID) ExerciseID { return ExerciseID(id) }
func (id ExerciseID) UUID() uuid.UUID       { return uuid.UUID(id) }
func (id ExerciseID) String() string        { return id.UUID().String() }

// --- Metrics ---

// Weight represents the weight used in a set.
type Weight float64

func NewWeight(w float64) (Weight, error) {
	if w < 0 {
		return 0, errors.New("weight cannot be negative")
	}
	return Weight(w), nil
}
func (w Weight) Float64() float64 { return float64(w) }

// Reps represents the number of repetitions performed.
type Reps int

func NewReps(r int) (Reps, error) {
	if r < 0 {
		return 0, errors.New("reps cannot be negative")
	}
	return Reps(r), nil
}
func (r Reps) Int() int { return int(r) }

// RPE represents the Rate of Perceived Exertion.
type RPE float64

func NewRPE(rpe float64) (RPE, error) {
	if rpe < 0 || rpe > 10 {
		return 0, errors.New("RPE must be between 0 and 10")
	}
	return RPE(rpe), nil
}
func (r RPE) Float64() float64 { return float64(r) }

// RIR represents the Reps In Reserve.
type RIR int

func NewRIR(rir int) (RIR, error) {
	if rir < 0 || rir > 10 {
		return 0, errors.New("RIR must be between 0 and 10")
	}
	return RIR(rir), nil
}
func (r RIR) Int() int { return int(r) }

// Volume represents the calculated volume of a set (Weight * Reps).
type Volume float64

// CalculateVolume computes the volume for a set.
func CalculateVolume(weight Weight, reps Reps) Volume {
	w := weight.Float64()
	r := float64(reps.Int())
	if w < 0 {
		w = 0
	}
	if r < 0 {
		r = 0
	}
	return Volume(w * r)
}
func (v Volume) Float64() float64 { return float64(v) }

// --- Shared Value Objects (Placeholders or to be imported) ---
// These represent dependencies on other domains' value objects.
// Actual implementation will require importing them from their respective packages
// e.g., "github.com/aiirononeko/bulktrack-api/internal/domain/device/value"

// DeviceID is a placeholder for the actual DeviceID type from the device domain.
type DeviceIDType interface {
	String() string
	// Add other methods if DeviceID has them and they are used by workout domain
}

// UserID is a placeholder for the actual UserID type from the user domain.
type UserIDType interface {
	String() string
	// Add other methods if UserID has them
}

// MenuID is a placeholder for the actual MenuID type from the menu domain.
type MenuIDType interface {
	String() string
	// Add other methods if MenuID has them
}
