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

func (r *menuRepository) ListMenusByUserId(ctx context.Context, userID string) ([]menu.Menu, error) {
	q := db.New(r.db) // Use alias

	rows, err := q.ListMenusByUserId(ctx, userID) // sqlc の生成コードを呼び出す
	if err != nil {
		// エラーログなどをここに追加する
		return nil, err // TODO: エラーをラップしてドメイン層/アプリ層向けのエラーにするのが望ましい
	}

	// sqlc の結果 (Row) をドメインオブジェクトに変換
	menus := make([]menu.Menu, 0, len(rows))
	for _, row := range rows {
		// Convert ID (string) to uuid.UUID
		id, err := uuid.Parse(row.ID)
		if err != nil {
			log.Printf("Error parsing UUID string '%s': %v", row.ID, err)
			// Decide how to handle error: skip row, return error, etc.
			// For now, let's skip this row.
			continue
		}

		// Handle nullable description
		var description *string
		if row.Description.Valid {
			descStr := row.Description.String
			description = &descStr
		}

		// Convert CreatedAt (string) to time.Time
		createdAt, err := time.Parse(sqliteTimeFormat, row.CreatedAt)
		if err != nil {
			log.Printf("Error parsing CreatedAt string '%s': %v", row.CreatedAt, err)
			continue // Skip row on parsing error
		}

		// Convert UpdatedAt (string) to time.Time
		updatedAt, err := time.Parse(sqliteTimeFormat, row.UpdatedAt)
		if err != nil {
			log.Printf("Error parsing UpdatedAt string '%s': %v", row.UpdatedAt, err)
			continue // Skip row on parsing error
		}

		menus = append(menus, menu.Menu{
			ID:          id,     // Use parsed UUID
			UserID:      userID, // user_id は引数から取得。Rowには含まれない場合もある
			Name:        row.Name,
			Description: description,
			SortOrder:   int(row.SortOrder), // 型変換
			CreatedAt:   createdAt,          // Use parsed time.Time
			UpdatedAt:   updatedAt,          // Use parsed time.Time
		})
	}

	return menus, nil
}
