package command

import "fmt"

// ErrValidation は、一般的なバリデーション失敗を表すエラーです。
// 詳細なバリデーションメッセージはラップされたエラーに含まれることを期待します。
type ErrValidation struct {
	Wrapped error
}

func (e *ErrValidation) Error() string {
	if e.Wrapped != nil {
		return fmt.Sprintf("validation failed: %v", e.Wrapped)
	}
	return "validation failed"
}

func (e *ErrValidation) Unwrap() error {
	return e.Wrapped
}

// NewErrValidation は新しい ErrValidation を生成します。
func NewErrValidation(err error) error {
	return &ErrValidation{Wrapped: err}
}

// ErrMenuNameConflict は、作成しようとしたメニュー名が既に存在する場合のエラーです。
type ErrMenuNameConflict struct {
	Name string
}

func (e *ErrMenuNameConflict) Error() string {
	return fmt.Sprintf("menu name '%s' already exists", e.Name)
}

// ErrResourceNotFound は、リソースが見つからない場合のエラーです。
// (今回は直接使わないかもしれませんが、汎用的なエラーとして定義しておくと便利です)
type ErrResourceNotFound struct {
	Resource string
	ID       string
}

func (e *ErrResourceNotFound) Error() string {
	return fmt.Sprintf("%s with ID '%s' not found", e.Resource, e.ID)
}
