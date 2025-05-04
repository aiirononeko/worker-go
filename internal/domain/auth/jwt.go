package auth

import (
	"context"
	"time"
)

// VerifiedTokenClaims は検証済みトークンから抽出されたクレームを保持します。
type VerifiedTokenClaims struct {
	UID string // ユーザー/デバイス識別子 ("device:uuid" または "user:uuid")
	JTI string // JWT ID (リフレッシュトークンの一意な識別子)
	// 必要に応じて他のクレーム (exp, iat など) も追加可能
}

// JWTService は JWT の生成と検証を抽象化するインターフェースです。
type JWTService interface {
	// GenerateTokens は指定された uid に対する新しいアクセストークンとリフレッシュトークンを生成します。
	// アクセストークンの有効期限 (expiresAt) と、生成された**リフレッシュトークンの jti** も返します。
	GenerateTokens(ctx context.Context, uid string) (accessToken string, refreshToken string, refreshTokenJTI string, expiresAt time.Time, err error)

	// VerifyToken は指定されたトークン文字列を検証し、含まれるクレーム情報を返します。
	// トークンが無効な場合や期限切れの場合はエラーを返します。
	VerifyToken(ctx context.Context, tokenString string) (*VerifiedTokenClaims, error)

	// TODO: 必要に応じて他のメソッド (例: RefreshToken の検証) を追加
}

// TODO: JWT関連のドメインエラー (例: ErrInvalidToken, ErrTokenExpired) を定義
