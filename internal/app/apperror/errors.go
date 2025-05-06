package apperror

import "fmt"

// ErrNotFound は、要求されたリソースが見つからなかったことを示すエラーです。
type ErrNotFound struct {
	Resource string
	ID       string
}

func (e *ErrNotFound) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("%s with ID '%s' not found", e.Resource, e.ID)
	}
	return fmt.Sprintf("%s not found", e.Resource)
}

// NewErrNotFound は新しい ErrNotFound エラーを生成します。
func NewErrNotFound(resource, id string) *ErrNotFound {
	return &ErrNotFound{Resource: resource, ID: id}
}

// ErrUnauthorized は、認証が必要だが提供されていない、または無効な認証情報であることを示すエラーです。
type ErrUnauthorized struct {
	Message string
}

func (e *ErrUnauthorized) Error() string {
	if e.Message == "" {
		return "unauthorized"
	}
	return e.Message
}

// NewErrUnauthorized は新しい ErrUnauthorized エラーを生成します。
func NewErrUnauthorized(message string) *ErrUnauthorized {
	return &ErrUnauthorized{Message: message}
}

// ErrForbidden は、認証は成功したが、要求されたリソースや操作に対する権限がないことを示すエラーです。
type ErrForbidden struct {
	Reason string
}

func (e *ErrForbidden) Error() string {
	if e.Reason == "" {
		return "forbidden"
	}
	return e.Reason
}

// NewErrForbidden は新しい ErrForbidden エラーを生成します。
func NewErrForbidden(reason string) *ErrForbidden {
	return &ErrForbidden{Reason: reason}
}

// ErrBadRequest は、クライアントのリクエストが無効であることを示すエラーです。
// 詳細なバリデーションエラーは command.ErrValidation を使用することを推奨します。
type ErrBadRequest struct {
	Message string
	Details string // バリデーションエラーなどの詳細
}

func (e *ErrBadRequest) Error() string {
	if e.Message == "" {
		return "bad request"
	}
	if e.Details != "" {
		return fmt.Sprintf("%s: %s", e.Message, e.Details)
	}
	return e.Message
}

// NewErrBadRequest は新しい ErrBadRequest エラーを生成します。
func NewErrBadRequest(message string, details string) *ErrBadRequest {
	return &ErrBadRequest{Message: message, Details: details}
}

// ErrInternal は、予期しないサーバー内部のエラーを示すエラーです。
// 通常、クライアントに詳細を公開すべきではないエラーに使用されます。
type ErrInternal struct {
	Message string // ログや内部追跡用のメッセージ
	Err     error  // 元となったエラー (nil の場合もある)
}

func (e *ErrInternal) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("internal server error: %s (caused by: %v)", e.Message, e.Err)
	}
	return fmt.Sprintf("internal server error: %s", e.Message)
}

// Unwrap は、ラップされたエラーを返すことで errors.Is や errors.As との互換性を提供します。
func (e *ErrInternal) Unwrap() error {
	return e.Err
}

// NewErrInternal は新しい ErrInternal エラーを生成します。
func NewErrInternal(message string, cause error) *ErrInternal {
	return &ErrInternal{Message: message, Err: cause}
}
