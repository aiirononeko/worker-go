package http

import (
	"net/http"

	"github.com/aiirononeko/bulktrack-api/internal/interface/http/handler"
	"github.com/aiirononeko/bulktrack-api/internal/interface/http/middleware"
)

// Route は単一のルート定義を保持します。
// ServeMux はメソッドベースのルーティングを直接サポートしないため、
// ハンドラ内でメソッドスイッチを行うか、パスごとにメソッド別のハンドラを定義する必要があります。
// ここでは、ハンドラが全てのメソッドを処理し、内部で分岐することを想定します。
type Route struct {
	Path        string                  // 例: "/v1/menus"
	Handler     http.Handler            // このルートのメインハンドラ
	Middlewares []middleware.Middleware // このルートにのみ適用されるミドルウェア (認証など)
	// Method      string                  // オプション: http.MethodGet など。ハンドラ側で分岐する場合不要
}

// RouterDependencies はルーターが依存するものをまとめます。
// NewRouter に渡すために使います。
type RouterDependencies struct {
	Routes            []Route
	GlobalMiddlewares []middleware.Middleware   // 全てのルートに適用されるミドルウェア (CORS, Logging)
	AuthHandler       *handler.AuthHandler      // 認証エンドポイント用
	PingHandler       *handler.PingHandler      // pingエンドポイント用
	ListMenusHandler  *handler.ListMenusHandler // メニュー一覧用 (Routesでラップされる想定だが、個別に渡すことも考慮)
}

// NewRouter はルート定義とグローバルミドルウェアを受け取り、HTTPルーターを初期化します。
func NewRouter(deps RouterDependencies) http.Handler {
	mux := http.NewServeMux()

	// 定義されたルートを登録 (これらはルート固有ミドルウェア + グローバルミドルウェアが適用される)
	for _, route := range deps.Routes {
		allMiddlewares := make([]middleware.Middleware, 0, len(route.Middlewares)+len(deps.GlobalMiddlewares))
		allMiddlewares = append(allMiddlewares, route.Middlewares...)      // 内側のミドルウェア (例: Auth)
		allMiddlewares = append(allMiddlewares, deps.GlobalMiddlewares...) // 外側のミドルウェア (例: Logging, CORS)
		finalHandler := middleware.Chain(route.Handler, allMiddlewares...)
		mux.Handle(route.Path, finalHandler)
	}

	// --- グローバルミドルウェアのみを適用するエンドポイント ---

	if deps.AuthHandler != nil {
		// /v1/auth/* エンドポイントにはグローバルミドルウェア (CORS, Logging) を適用
		mux.Handle("POST /v1/auth/device", middleware.Chain(http.HandlerFunc(deps.AuthHandler.ActivateDevice), deps.GlobalMiddlewares...))
		mux.Handle("POST /v1/auth/refresh", middleware.Chain(http.HandlerFunc(deps.AuthHandler.RefreshToken), deps.GlobalMiddlewares...))
		mux.Handle("POST /v1/auth/logout", middleware.Chain(http.HandlerFunc(deps.AuthHandler.Logout), deps.GlobalMiddlewares...))
	}

	if deps.PingHandler != nil {
		// /ping エンドポイントにもグローバルミドルウェアを適用
		mux.Handle("/ping", middleware.Chain(deps.PingHandler, deps.GlobalMiddlewares...))
	}

	return mux
}
