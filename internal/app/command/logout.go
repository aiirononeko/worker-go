package command

import (
	"context"
	"fmt"
	"log"

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
	log.Printf("INFO: Starting logout process.") // リフレッシュトークン自体はログに出さない

	if cmd.RefreshToken == "" {
		log.Printf("ERROR: Refresh token is empty in LogoutCommand")
		return fmt.Errorf("refresh token is required")
	}

	log.Printf("INFO: Verifying refresh token for logout.")
	claims, err := h.jwtService.VerifyToken(ctx, cmd.RefreshToken)
	if err != nil {
		// トークンが無効でもログアウト処理自体はエラーにしない場合もあるが、
		// KVから削除する対象を特定できないため、ここではエラーとする。
		log.Printf("ERROR: Failed to verify refresh token during logout: %v", err)
		return fmt.Errorf("failed to verify refresh token for logout: %w", err)
	}

	if claims.JTI == "" {
		log.Printf("ERROR: Invalid refresh token during logout: missing JTI claim. UID from token: %s", claims.UID)
		return fmt.Errorf("invalid refresh token for logout: missing jti claim")
	}
	log.Printf("INFO: Refresh token verified for logout. JTI: %s, UID: %s", claims.JTI, claims.UID)

	log.Printf("INFO: Deleting refresh token (JTI: %s) from repository for logout.", claims.JTI)
	if err := h.refreshTokenRepo.Delete(ctx, claims.JTI); err != nil {
		// 既に削除されている場合やKV操作失敗もエラーとして記録
		log.Printf("ERROR: Failed to delete refresh token (JTI: %s) from repository during logout: %v", claims.JTI, err)
		return fmt.Errorf("failed to delete refresh token from repository: %w", err)
	}

	log.Printf("INFO: Successfully logged out by deleting refresh token (JTI: %s) for UID: %s", claims.JTI, claims.UID)
	return nil
}
