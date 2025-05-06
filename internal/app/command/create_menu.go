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
func (h *CreateMenuHandler) Handle(ctx context.Context, cmd CreateMenuCommand) (*menu.Menu, error) {
	if cmd.DeviceID == "" {
		log.Printf("ERROR: Device ID is empty in CreateMenuCommand")
		return nil, NewErrValidation(fmt.Errorf("DeviceID cannot be empty"))
	}

	pureDeviceID := strings.TrimPrefix(cmd.DeviceID, "device:")
	if pureDeviceID == "" || pureDeviceID == cmd.DeviceID { // Check if prefix was not present or resulted in empty string
		log.Printf("ERROR: Invalid DeviceID format in CreateMenuCommand. Expected 'device:uuid', got '%s'", cmd.DeviceID)
		return nil, NewErrValidation(fmt.Errorf("invalid DeviceID format: %s", cmd.DeviceID))
	}

	if _, err := uuid.Parse(pureDeviceID); err != nil {
		log.Printf("ERROR: Parsed pureDeviceID '%s' is not a valid UUID: %v", pureDeviceID, err)
		return nil, NewErrValidation(fmt.Errorf("parsed pureDeviceID '%s' is not a valid UUID: %w", pureDeviceID, err))
	}

	if strings.TrimSpace(cmd.Name) == "" {
		log.Printf("WARN: Menu name is empty for DeviceID: %s", pureDeviceID)
		return nil, NewErrValidation(fmt.Errorf("menu name cannot be empty"))
	}

	now := time.Now().UTC()
	newMenu := &menu.Menu{
		ID:          uuid.New(),
		DeviceID:    pureDeviceID,
		Name:        cmd.Name,
		Description: cmd.Description,
		SortOrder:   cmd.SortOrder,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err := h.menuRepo.Create(ctx, newMenu)
	if err != nil {
		// Attempt to check for a unique constraint violation (this is DB-specific and fragile)
		// For D1 (SQLite), a common way unique constraint errors appear is with "UNIQUE constraint failed"
		// or via specific error codes if the driver provides them.
		// sqlc might wrap these, so direct checking can be hard.
		// A more robust solution involves the repository layer identifying and returning specific domain errors.
		if strings.Contains(strings.ToLower(err.Error()), "unique constraint failed") ||
			strings.Contains(strings.ToLower(err.Error()), "duplicate entry") { // A common SQL error text
			log.Printf("WARN: Menu name '%s' likely conflicts for DeviceID %s: %v", cmd.Name, pureDeviceID, err)
			return nil, &ErrMenuNameConflict{Name: cmd.Name}
		}

		// Check for other known database errors if possible, e.g., foreign key issues not caught earlier.
		// var sqliteErr *sqlite.Error // Example: if using a specific sqlite driver and error type
		// if errors.As(err, &sqliteErr) {
		// 	if sqliteErr.Code() == lib.SQLITE_CONSTRAINT_UNIQUE {
		// 		return nil, &ErrMenuNameConflict{Name: cmd.Name}
		// 	}
		// }

		log.Printf("ERROR: Failed to create menu '%s' in repository: %v", cmd.Name, err)
		return nil, fmt.Errorf("failed to create menu '%s': %w", cmd.Name, err) // Generic internal error
	}

	log.Printf("INFO: CreateMenuHandler successfully created menu %s for pure device ID %s (original cmd.DeviceID: %s)", newMenu.ID, pureDeviceID, cmd.DeviceID)
	return newMenu, nil
}
