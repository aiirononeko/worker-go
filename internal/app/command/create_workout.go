package command

import (
	"context"
	"time"

	"github.com/aiirononeko/bulktrack-api/internal/app/apperror"
	"github.com/aiirononeko/bulktrack-api/internal/app/dto"
	"github.com/aiirononeko/bulktrack-api/internal/domain/entity"
	"github.com/aiirononeko/bulktrack-api/internal/domain/workout"
)

// CreateWorkoutCommand represents the command to create a new workout.
type CreateWorkoutCommand struct {
	DeviceID    entity.DeviceID
	MenuID      string // Will be parsed to entity.MenuID
	PerformedAt string // Will be parsed to time.Time
	Notes       *string
	Sets        []dto.WorkoutSetInput
}

// CreateWorkoutResult represents the result of the CreateWorkoutCommand.
type CreateWorkoutResult struct {
	Workout *dto.WorkoutDTO
}

// CreateWorkoutHandler handles the CreateWorkoutCommand.
type CreateWorkoutHandler struct {
	workoutRepo workout.WorkoutRepository
	// menuRepo menu.MenuRepository // Optional: to validate menu existence
	// txManager persistence.TransactionManager // Optional: for managing database transactions
}

// NewCreateWorkoutHandler creates a new CreateWorkoutHandler.
func NewCreateWorkoutHandler(
	workoutRepo workout.WorkoutRepository,
	// menuRepo menu.MenuRepository,
	// txManager persistence.TransactionManager,
) *CreateWorkoutHandler {
	return &CreateWorkoutHandler{
		workoutRepo: workoutRepo,
		// menuRepo: menuRepo,
		// txManager: txManager,
	}
}

// Handle executes the create workout command.
// It will parse inputs, create domain entities, and persist them.
func (h *CreateWorkoutHandler) Handle(ctx context.Context, cmd CreateWorkoutCommand) (*CreateWorkoutResult, error) {
	// TODO: Implement the handler logic
	// 1. Parse MenuID string to entity.MenuID
	// 2. Parse PerformedAt string to time.Time
	// 3. Validate MenuID exists (using menuRepo if injected)
	// 4. Create workout.NewWorkout domain entity
	// 5. For each set in cmd.Sets, create workout.NewWorkoutSet and add to Workout entity
	// 6. Persist using workoutRepo (potentially within a transaction)
	//    - workoutRepo.CreateWorkout(ctx, workoutEntity)
	//    - For each set: workoutRepo.CreateWorkoutSet(ctx, setEntity)
	// 7. Map persisted entities back to dto.WorkoutDTO
	// 8. Return result

	// Placeholder implementation
	parsedPerformedAt, err := time.Parse(time.RFC3339Nano, cmd.PerformedAt)
	if err != nil {
		return nil, apperror.NewErrBadRequest("Invalid performedAt format", err.Error())
	}

	parsedMenuID, err := entity.NewMenuIDFromString(cmd.MenuID)
	if err != nil {
		return nil, apperror.NewErrBadRequest("Invalid menu ID format", err.Error())
	}

	workoutEntity := workout.NewWorkout(cmd.DeviceID, parsedMenuID, parsedPerformedAt, cmd.Notes)

	for _, setInput := range cmd.Sets {
		exerciseID, err := entity.NewExerciseIDFromString(setInput.ExerciseID)
		if err != nil {
			return nil, apperror.NewErrBadRequest("Invalid exercise ID format in set", err.Error())
		}
		setEntity := workout.NewWorkoutSet(exerciseID, setInput.Weight, setInput.Reps, setInput.Interval)
		workoutEntity.AddSet(setEntity)
	}

	// NOTE: Transaction removed in repository layer due to D1 limitations.
	if err := h.workoutRepo.CreateWorkout(ctx, workoutEntity); err != nil {
		return nil, err // repo layer should return appropriate apperror
	}

	// Map to DTO
	workoutDTO := &dto.WorkoutDTO{
		ID:          workoutEntity.ID.String(),
		MenuID:      workoutEntity.MenuID.String(),
		PerformedAt: workoutEntity.PerformedAt,
		Notes:       workoutEntity.Notes,
		CreatedAt:   workoutEntity.CreatedAt,
		UpdatedAt:   workoutEntity.UpdatedAt,
		Sets:        make([]dto.WorkoutSetDTO, len(workoutEntity.Sets)),
	}

	for i, s := range workoutEntity.Sets {
		workoutDTO.Sets[i] = dto.WorkoutSetDTO{
			ID:         s.ID.String(),
			ExerciseID: s.ExerciseID.String(),
			SetOrder:   s.SetOrder,
			Weight:     s.Weight,
			Reps:       s.Reps,
			Interval:   s.Interval,
		}
	}

	return &CreateWorkoutResult{Workout: workoutDTO}, nil
}
