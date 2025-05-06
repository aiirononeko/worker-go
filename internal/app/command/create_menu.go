package command

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

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
// context はリポジトリ呼び出しのために残すが、deviceID取得には使わない
func (h *CreateMenuHandler) Handle(ctx context.Context, cmd CreateMenuCommand) (*menu.Menu, error) {
	if cmd.DeviceID == "" {
		log.Printf("ERROR: Device ID is empty in CreateMenuCommand")
		return nil, fmt.Errorf("invalid argument: DeviceID cannot be empty")
	}

	// Trim the "device:" prefix from DeviceID to get the pure UUID
	pureDeviceID := strings.TrimPrefix(cmd.DeviceID, "device:")
	if pureDeviceID == "" || pureDeviceID == cmd.DeviceID { // Check if prefix was not present or resulted in empty string
		log.Printf("ERROR: Invalid DeviceID format in CreateMenuCommand. Expected 'device:uuid', got '%s'", cmd.DeviceID)
		return nil, fmt.Errorf("invalid device ID format in command: '%s'", cmd.DeviceID)
	}

	// Validate if the pureDeviceID is a valid UUID (optional but good practice)
	if _, err := uuid.Parse(pureDeviceID); err != nil {
		log.Printf("ERROR: Parsed pureDeviceID '%s' is not a valid UUID: %v", pureDeviceID, err)
		return nil, fmt.Errorf("parsed pureDeviceID is not a valid UUID: %w", err)
	}

	now := time.Now().UTC()
	newMenu := &menu.Menu{
		ID:          uuid.New(),
		DeviceID:    pureDeviceID, // Use the pure UUID
		Name:        cmd.Name,
		Description: cmd.Description,
		SortOrder:   cmd.SortOrder,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err := h.menuRepo.Create(ctx, newMenu)
	if err != nil {
		log.Printf("ERROR: Failed to create menu in repository: %v", err)
		return nil, fmt.Errorf("failed to create menu: %w", err)
	}

	log.Printf("INFO: CreateMenuHandler successfully created menu %s for pure device ID %s (original cmd.DeviceID: %s)", newMenu.ID, pureDeviceID, cmd.DeviceID)
	return newMenu, nil
}
