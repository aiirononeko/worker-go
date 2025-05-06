package command

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aiirononeko/bulktrack-api/internal/app/apperror"
	domainAuth "github.com/aiirononeko/bulktrack-api/internal/domain/auth"
)

// LogoutCommand はログアウト処理の入力データを保持します。
// リフレッシュトークンそのものを送るか、そのJTIを送るかは設計次第ですが、
// ここではクライアントがリフレッシュトークン自体を持っている前提とします。
// そのリフレッシュトークンをパースしてJTIを取得する必要があります。
type LogoutCommand struct {
	RefreshToken string
}

// LogoutHandler はログアウトのユースケースを実行します。
type LogoutHandler interface {
	Handle(ctx context.Context, cmd LogoutCommand) error
}

// --- Implementation ---

type logoutHandler struct {
	jwtService       domainAuth.JWTService             // RefreshTokenからJTIを抽出するために使用
	refreshTokenRepo domainAuth.RefreshTokenRepository // KVからJTIを削除するために使用
}

// NewLogoutHandler は新しい logoutHandler を初期化します。
func NewLogoutHandler(jwtService domainAuth.JWTService, refreshTokenRepo domainAuth.RefreshTokenRepository) LogoutHandler {
	return &logoutHandler{
		jwtService:       jwtService,
		refreshTokenRepo: refreshTokenRepo,
	}
}

func (h *logoutHandler) Handle(ctx context.Context, cmd LogoutCommand) error {
	slog.InfoContext(ctx, "Starting logout process.")

	if cmd.RefreshToken == "" {
		slog.WarnContext(ctx, "Refresh token is empty in LogoutCommand")
		return apperror.NewErrBadRequest("Refresh token is required", "")
	}

	slog.InfoContext(ctx, "Verifying refresh token for logout.")
	claims, err := h.jwtService.VerifyToken(ctx, cmd.RefreshToken)
	if err != nil {
		slog.WarnContext(ctx, "Failed to verify refresh token during logout", slog.Any("original_error", err.Error()))
		return apperror.NewErrUnauthorized(fmt.Sprintf("Invalid refresh token for logout: %s", err.Error()))
	}

	if claims.JTI == "" {
		slog.WarnContext(ctx, "Invalid refresh token during logout: missing JTI claim", slog.String("uid_from_token", claims.UID))
		return apperror.NewErrUnauthorized("Invalid refresh token for logout: missing JTI claim")
	}
	slog.InfoContext(ctx, "Refresh token verified for logout", slog.String("jti", claims.JTI), slog.String("uid", claims.UID))

	slog.InfoContext(ctx, "Deleting refresh token from repository for logout", slog.String("jti", claims.JTI))
	if err := h.refreshTokenRepo.Delete(ctx, claims.JTI); err != nil {
		// 既に削除されている場合(ErrNotFoundなど)や、KVストアの一時的な障害(ErrInternal)などが考えられる。
		// ログアウト処理としては、既に存在しないなら成功とみなしても良いが、ここではKV操作の失敗は内部エラーとする。
		slog.ErrorContext(ctx, "Failed to delete refresh token from repository during logout",
			slog.String("jti", claims.JTI),
			slog.Any("original_error", err.Error()),
		)
		return apperror.NewErrInternal("Failed to delete refresh token from KV store during logout", err)
	}

	slog.InfoContext(ctx, "Successfully logged out by deleting refresh token", slog.String("jti", claims.JTI), slog.String("uid", claims.UID))
	return nil
}
