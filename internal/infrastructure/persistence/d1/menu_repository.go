package d1

import (
	"context" // For sql.NullString & sql.OpenDB
	"errors"

	// Need this for sql.DBTX if not aliased
	// For errors.Is and potentially future use
	"fmt"
	"log/slog" // For UNIQUE constraint check
	"time"     // For time parsing

	"github.com/aiirononeko/bulktrack-api/internal/app/apperror" // apperror をインポート
	"github.com/aiirononeko/bulktrack-api/internal/domain/entity"
	"github.com/aiirononeko/bulktrack-api/internal/domain/menu"

	// Import sqlc generated package from the correct path with alias 'db'
	"database/sql" // For sql.NullString

	db "github.com/aiirononeko/bulktrack-api/internal/infrastructure/persistence/d1/sql"
	// For UUID parsing
)

type menuRepository struct {
	db db.DBTX // Use alias
}

// NewMenuRepository は MenuRepository の新しいインスタンスを生成します。
// 引数は sqlc の Queries を生成する際に渡すものに合わせます。
func NewMenuRepository(database db.DBTX) menu.MenuRepository {
	return &menuRepository{db: database}
}

// SQLite uses "YYYY-MM-DD HH:MM:SS" by default for datetime('now')
const sqliteTimeFormat = "2006-01-02 15:04:05"

// ListMenusByDeviceId は指定されたデバイスIDのメニュー一覧を取得します。
func (r *menuRepository) ListMenusByDeviceId(ctx context.Context, deviceID entity.DeviceID) ([]menu.Menu, error) {
	q := db.New(r.db)
	slog.InfoContext(ctx, "Listing menus by device ID via sqlc", slog.String("deviceID", deviceID.String()))

	rows, err := q.ListMenusByDeviceId(ctx, deviceID.String())
	if err != nil {
		slog.ErrorContext(ctx, "Failed to execute sqlc ListMenusByDeviceId query",
			slog.String("deviceID", deviceID.String()),
			slog.Any("error", err),
		)
		// DB Query Error
		return nil, apperror.NewErrInternal(fmt.Sprintf("Failed to list menus for device %s from DB", deviceID.String()), err)
	}

	menusResult := make([]menu.Menu, 0, len(rows))
	for _, row := range rows {
		// Parse Menu ID
		menuID, err := entity.NewMenuIDFromString(row.ID)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to parse Menu ID UUID string from DB",
				slog.String("db_menu_id", row.ID),
				slog.String("deviceID", deviceID.String()),
				slog.Any("error", err),
			)
			// Return error instead of skipping
			return nil, apperror.NewErrInternal(fmt.Sprintf("Failed to parse menu ID '%s' from DB", row.ID), err)
		}

		// Parse Device ID
		rowDeviceID, err := entity.NewDeviceID(row.DeviceID)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to parse DeviceID string from DB for Menu", // Log as Error
				slog.String("db_deviceID", row.DeviceID),
				slog.String("db_menu_id", row.ID),
				slog.Any("error", err),
			)
			// Return error instead of skipping
			return nil, apperror.NewErrInternal(fmt.Sprintf("Failed to parse device ID '%s' from DB for menu %s", row.DeviceID, row.ID), err)
		}

		var description *string
		if row.Description.Valid {
			descStr := row.Description.String
			description = &descStr
		}

		// Parse CreatedAt
		createdAt, err := time.Parse(sqliteTimeFormat, row.CreatedAt)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to parse CreatedAt string from DB for Menu", // Log as Error
				slog.String("db_created_at", row.CreatedAt),
				slog.String("db_menu_id", row.ID),
				slog.String("deviceID", deviceID.String()),
				slog.Any("error", err),
			)
			// Return error instead of skipping
			return nil, apperror.NewErrInternal(fmt.Sprintf("Failed to parse timestamp for menu %s from DB", row.ID), err)
		}

		// Parse UpdatedAt
		updatedAt, err := time.Parse(sqliteTimeFormat, row.UpdatedAt)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to parse UpdatedAt string from DB for Menu", // Log as Error
				slog.String("db_updated_at", row.UpdatedAt),
				slog.String("db_menu_id", row.ID),
				slog.String("deviceID", deviceID.String()),
				slog.Any("error", err),
			)
			// Return error instead of skipping
			return nil, apperror.NewErrInternal(fmt.Sprintf("Failed to parse timestamp for menu %s from DB", row.ID), err)
		}

		// When listing menus, we don't typically fetch all their exercises unless specified.
		// The Exercises field will be empty here. It can be populated by a call to ListMenuExercises if needed.
		menusResult = append(menusResult, menu.Menu{
			ID:          menuID,
			DeviceID:    rowDeviceID,
			Name:        row.Name,
			Description: description,
			SortOrder:   int(row.SortOrder),
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
			Exercises:   nil, // Explicitly nil for list view
		})
	}
	slog.InfoContext(ctx, "Successfully listed menus by device ID via sqlc",
		slog.String("deviceID", deviceID.String()),
		slog.Int("menu_count", len(menusResult)),
	)
	return menusResult, nil
}

// Create は新しいメニューエンティティをデータベースに保存します。
func (r *menuRepository) Create(ctx context.Context, m *menu.Menu) error {
	q := db.New(r.db)
	slog.InfoContext(ctx, "Attempting to create menu via sqlc",
		slog.String("menuID", m.ID.String()),
		slog.String("deviceID", m.DeviceID.String()),
		slog.String("menuName", m.Name),
	)

	var desc sql.NullString
	if m.Description != nil {
		desc = sql.NullString{String: *m.Description, Valid: true}
	}

	params := db.CreateMenuParams{
		ID:          m.ID.String(),
		DeviceID:    m.DeviceID.String(),
		Name:        m.Name,
		Description: desc,
		SortOrder:   int64(m.SortOrder),
		CreatedAt:   m.CreatedAt.Format(sqliteTimeFormat),
		UpdatedAt:   m.UpdatedAt.Format(sqliteTimeFormat),
	}

	_, err := q.CreateMenu(ctx, params)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to execute sqlc CreateMenu query",
			slog.String("menuID", m.ID.String()),
			slog.String("deviceID", m.DeviceID.String()),
			slog.String("menuName", m.Name),
			slog.Any("error", err),
		)
		// Wrap DB error with ErrInternal, allowing App layer to inspect the cause if needed.
		// The App layer (command handler) will check for UNIQUE constraint based on the wrapped error.
		return apperror.NewErrInternal(fmt.Sprintf("Failed to create menu '%s' in DB", m.Name), err)
	}

	slog.InfoContext(ctx, "Successfully created menu via sqlc",
		slog.String("menuID", m.ID.String()),
		slog.String("deviceID", m.DeviceID.String()),
	)
	return nil
}

func (r *menuRepository) FindMenuByID(ctx context.Context, id entity.MenuID, deviceID entity.DeviceID) (*menu.Menu, error) {
	q := db.New(r.db)
	row, err := q.GetMenu(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.NewErrNotFound("Menu", id.String())
		}
		slog.ErrorContext(ctx, "Failed to execute sqlc GetMenu query", slog.String("menuID", id.String()), slog.Any("error", err))
		return nil, apperror.NewErrInternal(fmt.Sprintf("Failed to get menu %s from DB", id.String()), err)
	}

	if row.DeviceID != deviceID.String() {
		slog.WarnContext(ctx, "Menu found but does not belong to the requesting device", slog.String("menuID", id.String()), slog.String("menuDeviceID", row.DeviceID), slog.String("requestDeviceID", deviceID.String()))
		return nil, apperror.NewErrForbidden(fmt.Sprintf("Access to menu %s denied", id.String()))
	}

	menuID, err := entity.NewMenuIDFromString(row.ID)
	if err != nil { /* handle error */
		return nil, apperror.NewErrInternal("parsing menu id", err)
	}
	rowDeviceID, err := entity.NewDeviceID(row.DeviceID)
	if err != nil { /* handle error */
		return nil, apperror.NewErrInternal("parsing device id", err)
	}
	var description *string
	if row.Description.Valid {
		descStr := row.Description.String
		description = &descStr
	}
	createdAt, err := time.Parse(sqliteTimeFormat, row.CreatedAt)
	if err != nil { /* handle error */
		return nil, apperror.NewErrInternal("parsing created_at", err)
	}
	updatedAt, err := time.Parse(sqliteTimeFormat, row.UpdatedAt)
	if err != nil { /* handle error */
		return nil, apperror.NewErrInternal("parsing updated_at", err)
	}

	// Exercises are not populated here by default. Call ListMenuExercises separately if needed.
	return &menu.Menu{
		ID:          menuID,
		DeviceID:    rowDeviceID,
		Name:        row.Name,
		Description: description,
		SortOrder:   int(row.SortOrder),
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		Exercises:   nil, // Populate with ListMenuExercises if returning MenuWithExercises directly
	}, nil
}

func (r *menuRepository) UpdateMenuExercises(ctx context.Context, menuID entity.MenuID, deviceID entity.DeviceID, exercises []menu.MenuExerciseItem) error {
	q := db.New(r.db)

	_, err := r.FindMenuByID(ctx, menuID, deviceID)
	if err != nil {
		return err
	}

	err = q.DeleteMenuExercisesByMenuID(ctx, menuID.String())
	if err != nil {
		slog.ErrorContext(ctx, "Failed to delete existing exercises for menu", slog.String("menuID", menuID.String()), slog.Any("error", err))
		return apperror.NewErrInternal(fmt.Sprintf("failed to delete existing exercises for menu %s", menuID.String()), err)
	}

	for _, meItem := range exercises {
		params := db.CreateMenuExerciseParams{
			MenuID:     menuID.String(),
			ExerciseID: meItem.ExerciseID.String(),
			Position:   int64(meItem.Position),
		}
		err := q.CreateMenuExercise(ctx, params)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to create menu_exercise link", slog.String("menuID", menuID.String()), slog.String("exerciseID", meItem.ExerciseID.String()), slog.Any("error", err))
			return apperror.NewErrInternal(fmt.Sprintf("failed to create menu_exercise link for menu %s, exercise %s", menuID.String(), meItem.ExerciseID.String()), err)
		}
	}
	slog.InfoContext(ctx, "Successfully updated exercises for menu", slog.String("menuID", menuID.String()), slog.Int("num_exercises", len(exercises)))
	return nil
}

func (r *menuRepository) ListMenuExercises(ctx context.Context, menuID entity.MenuID) ([]menu.MenuExerciseItem, error) {
	q := db.New(r.db)
	dbMenuExerciseRows, err := q.ListMenuExercisesByMenuID(ctx, menuID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []menu.MenuExerciseItem{}, nil
		}
		slog.ErrorContext(ctx, "Failed to list exercises for menu from DB", slog.String("menuID", menuID.String()), slog.Any("error", err))
		return nil, apperror.NewErrInternal(fmt.Sprintf("failed to list exercises for menu %s from DB", menuID.String()), err)
	}

	domainItems := make([]menu.MenuExerciseItem, len(dbMenuExerciseRows))
	for i, dbRow := range dbMenuExerciseRows {
		exID, err := entity.NewExerciseIDFromString(dbRow.ExerciseID) // Assumes NewExerciseIDFromString exists
		if err != nil {
			slog.ErrorContext(ctx, "Invalid exercise ID from DB for menu_exercise", slog.String("dbExerciseID", dbRow.ExerciseID), slog.String("menuID", menuID.String()), slog.Any("error", err))
			return nil, apperror.NewErrInternal(fmt.Sprintf("invalid exercise ID '%s' from DB for menu %s", dbRow.ExerciseID, menuID.String()), err)
		}
		domainItems[i] = menu.MenuExerciseItem{
			ExerciseID:   exID,
			ExerciseName: dbRow.ExerciseName,
			Position:     int(dbRow.Position),
		}
	}
	slog.InfoContext(ctx, "Successfully listed exercises for menu", slog.String("menuID", menuID.String()), slog.Int("num_exercises", len(domainItems)))
	return domainItems, nil
}
