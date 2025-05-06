package command

import (
	"context"
	"fmt"

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
	if cmd.RefreshToken == "" {
		return fmt.Errorf("refresh token is required")
	}

	// 1. リフレッシュトークンをパースして JTI (JWT ID) を取得
	claims, err := h.jwtService.VerifyToken(ctx, cmd.RefreshToken)
	if err != nil {
		// トークンが無効（期限切れ、不正な形式など）でも、クライアントはログアウトしたいはずなので、
		// エラーにはせず、単に削除処理に進まない（あるいはログだけ出す）という考え方もある。
		// ここでは、有効なリフレッシュトークンでないとKVから削除できないためエラーとする。
		return fmt.Errorf("failed to verify refresh token: %w", err)
	}

	// リフレッシュトークンには JTI が必須
	if claims.JTI == "" {
		return fmt.Errorf("invalid refresh token: missing jti claim")
	}

	// 2. RefreshTokenRepository を使って KV ストアから JTI を削除
	if err := h.refreshTokenRepo.Delete(ctx, claims.JTI); err != nil {
		// 既に削除されている場合や、何らかの理由でKV操作に失敗した場合
		// ログアウト処理としては「成功」として扱っても良いかもしれないが、ここではエラーを返す
		return fmt.Errorf("failed to delete refresh token from repository: %w", err)
	}

	return nil
}
