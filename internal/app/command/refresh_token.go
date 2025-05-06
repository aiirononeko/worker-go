package command

import (
	"context"
	"fmt"
	"log"
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
	log.Printf("INFO: Starting token refresh process.") // リフレッシュトークン自体はログに出さない

	if cmd.RefreshToken == "" {
		log.Printf("ERROR: Refresh token is empty in RefreshTokenCommand")
		return nil, fmt.Errorf("refresh token is required")
	}

	log.Printf("INFO: Verifying provided refresh token.")
	claims, err := h.jwtService.VerifyToken(ctx, cmd.RefreshToken)
	if err != nil {
		log.Printf("ERROR: Failed to verify refresh token: %v", err)
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}
	if claims.JTI == "" {
		log.Printf("ERROR: Invalid refresh token: missing JTI claim. UID from token: %s", claims.UID)
		return nil, fmt.Errorf("invalid refresh token: missing jti claim")
	}
	log.Printf("INFO: Refresh token verified. JTI: %s, UID: %s", claims.JTI, claims.UID)

	log.Printf("INFO: Validating refresh token (JTI: %s) in KV store for UID: %s", claims.JTI, claims.UID)
	if err := h.refreshTokenRepo.Validate(ctx, claims.JTI, claims.UID); err != nil {
		log.Printf("ERROR: Refresh token validation failed in KV store for JTI %s, UID %s: %v", claims.JTI, claims.UID, err)
		return nil, fmt.Errorf("refresh token validation failed: %w", err)
	}

	log.Printf("INFO: Deleting old refresh token (JTI: %s) from KV store.", claims.JTI)
	if err := h.refreshTokenRepo.Delete(ctx, claims.JTI); err != nil {
		// 削除失敗は警告ログに留めるが、処理は続行する (トークン回転の主要な目的は新しいトークンの発行)
		log.Printf("WARN: Failed to delete old refresh token (JTI: %s) during rotation, but proceeding: %v", claims.JTI, err)
	}

	log.Printf("INFO: Generating new token pair for UID: %s", claims.UID)
	newAccessToken, newRefreshToken, newJTI, expiresAt, err := h.jwtService.GenerateTokens(ctx, claims.UID)
	if err != nil {
		log.Printf("ERROR: Failed to generate new tokens for UID %s: %v", claims.UID, err)
		return nil, fmt.Errorf("failed to generate new tokens: %w", err)
	}

	log.Printf("INFO: Saving new refresh token to KV for new JTI: %s, UID: %s", newJTI, claims.UID)
	if err := h.refreshTokenRepo.Save(ctx, newJTI, claims.UID, h.refreshTokenTTL); err != nil {
		log.Printf("ERROR: Failed to save new refresh token to KV for new JTI %s, UID %s: %v", newJTI, claims.UID, err)
		return nil, fmt.Errorf("failed to save new refresh token: %w", err)
	}

	log.Printf("INFO: Successfully refreshed tokens for UID: %s. New JTI: %s", claims.UID, newJTI)
	return &RefreshTokenResult{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}
