package command

import (
	"context"
	"fmt"

	"github.com/aiirononeko/bulktrack-api/internal/app/apperror"
	"github.com/aiirononeko/bulktrack-api/internal/app/dto"
	"github.com/aiirononeko/bulktrack-api/internal/domain/entity"
	"github.com/aiirononeko/bulktrack-api/internal/domain/menu"
	// "github.com/aiirononeko/bulktrack-api/internal/domain/exercise" // ExerciseRepository用 (後で作成)
)

// UpdateMenuExercisesCommand defines the command for updating exercises in a menu.
type UpdateMenuExercisesCommand struct {
	MenuID    string // Parsed from path
	DeviceID  string // From JWT
	Exercises []dto.UpdateMenuExerciseItemDTO
}

// UpdateMenuExercisesResult is the result of successfully updating menu exercises.
type UpdateMenuExercisesResult struct {
	Menu dto.MenuWithExercisesResponseDTO // The updated menu with its exercises
}

// UpdateMenuExercisesHandler handles the UpdateMenuExercisesCommand.
type UpdateMenuExercisesHandler struct {
	menuRepo menu.MenuRepository
	// exerciseRepo exercise.Repository // ExerciseRepository (後で作成・注入)
}

// NewUpdateMenuExercisesHandler creates a new UpdateMenuExercisesHandler.
func NewUpdateMenuExercisesHandler(
	menuRepo menu.MenuRepository,
	// exerciseRepo exercise.Repository,
) *UpdateMenuExercisesHandler {
	return &UpdateMenuExercisesHandler{
		menuRepo: menuRepo,
		// exerciseRepo: exerciseRepo,
	}
}

// Handle executes the command.
func (h *UpdateMenuExercisesHandler) Handle(ctx context.Context, cmd UpdateMenuExercisesCommand) (*UpdateMenuExercisesResult, error) {
	menuID, err := entity.NewMenuIDFromString(cmd.MenuID)
	if err != nil {
		return nil, apperror.NewErrBadRequest("Invalid menu ID format", err.Error())
	}
	deviceID, err := entity.NewDeviceID(cmd.DeviceID)
	if err != nil {
		return nil, apperror.NewErrInternal("Invalid device ID format from token", err)
	}

	positions := make(map[int]bool)
	domainMenuExercises := make([]menu.MenuExerciseItem, len(cmd.Exercises))
	for i, exDTO := range cmd.Exercises {
		if _, exists := positions[exDTO.Position]; exists {
			return nil, apperror.NewErrBadRequest(fmt.Sprintf("Duplicate position %d found for exercises", exDTO.Position), "")
		}
		positions[exDTO.Position] = true

		exerciseID, err := entity.NewExerciseIDFromString(exDTO.ExerciseID)
		if err != nil {
			return nil, apperror.NewErrBadRequest(fmt.Sprintf("Invalid exercise ID format '%s'", exDTO.ExerciseID), err.Error())
		}
		// TODO: Check exercise existence using exerciseRepo.FindExerciseByID(ctx, exerciseID)

		domainMenuExercises[i] = menu.MenuExerciseItem{
			ExerciseID: exerciseID,
			Position:   exDTO.Position,
		}
	}

	err = h.menuRepo.UpdateMenuExercises(ctx, menuID, deviceID, domainMenuExercises)
	if err != nil {
		return nil, err
	}

	updatedMenu, err := h.menuRepo.FindMenuByID(ctx, menuID, deviceID)
	if err != nil {
		return nil, apperror.NewErrInternal("Failed to retrieve updated menu after update", err)
	}
	updatedMenuExercises, err := h.menuRepo.ListMenuExercises(ctx, menuID)
	if err != nil {
		return nil, apperror.NewErrInternal("Failed to retrieve updated menu exercises after update", err)
	}
	updatedMenu.Exercises = updatedMenuExercises

	responseDTO := mapMenuToMenuWithExercisesResponseDTO(updatedMenu)

	return &UpdateMenuExercisesResult{Menu: responseDTO}, nil
}

func mapMenuToMenuWithExercisesResponseDTO(m *menu.Menu) dto.MenuWithExercisesResponseDTO {
	exerciseDTOs := make([]dto.MenuExerciseItemResponseDTO, len(m.Exercises))
	for i, exItem := range m.Exercises {
		exerciseDTOs[i] = dto.MenuExerciseItemResponseDTO{
			ExerciseID: exItem.ExerciseID.String(),
			Name:       exItem.ExerciseName,
			Position:   exItem.Position,
		}
	}
	return dto.MenuWithExercisesResponseDTO{
		ID:          m.ID.String(),
		Name:        m.Name,
		Description: m.Description,
		SortOrder:   m.SortOrder,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		Exercises:   exerciseDTOs,
	}
}
