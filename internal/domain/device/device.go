package device

import (
	"fmt"
	"time"

	"github.com/aiirononeko/bulktrack-api/internal/domain/entity"
)

// Device はユーザーのデバイスを表すドメインエンティティです。
type Device struct {
	ID         entity.DeviceID // Changed to entity.DeviceID
	UserID     *string         // 関連付けられたユーザーID (nullable)
	CreatedAt  time.Time       // 作成日時
	LastSeenAt time.Time       // 最終アクセス日時
}

// NewDevice は新しい Device エンティティを作成します。
// idValue は有効なUUID文字列である必要があります。
func NewDevice(idValue string, userID *string) (*Device, error) {
	deviceID, err := entity.NewDeviceID(idValue)
	if err != nil {
		return nil, fmt.Errorf("failed to create new device id: %w", err)
	}

	now := time.Now()
	return &Device{
		ID:         deviceID,
		UserID:     userID,
		CreatedAt:  now,
		LastSeenAt: now,
	}, nil
}

// UpdateLastSeen は最終アクセス日時を現在時刻に更新します。
func (d *Device) UpdateLastSeen() {
	d.LastSeenAt = time.Now()
}
