package d1

import (
	"context" // For sql.NullString & sql.OpenDB
	// Need this for sql.DBTX if not aliased
	"fmt"
	"log"  // For error logging during conversion
	"time" // For time parsing

	"github.com/aiirononeko/bulktrack-api/internal/domain/entity"
	"github.com/aiirononeko/bulktrack-api/internal/domain/menu"

	// Import sqlc generated package from the correct path with alias 'db'
	"database/sql" // For sql.NullString

	db "github.com/aiirononeko/bulktrack-api/internal/infrastructure/persistence/d1/sql"
	"github.com/google/uuid" // For UUID parsing
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

	rows, err := q.ListMenusByDeviceId(ctx, deviceID.String()) // Use .String()
	if err != nil {
		// D1クエリ実行時のエラーをログに出力
		log.Printf("ERROR: Failed to execute ListMenusByDeviceId query for DeviceID %s: %v", deviceID.String(), err)
		return nil, fmt.Errorf("d1 query ListMenusByDeviceId for device %s failed: %w", deviceID.String(), err) // エラーをラップ
	}

	menusResult := make([]menu.Menu, 0, len(rows))
	for _, row := range rows {
		id, err := uuid.Parse(row.ID)
		if err != nil {
			log.Printf("WARN: Failed to parse Menu ID UUID string '%s' for DeviceID %s: %v", row.ID, deviceID.String(), err)
			continue // この行をスキップ
		}

		rowDeviceID, err := entity.NewDeviceID(row.DeviceID) // Convert string from DB to entity.DeviceID
		if err != nil {
			log.Printf("WARN: Failed to parse DeviceID string '%s' from DB for Menu ID %s: %v", row.DeviceID, row.ID, err)
			continue
		}

		var description *string
		if row.Description.Valid {
			descStr := row.Description.String
			description = &descStr
		}

		createdAt, err := time.Parse(sqliteTimeFormat, row.CreatedAt)
		if err != nil {
			log.Printf("WARN: Failed to parse CreatedAt string '%s' for Menu ID %s, DeviceID %s: %v", row.CreatedAt, row.ID, deviceID.String(), err)
			continue
		}

		updatedAt, err := time.Parse(sqliteTimeFormat, row.UpdatedAt)
		if err != nil {
			log.Printf("WARN: Failed to parse UpdatedAt string '%s' for Menu ID %s, DeviceID %s: %v", row.UpdatedAt, row.ID, deviceID.String(), err)
			continue
		}

		menusResult = append(menusResult, menu.Menu{
			ID:          id,
			DeviceID:    rowDeviceID, // Assign converted entity.DeviceID
			Name:        row.Name,
			Description: description,
			SortOrder:   int(row.SortOrder),
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		})
	}
	log.Printf("INFO: MenuRepository.ListMenusByDeviceId for DeviceID %s found %d menus.", deviceID.String(), len(menusResult))
	return menusResult, nil
}

// Create は新しいメニューエンティティをデータベースに保存します。
func (r *menuRepository) Create(ctx context.Context, m *menu.Menu) error {
	q := db.New(r.db)

	var desc sql.NullString
	if m.Description != nil {
		desc = sql.NullString{String: *m.Description, Valid: true}
	}

	params := db.CreateMenuParams{
		ID:          m.ID.String(),
		DeviceID:    m.DeviceID.String(), // Use .String()
		Name:        m.Name,
		Description: desc,
		SortOrder:   int64(m.SortOrder),
		CreatedAt:   m.CreatedAt.Format(sqliteTimeFormat),
		UpdatedAt:   m.UpdatedAt.Format(sqliteTimeFormat),
	}

	_, err := q.CreateMenu(ctx, params)
	if err != nil {
		log.Printf("ERROR: Failed to execute CreateMenu query for Menu ID %s, DeviceID %s: %v", m.ID.String(), m.DeviceID.String(), err)
		return fmt.Errorf("d1 query CreateMenu failed for menu %s: %w", m.Name, err)
	}

	log.Printf("INFO: MenuRepository.Create succeeded for Menu ID %s, DeviceID %s.", m.ID.String(), m.DeviceID.String())
	return nil
}
