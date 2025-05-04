package command

import (
	"context"
	"fmt"
	"time"

	domainAuth "github.com/aiirononeko/bulktrack-api/internal/domain/auth"
	// TODO: Refresh Token Repository (KV) の Domain IF を import
)

// RefreshTokenCommand はリフレッシュトークン処理の入力データを保持します。
type RefreshTokenCommand struct {
	RefreshToken string
}

// RefreshTokenResult はリフレッシュトークン処理の結果を保持します。
type RefreshTokenResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time // Access token の有効期限
}

// RefreshTokenHandler はリフレッシュトークンのユースケースを実行します。
type RefreshTokenHandler interface {
	Handle(ctx context.Context, cmd RefreshTokenCommand) (*RefreshTokenResult, error)
}

// --- Implementation ---

type refreshTokenHandler struct {
	jwtService       domainAuth.JWTService
	refreshTokenRepo domainAuth.RefreshTokenRepository // リポジトリへの依存を追加
	refreshTokenTTL  time.Duration                     // リフレッシュトークンの TTL (設定から取得)
}

// NewRefreshTokenHandler は新しい refreshTokenHandler を初期化します。
func NewRefreshTokenHandler(jwtService domainAuth.JWTService, refreshTokenRepo domainAuth.RefreshTokenRepository, refreshTokenTTL time.Duration) RefreshTokenHandler {
	return &refreshTokenHandler{
		jwtService:       jwtService,
		refreshTokenRepo: refreshTokenRepo,
		refreshTokenTTL:  refreshTokenTTL, // TTL を保持
	}
}

func (h *refreshTokenHandler) Handle(ctx context.Context, cmd RefreshTokenCommand) (*RefreshTokenResult, error) {
	// log.Printf("[RefreshTokenHandler] Starting refresh...") // ログ削除

	// 1. Refresh Token を検証し、クレームを取得
	claims, err := h.jwtService.VerifyToken(ctx, cmd.RefreshToken)
	if err != nil {
		// log.Printf("[RefreshTokenHandler] Error verifying token: %v", err) // ログ削除
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}
	if claims.JTI == "" {
		// log.Printf("[RefreshTokenHandler] Invalid token: missing jti claim...") // ログ削除
		return nil, fmt.Errorf("invalid refresh token: missing jti claim")
	}
	// log.Printf("[RefreshTokenHandler] Token verified...") // ログ削除

	// 2. KV ストアでトークンの有効性を検証
	if err := h.refreshTokenRepo.Validate(ctx, claims.JTI, claims.UID); err != nil {
		// log.Printf("[RefreshTokenHandler] Error validating token in KV...") // ログ削除
		return nil, fmt.Errorf("refresh token validation failed: %w", err)
	}
	// log.Printf("[RefreshTokenHandler] Token validated in KV...") // ログ削除

	// --- トークン回転 ---

	// 3. 古いリフレッシュトークンを KV から削除
	if err := h.refreshTokenRepo.Delete(ctx, claims.JTI); err != nil {
		// 削除失敗は警告ログに留める (本番ではより詳細なロギング/監視を推奨)
		fmt.Printf("Warning: failed to delete old refresh token (jti: %s) during rotation: %v\n", claims.JTI, err)
	}

	// 4. 新しいトークンペアを生成
	newAccessToken, newRefreshToken, newJTI, expiresAt, err := h.jwtService.GenerateTokens(ctx, claims.UID)
	if err != nil {
		// log.Printf("[RefreshTokenHandler] Error generating new tokens...") // ログ削除
		return nil, fmt.Errorf("failed to generate new tokens: %w", err)
	}
	// log.Printf("[RefreshTokenHandler] New tokens generated...") // ログ削除

	// 5. 新しいリフレッシュトークン情報を KV に保存
	if err := h.refreshTokenRepo.Save(ctx, newJTI, claims.UID, h.refreshTokenTTL); err != nil {
		// log.Printf("[RefreshTokenHandler] Error saving new refresh token to KV...") // ログ削除
		return nil, fmt.Errorf("failed to save new refresh token: %w", err)
	}
	// log.Printf("[RefreshTokenHandler] New refresh token saved to KV...") // ログ削除

	// 6. 結果を返す
	// log.Printf("[RefreshTokenHandler] Refresh successful...") // ログ削除
	return &RefreshTokenResult{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}
