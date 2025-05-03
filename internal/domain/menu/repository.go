package menu

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Menu はドメインエンティティを表します。
type Menu struct {
	ID          uuid.UUID
	UserID      string // ドメイン層では userID が誰かは関知しない
	Name        string
	Description *string // ポインタ or sql.NullString に合わせた型が良いかも
	SortOrder   int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// MenuRepository はメニューデータの永続化を抽象化するインターフェースです。
type MenuRepository interface {
	// ListMenusByUserId は指定されたユーザーIDのメニュー一覧をソート順で取得します。
	ListMenusByUserId(ctx context.Context, userID string) ([]Menu, error)
}
