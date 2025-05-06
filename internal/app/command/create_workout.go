package command

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/aiirononeko/bulktrack-api/internal/domain/workout"
)

// WorkoutRepository now refers to the one in the workout domain.
// The previous local definition is removed.
type WorkoutRepository workout.Repository

// TransactionManager interface definition is now in transaction.go

// CreateWorkoutCommand uses types from the workout domain or specific domain value packages.
// CreateWorkoutHandler handles the logic for the CreateWorkoutCommand.
type CreateWorkoutHandler struct {
	repo WorkoutRepository // This now uses the aliased workout.Repository
}

// NewCreateWorkoutHandler creates a new CreateWorkoutHandler.
func NewCreateWorkoutHandler(repo WorkoutRepository) *CreateWorkoutHandler {
	return &CreateWorkoutHandler{
		repo: repo,
	}
}

// Handle executes the create workout command.
func (h *CreateWorkoutHandler) Handle(ctx context.Context, cmd CreateWorkoutCommand) (*CreateWorkoutResponse, error) {
	now := time.Now()
	workoutID := workout.NewWorkoutID(uuid.New())

	workoutEntity := workout.NewWorkout(
		workoutID,
		cmd.DeviceID,
		cmd.MenuID,
		cmd.PerformedAt,
		now, // CreatedAt
		now, // UpdatedAt
	)

	workoutSets := make([]*workout.WorkoutSet, 0, len(cmd.Sets))
	respSets := make([]WorkoutSetResponseItem, 0, len(cmd.Sets))

	for _, setInput := range cmd.Sets {
		setID := workout.NewSetID(uuid.New())
		setEntity, err := workout.NewWorkoutSet(
			setID,
			workoutID,
			setInput.ExerciseID,
			setInput.Weight,
			setInput.Reps,
			setInput.RPE,
			setInput.RIR,
			now, // CreatedAt
			now, // UpdatedAt
		)
		if err != nil {
			return nil, err
		}
		workoutSets = append(workoutSets, setEntity)
		respSets = append(respSets, WorkoutSetResponseItem{
			ID:         setEntity.ID(),
			ExerciseID: setEntity.ExerciseID(),
			Weight:     setEntity.Weight(),
			Reps:       setEntity.Reps(),
			Volume:     setEntity.Volume(),
			RPE:        setEntity.RPE(),
			RIR:        setEntity.RIR(),
		})
	}

	if err := h.repo.SaveWorkout(ctx, workoutEntity); err != nil {
		return nil, err
	}

	if err := h.repo.SaveWorkoutSets(ctx, workoutSets); err != nil {
		return nil, err
	}

	return &CreateWorkoutResponse{
		ID:          workoutEntity.ID(),
		DeviceID:    cmd.DeviceID,
		MenuID:      cmd.MenuID,
		PerformedAt: workoutEntity.PerformedAt(),
		CreatedAt:   workoutEntity.CreatedAt(),
		UpdatedAt:   workoutEntity.UpdatedAt(),
		Sets:        respSets,
	}, nil
}

// CreateWorkoutCommand uses types from the workout domain or specific domain value packages.
type CreateWorkoutCommand struct {
	DeviceID    workout.DeviceIDType // e.g., devValue.DeviceID
	MenuID      workout.MenuIDType   // e.g., menuValue.MenuID
	PerformedAt time.Time
	Sets        []WorkoutSetInput
}

// WorkoutSetInput uses types from the workout domain.
type WorkoutSetInput struct {
	ExerciseID workout.ExerciseID
	Weight     workout.Weight
	Reps       workout.Reps
	RPE        *workout.RPE
	RIR        *workout.RIR
}

// CreateWorkoutResponse uses types from the workout domain.
type CreateWorkoutResponse struct {
	ID          workout.WorkoutID
	DeviceID    workout.DeviceIDType // Consistent with command
	MenuID      workout.MenuIDType   // Consistent with command
	PerformedAt time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Sets        []WorkoutSetResponseItem
}

// WorkoutSetResponseItem uses types from the workout domain.
type WorkoutSetResponseItem struct {
	ID         workout.SetID
	ExerciseID workout.ExerciseID
	Weight     workout.Weight
	Reps       workout.Reps
	Volume     workout.Volume
	RPE        *workout.RPE
	RIR        *workout.RIR
}
