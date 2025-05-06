package d1

import (
	"context" // For sql.NullString & sql.OpenDB
	// Need this for sql.DBTX if not aliased
	"log"  // For error logging during conversion
	"time" // For time parsing

	"github.com/aiirononeko/bulktrack-api/internal/domain/menu"
	// Import sqlc generated package from the correct path with alias 'db'
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
func (r *menuRepository) ListMenusByDeviceId(ctx context.Context, deviceID string) ([]menu.Menu, error) {
	q := db.New(r.db)

	// sqlc の生成コードを呼び出す (メソッド名を ListMenusByDeviceId に変更)
	rows, err := q.ListMenusByDeviceId(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	menus := make([]menu.Menu, 0, len(rows))
	for _, row := range rows {
		id, err := uuid.Parse(row.ID)
		if err != nil {
			log.Printf("Error parsing UUID string '%s': %v", row.ID, err)
			continue
		}

		var description *string
		if row.Description.Valid {
			descStr := row.Description.String
			description = &descStr
		}

		createdAt, err := time.Parse(sqliteTimeFormat, row.CreatedAt)
		if err != nil {
			log.Printf("Error parsing CreatedAt string '%s': %v", row.CreatedAt, err)
			continue
		}

		updatedAt, err := time.Parse(sqliteTimeFormat, row.UpdatedAt)
		if err != nil {
			log.Printf("Error parsing UpdatedAt string '%s': %v", row.UpdatedAt, err)
			continue
		}

		menus = append(menus, menu.Menu{
			ID:          id,
			DeviceID:    row.DeviceID, // UserID から DeviceID に変更し、row から取得
			Name:        row.Name,
			Description: description,
			SortOrder:   int(row.SortOrder),
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		})
	}

	return menus, nil
}
