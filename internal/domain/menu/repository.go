package menu

import (
	"context"
	"time"

	"github.com/aiirononeko/bulktrack-api/internal/domain/entity"
)

// Menu はドメインエンティティを表します。
type Menu struct {
	ID          entity.MenuID
	DeviceID    entity.DeviceID
	Name        string
	Description *string // ポインタ or sql.NullString に合わせた型が良いかも
	SortOrder   int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// MenuRepository はメニューデータの永続化を抽象化するインターフェースです。
//
//go:generate mockery --name MenuRepository --output ./mocks --inpackage
type MenuRepository interface {
	// ListMenusByDeviceId は指定されたデバイスIDのメニュー一覧をソート順で取得します。
	ListMenusByDeviceId(ctx context.Context, deviceID entity.DeviceID) ([]Menu, error)

	// Create は新しいメニューエンティティを永続化層に保存します。
	Create(ctx context.Context, menu *Menu) error
}
