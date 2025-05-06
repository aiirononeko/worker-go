package command

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aiirononeko/bulktrack-api/internal/domain/entity"
	"github.com/aiirononeko/bulktrack-api/internal/domain/menu"
	"github.com/google/uuid"
)

// CreateMenuCommand represents the input data for creating a menu.
type CreateMenuCommand struct {
	DeviceID    string // ハンドラーから渡される (e.g., "device:uuid")
	Name        string
	Description *string
	SortOrder   int
}

// CreateMenuHandler handles the CreateMenuCommand.
type CreateMenuHandler struct {
	menuRepo menu.MenuRepository
}

// NewCreateMenuHandler creates a new CreateMenuHandler.
func NewCreateMenuHandler(menuRepo menu.MenuRepository) *CreateMenuHandler {
	return &CreateMenuHandler{menuRepo: menuRepo}
}

// Handle executes the create menu command.
func (h *CreateMenuHandler) Handle(ctx context.Context, cmd CreateMenuCommand) (*menu.Menu, error) {
	if cmd.DeviceID == "" {
		log.Printf("ERROR: Device ID is empty in CreateMenuCommand")
		return nil, NewErrValidation(fmt.Errorf("DeviceID cannot be empty"))
	}

	stringDeviceIDFromCmd := strings.TrimPrefix(cmd.DeviceID, "device:")
	if stringDeviceIDFromCmd == "" || stringDeviceIDFromCmd == cmd.DeviceID { // Check if prefix was not present or resulted in empty string
		log.Printf("ERROR: Invalid DeviceID format in CreateMenuCommand. Expected 'device:uuid', got '%s'", cmd.DeviceID)
		return nil, NewErrValidation(fmt.Errorf("invalid DeviceID format: %s", cmd.DeviceID))
	}

	// Convert string to entity.DeviceID
	domainDeviceID, err := entity.NewDeviceID(stringDeviceIDFromCmd)
	if err != nil {
		log.Printf("ERROR: Parsed pureDeviceID '%s' is not a valid UUID: %v", stringDeviceIDFromCmd, err)
		return nil, NewErrValidation(fmt.Errorf("parsed pureDeviceID '%s' is not a valid UUID: %w", stringDeviceIDFromCmd, err))
	}

	if strings.TrimSpace(cmd.Name) == "" {
		log.Printf("WARN: Menu name is empty for DeviceID: %s", domainDeviceID.String())
		return nil, NewErrValidation(fmt.Errorf("menu name cannot be empty"))
	}

	now := time.Now().UTC()
	newMenu := &menu.Menu{
		ID:          uuid.New(),
		DeviceID:    domainDeviceID,
		Name:        cmd.Name,
		Description: cmd.Description,
		SortOrder:   cmd.SortOrder,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err = h.menuRepo.Create(ctx, newMenu)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique constraint failed") ||
			strings.Contains(strings.ToLower(err.Error()), "duplicate entry") {
			log.Printf("WARN: Menu name '%s' likely conflicts for DeviceID %s: %v", cmd.Name, domainDeviceID.String(), err)
			return nil, &ErrMenuNameConflict{Name: cmd.Name}
		}
		log.Printf("ERROR: Failed to create menu '%s' in repository: %v", cmd.Name, err)
		return nil, fmt.Errorf("failed to create menu '%s': %w", cmd.Name, err)
	}

	log.Printf("INFO: CreateMenuHandler successfully created menu %s for device ID %s (original cmd.DeviceID: %s)", newMenu.ID, domainDeviceID.String(), cmd.DeviceID)
	return newMenu, nil
}
