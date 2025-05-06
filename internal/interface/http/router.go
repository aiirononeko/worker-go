package http

import (
	"net/http"

	"github.com/aiirononeko/bulktrack-api/internal/interface/http/handler"
)

// RouterDependencies はルーターが依存するハンドラーをまとめます。
type RouterDependencies struct {
	AuthHandler      *handler.AuthHandler
	PingHandler      *handler.PingHandler
	ListMenusHandler *handler.ListMenusHandler
	// TODO: 他に必要なハンドラーがあれば追加
}

// NewRouter は依存関係を受け取り、ルーティングが設定された新しい HTTP ルーターを初期化して返します。
func NewRouter(deps RouterDependencies) http.Handler {
	mux := http.NewServeMux()

	// --- ルーティング設定 ---

	// Auth routes
	if deps.AuthHandler != nil {
		mux.HandleFunc("POST /v1/auth/device", deps.AuthHandler.ActivateDevice)
		mux.HandleFunc("POST /v1/auth/refresh", deps.AuthHandler.RefreshToken)
		mux.HandleFunc("POST /v1/auth/logout", deps.AuthHandler.Logout)
	}

	// Menu routes
	if deps.ListMenusHandler != nil {
		mux.Handle("/v1/menus", deps.ListMenusHandler) // ServeHTTP を実装しているので Handle
		// TODO: 他のメニュー関連エンドポイントを追加
	}

	// Ping route
	if deps.PingHandler != nil {
		mux.Handle("/ping", deps.PingHandler) // ServeHTTP を実装しているので Handle
	}

	// TODO: ミドルウェアの適用 (例: CORS, Logging, Auth) をここで行うか、main.go でルーターをラップする

	return mux
}
