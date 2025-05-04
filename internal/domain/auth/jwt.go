package auth

import (
	"context"
	"time"
)

// JWTService は JWT の生成と検証を抽象化するインターフェースです。
type JWTService interface {
	// GenerateTokens は指定された uid に対する新しいアクセストークンとリフレッシュトークンを生成します。
	// アクセストークンの有効期限 (expiresAt) も返します。
	GenerateTokens(ctx context.Context, uid string) (accessToken string, refreshToken string, expiresAt time.Time, err error)

	// VerifyToken は指定されたトークン文字列を検証し、含まれる uid を返します。
	// トークンが無効な場合や期限切れの場合はエラーを返します。
	// これは主に Auth ミドルウェアで使用される想定です。
	VerifyToken(ctx context.Context, tokenString string) (uid string, err error)

	// TODO: 必要に応じて他のメソッド (例: RefreshToken の検証) を追加
}

// TODO: JWT関連のドメインエラー (例: ErrInvalidToken, ErrTokenExpired) を定義
