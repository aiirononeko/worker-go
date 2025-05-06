package entity

import (
	"fmt"

	"github.com/google/uuid"
)

// MenuID はメニューの一意な識別子を表します。
type MenuID uuid.UUID

// NewMenuID は新しい MenuID (UUID v4) を生成します。
func NewMenuID() MenuID {
	return MenuID(uuid.New())
}

// NewMenuIDFromString は文字列から MenuID を生成します。
// 文字列が無効なUUID形式の場合はエラーを返します。
func NewMenuIDFromString(id string) (MenuID, error) {
	u, err := uuid.Parse(id)
	if err != nil {
		return MenuID(uuid.Nil), fmt.Errorf("invalid menu ID format '%s': %w", id, err)
	}
	return MenuID(u), nil
}

// String は MenuID を文字列として返します。
func (mid MenuID) String() string {
	return uuid.UUID(mid).String()
}

// Equals は別の MenuID と等しいかどうかを比較します。
func (mid MenuID) Equals(other MenuID) bool {
	return uuid.UUID(mid) == uuid.UUID(other)
}

// IsNil は MenuID がnil（ゼロ値のUUID）であるかどうかを確認します。
func (mid MenuID) IsNil() bool {
	return uuid.UUID(mid) == uuid.Nil
}
