package device

import "time"

// Device はユーザーのデバイスを表すドメインエンティティです。
type Device struct {
	ID         string    // デバイスID (UUID)
	UserID     *string   // 関連付けられたユーザーID (nullable)
	CreatedAt  time.Time // 作成日時
	LastSeenAt time.Time // 最終アクセス日時
}

// NewDevice は新しい Device エンティティを作成します。
// 現時点ではシンプルな初期化のみ。
// TODO: ID のバリデーション (UUID形式など) を追加する可能性あり。
func NewDevice(id string, userID *string) *Device {
	now := time.Now()
	return &Device{
		ID:         id,
		UserID:     userID,
		CreatedAt:  now,
		LastSeenAt: now,
	}
}

// UpdateLastSeen は最終アクセス日時を現在時刻に更新します。
func (d *Device) UpdateLastSeen() {
	d.LastSeenAt = time.Now()
}
