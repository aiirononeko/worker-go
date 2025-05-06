package dto

import (
	"time"
)

// MenuDTO はメニューデータのレスポンス表現です。
type MenuDTO struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"` // omitempty で NULL の場合はキー自体を省略
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// --- For PUT /v1/menus/{menuId}/exercises ---

// UpdateMenuExerciseItemDTO represents a single exercise item in the request to update menu exercises.
type UpdateMenuExerciseItemDTO struct {
	ExerciseID string `json:"exercise_id" validate:"required,uuid"`
	Position   int    `json:"position" validate:"gte=0"` // 0-indexed
}

// UpdateMenuExercisesRequestDTO is the request body for updating exercises associated with a menu.
type UpdateMenuExercisesRequestDTO struct {
	Exercises []UpdateMenuExerciseItemDTO `json:"exercises" validate:"dive"` // dive to validate each item in the slice
}

// MenuExerciseItemResponseDTO represents a single exercise item in a menu response.
type MenuExerciseItemResponseDTO struct {
	ExerciseID string `json:"exercise_id"`
	Name       string `json:"name"`
	Position   int    `json:"position"`
}

// MenuWithExercisesResponseDTO represents a menu along with its associated exercises.
type MenuWithExercisesResponseDTO struct {
	ID          string                        `json:"id"`
	Name        string                        `json:"name"`
	Description *string                       `json:"description,omitempty"`
	SortOrder   int                           `json:"sort_order"`
	CreatedAt   time.Time                     `json:"created_at"`
	UpdatedAt   time.Time                     `json:"updated_at"`
	Exercises   []MenuExerciseItemResponseDTO `json:"exercises"`
}

// CreateMenuRequest は新しいメニューを作成する際のリクエストボディです。
type CreateMenuRequest struct {
	Name        string  `json:"name" validate:"required,min=1,max=100"`
	Description *string `json:"description,omitempty" validate:"max=1000"`
	SortOrder   int     `json:"sort_order" validate:"gte=0"`
}
