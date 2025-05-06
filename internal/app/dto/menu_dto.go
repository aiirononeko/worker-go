package dto

import (
	"time"
)

// MenuDTO はメニューデータのレスポンス表現です。
type MenuDTO struct {
	ID          string    `json:"id"` // uuid.UUID から string に変更
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"` // omitempty で NULL の場合はキー自体を省略
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
