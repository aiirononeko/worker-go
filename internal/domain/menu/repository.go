package menu

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Menu はドメインエンティティを表します。
type Menu struct {
	ID          uuid.UUID
	DeviceID    string
	Name        string
	Description *string // ポインタ or sql.NullString に合わせた型が良いかも
	SortOrder   int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// MenuRepository はメニューデータの永続化を抽象化するインターフェースです。
type MenuRepository interface {
	// ListMenusByDeviceId は指定されたデバイスIDのメニュー一覧をソート順で取得します。
	ListMenusByDeviceId(ctx context.Context, deviceID string) ([]Menu, error)
}
