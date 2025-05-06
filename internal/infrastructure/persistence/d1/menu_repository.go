package d1

import (
	"context" // For sql.NullString & sql.OpenDB
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

		menusResult = append(menusResult, menu.Menu{
			ID:          menuID, // Use parsed entity.MenuID
			DeviceID:    rowDeviceID,
			Name:        row.Name,
			Description: description,
			SortOrder:   int(row.SortOrder),
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
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
