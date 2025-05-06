package entity

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// DeviceID はデバイスの一意な識別子を表す値オブジェクトです。
// 内部的にはUUID文字列を保持します。
type DeviceID string

// NewDeviceID は文字列から新しい DeviceID を生成します。
// value は有効なUUID文字列である必要があります。
func NewDeviceID(value string) (DeviceID, error) {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return "", fmt.Errorf("device id cannot be empty")
	}
	parsedUUID, err := uuid.Parse(trimmedValue)
	if err != nil {
		return "", fmt.Errorf("invalid device id format (must be a UUID): %w", err)
	}
	// 常に小文字のUUIDとして保存するなどの正規化もここで行える
	return DeviceID(parsedUUID.String()), nil
}

// String は DeviceID の文字列表現 (UUID) を返します。
func (id DeviceID) String() string {
	return string(id)
}

// IsZero は DeviceID が空または未初期化であるかを返します。
func (id DeviceID) IsZero() bool {
	return string(id) == ""
}
