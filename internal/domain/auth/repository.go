package auth

import (
	"context"
	"errors"
	"time"
)

// --- Domain Errors ---

var (
	// ErrRefreshTokenNotFound はリフレッシュトークンがストアに見つからない場合に返されます。
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	// ErrRefreshTokenExpired はリフレッシュトークンが期限切れの場合に返されます。
	// (KVのTTLで管理されるため、通常は NotFound となることが多いが、明示的に返すケースも考慮)
	ErrRefreshTokenExpired = errors.New("refresh token expired")
	// ErrMismatchedUID はリフレッシュトークンに関連付けられたUIDが期待値と異なる場合に返されます。
	ErrMismatchedUID = errors.New("mismatched user/device ID for refresh token")
	// ErrRefreshTokenRevoked はリフレッシュトークンが失効済みの場合に返されます (ログアウトなど)。
	// (失効リストを別途管理する場合や、値を特殊な状態にする場合)
	ErrRefreshTokenRevoked = errors.New("refresh token has been revoked")
)

// RefreshTokenRepository はリフレッシュトークンの永続化 (KVストア) を抽象化します。
type RefreshTokenRepository interface {
	// Save は新しいリフレッシュトークン情報 (jti をキーに uid と TTL) を保存します。
	// 既に存在する jti の場合は上書き、またはエラーとすることが考えられます (実装依存)。
	Save(ctx context.Context, jti string, uid string, ttl time.Duration) error

	// Validate は指定された jti が有効か検証します。
	// 検証項目:
	// 1. jti がストアに存在するか (存在しない -> ErrRefreshTokenNotFound)
	// 2. ストアされた uid が expectedUID と一致するか (不一致 -> ErrMismatchedUID)
	// 3. (オプション) 明示的に失効されていないか (失効 -> ErrRefreshTokenRevoked)
	// KVのTTLで有効期限は管理されるため、通常は期限切れ = NotFound となる。
	Validate(ctx context.Context, jti string, expectedUID string) error

	// Delete は指定された jti のリフレッシュトークン情報を削除します。
	// 存在しない jti に対して呼び出された場合でもエラーを返さないことが望ましい (冪等性)。
	Delete(ctx context.Context, jti string) error

	// TODO: 失効リストを別途管理する場合はメソッド追加 (MarkRevoked など)
}
