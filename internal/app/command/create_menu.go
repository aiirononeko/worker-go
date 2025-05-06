package command

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/aiirononeko/bulktrack-api/internal/app/apperror"
	"github.com/aiirononeko/bulktrack-api/internal/domain/entity"
	"github.com/aiirononeko/bulktrack-api/internal/domain/menu"
	// "github.com/google/uuid" // entity.NewMenuID() を使うので直接は不要になる
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
		slog.WarnContext(ctx, "Device ID is empty in CreateMenuCommand")
		return nil, apperror.NewErrBadRequest("DeviceID cannot be empty", "")
	}

	if !strings.HasPrefix(cmd.DeviceID, "device:") {
		slog.WarnContext(ctx, "Invalid DeviceID format in CreateMenuCommand: missing prefix", slog.String("raw_deviceID", cmd.DeviceID))
		return nil, apperror.NewErrBadRequest(fmt.Sprintf("Invalid DeviceID format: %s, must have 'device:' prefix", cmd.DeviceID), "")
	}
	stringDeviceIDFromCmd := strings.TrimPrefix(cmd.DeviceID, "device:")
	if stringDeviceIDFromCmd == "" {
		slog.WarnContext(ctx, "Invalid DeviceID format in CreateMenuCommand: empty after prefix", slog.String("raw_deviceID", cmd.DeviceID))
		return nil, apperror.NewErrBadRequest(fmt.Sprintf("Invalid DeviceID format: %s, empty after 'device:' prefix", cmd.DeviceID), "")
	}

	domainDeviceID, err := entity.NewDeviceID(stringDeviceIDFromCmd)
	if err != nil {
		slog.WarnContext(ctx, "Parsed pureDeviceID is not a valid UUID",
			slog.String("parsed_deviceID", stringDeviceIDFromCmd),
			slog.Any("original_error", err.Error()),
		)
		return nil, apperror.NewErrBadRequest(fmt.Sprintf("Invalid DeviceID UUID format: %s", stringDeviceIDFromCmd), err.Error())
	}

	if strings.TrimSpace(cmd.Name) == "" {
		slog.WarnContext(ctx, "Menu name is empty", slog.String("deviceID", domainDeviceID.String()))
		return nil, NewErrValidation(fmt.Errorf("Menu name cannot be empty"))
	}

	now := time.Now().UTC()
	newMenuID := entity.NewMenuID()

	newMenu := &menu.Menu{
		ID:          newMenuID,
		DeviceID:    domainDeviceID,
		Name:        cmd.Name,
		Description: cmd.Description,
		SortOrder:   cmd.SortOrder,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	slog.InfoContext(ctx, "Attempting to create menu in repository",
		slog.String("menuID", newMenu.ID.String()),
		slog.String("deviceID", newMenu.DeviceID.String()),
		slog.String("menuName", newMenu.Name),
	)
	err = h.menuRepo.Create(ctx, newMenu)
	if err != nil {
		var conflictErr *ErrMenuNameConflict
		if errors.As(err, &conflictErr) {
			slog.WarnContext(ctx, "Menu name conflicts",
				slog.String("menuName", cmd.Name),
				slog.String("deviceID", domainDeviceID.String()),
				slog.Any("error", err),
			)
			return nil, conflictErr
		} else if strings.Contains(strings.ToLower(err.Error()), "unique constraint failed") ||
			strings.Contains(strings.ToLower(err.Error()), "duplicate entry") {
			slog.WarnContext(ctx, "Menu name likely conflicts (fallback check)",
				slog.String("menuName", cmd.Name),
				slog.String("deviceID", domainDeviceID.String()),
				slog.Any("error", err),
			)
			return nil, &ErrMenuNameConflict{Name: cmd.Name}
		}

		slog.ErrorContext(ctx, "Failed to create menu in repository",
			slog.String("menuName", cmd.Name),
			slog.Any("original_error", err.Error()),
		)
		return nil, apperror.NewErrInternal(fmt.Sprintf("Failed to create menu '%s' in repository", cmd.Name), err)
	}

	slog.InfoContext(ctx, "Successfully created menu",
		slog.String("menuID", newMenu.ID.String()),
		slog.String("deviceID", domainDeviceID.String()),
		slog.String("original_cmd_deviceID", cmd.DeviceID),
	)
	return newMenu, nil
}
