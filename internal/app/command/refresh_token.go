package command

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aiirononeko/bulktrack-api/internal/app/apperror"
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
	slog.InfoContext(ctx, "Starting token refresh process.")

	if cmd.RefreshToken == "" {
		slog.WarnContext(ctx, "Refresh token is empty in RefreshTokenCommand")
		return nil, apperror.NewErrBadRequest("Refresh token is required", "")
	}

	slog.InfoContext(ctx, "Verifying provided refresh token.")
	claims, err := h.jwtService.VerifyToken(ctx, cmd.RefreshToken)
	if err != nil {
		slog.WarnContext(ctx, "Failed to verify refresh token", slog.Any("original_error", err.Error()))
		return nil, apperror.NewErrUnauthorized(fmt.Sprintf("Invalid refresh token: %s", err.Error()))
	}
	if claims.JTI == "" {
		slog.WarnContext(ctx, "Invalid refresh token: missing JTI claim", slog.String("uid_from_token", claims.UID))
		return nil, apperror.NewErrUnauthorized("Invalid refresh token: missing JTI claim")
	}
	slog.InfoContext(ctx, "Refresh token verified", slog.String("jti", claims.JTI), slog.String("uid", claims.UID))

	slog.InfoContext(ctx, "Validating refresh token in KV store", slog.String("jti", claims.JTI), slog.String("uid", claims.UID))
	if err := h.refreshTokenRepo.Validate(ctx, claims.JTI, claims.UID); err != nil {
		slog.WarnContext(ctx, "Refresh token validation failed in KV store", slog.String("jti", claims.JTI), slog.String("uid", claims.UID), slog.Any("original_error", err.Error()))
		return nil, apperror.NewErrUnauthorized(fmt.Sprintf("Refresh token validation failed: %s", err.Error()))
	}

	slog.InfoContext(ctx, "Deleting old refresh token from KV store.", slog.String("jti", claims.JTI))
	if err := h.refreshTokenRepo.Delete(ctx, claims.JTI); err != nil {
		slog.WarnContext(ctx, "Failed to delete old refresh token during rotation, but proceeding", slog.String("jti", claims.JTI), slog.Any("original_error", err.Error()))
	}

	slog.InfoContext(ctx, "Generating new token pair", slog.String("uid", claims.UID))
	newAccessToken, newRefreshToken, newJTI, expiresAt, err := h.jwtService.GenerateTokens(ctx, claims.UID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to generate new tokens", slog.String("uid", claims.UID), slog.Any("original_error", err.Error()))
		return nil, apperror.NewErrInternal("Failed to generate new tokens", err)
	}

	slog.InfoContext(ctx, "Saving new refresh token to KV", slog.String("new_jti", newJTI), slog.String("uid", claims.UID))
	if err := h.refreshTokenRepo.Save(ctx, newJTI, claims.UID, h.refreshTokenTTL); err != nil {
		slog.ErrorContext(ctx, "Failed to save new refresh token to KV", slog.String("new_jti", newJTI), slog.String("uid", claims.UID), slog.Any("original_error", err.Error()))
		return nil, apperror.NewErrInternal("Failed to save new refresh token to KV store", err)
	}

	slog.InfoContext(ctx, "Successfully refreshed tokens", slog.String("uid", claims.UID), slog.String("new_jti", newJTI))
	return &RefreshTokenResult{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}
