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
	Routes                []Route
	GlobalMiddlewares     []middleware.Middleware   // 全てのルートに適用されるミドルウェア (CORS, Logging)
	AuthHandler           *handler.AuthHandler      // 認証エンドポイント用
	PingHandler           *handler.PingHandler      // pingエンドポイント用
	MenuHandler           *handler.MenuHandler      // /v1/menus 用 (GET, POST)
	WorkoutHandler        *handler.WorkoutHandler   // Added WorkoutHandler
	DashboardHandler      *handler.DashboardHandler // Added DashboardHandler
	RequireAuthMiddleware middleware.Middleware     // 認証ミドルウェアのインスタンス
}

// NewRouter はルート定義とグローバルミドルウェアを受け取り、HTTPルーターを初期化します。
func NewRouter(deps RouterDependencies) http.Handler {
	mux := http.NewServeMux()

	// /v1/menus ルート (認証が必要)
	if deps.MenuHandler != nil {
		menuMiddlewares := []middleware.Middleware{deps.RequireAuthMiddleware} // ルート固有ミドルウェア
		allMenuMiddlewares := make([]middleware.Middleware, 0, len(menuMiddlewares)+len(deps.GlobalMiddlewares))
		allMenuMiddlewares = append(allMenuMiddlewares, menuMiddlewares...)        // 内側 (Auth)
		allMenuMiddlewares = append(allMenuMiddlewares, deps.GlobalMiddlewares...) // 外側 (Logging, CORS)
		finalMenuHandler := middleware.Chain(deps.MenuHandler, allMenuMiddlewares...)
		// Register for /v1/menus.
		// If you also need to handle /v1/menus/*, you might need a separate registration
		// or ensure your MenuHandler can correctly dispatch based on the full path.
		mux.Handle("/v1/menus", finalMenuHandler)
	}

	// /v1/workouts ルート (認証が必要)
	if deps.WorkoutHandler != nil {
		workoutMiddlewares := []middleware.Middleware{deps.RequireAuthMiddleware} // Route specific middleware
		allWorkoutMiddlewares := make([]middleware.Middleware, 0, len(workoutMiddlewares)+len(deps.GlobalMiddlewares))
		allWorkoutMiddlewares = append(allWorkoutMiddlewares, workoutMiddlewares...)     // Inner (Auth)
		allWorkoutMiddlewares = append(allWorkoutMiddlewares, deps.GlobalMiddlewares...) // Outer (Logging, CORS)
		finalWorkoutHandler := middleware.Chain(deps.WorkoutHandler, allWorkoutMiddlewares...)
		mux.Handle("/v1/workouts", finalWorkoutHandler) // Path only, WorkoutHandler handles methods internally (expects POST)
	}

	// /v1/dashboard ルート (認証が必要)
	if deps.DashboardHandler != nil {
		dashboardMiddlewares := []middleware.Middleware{deps.RequireAuthMiddleware}
		allDashboardMiddlewares := make([]middleware.Middleware, 0, len(dashboardMiddlewares)+len(deps.GlobalMiddlewares))
		allDashboardMiddlewares = append(allDashboardMiddlewares, dashboardMiddlewares...)
		allDashboardMiddlewares = append(allDashboardMiddlewares, deps.GlobalMiddlewares...)
		finalDashboardHandler := middleware.Chain(deps.DashboardHandler, allDashboardMiddlewares...)
		// DashboardHandler は /v1/dashboard/* のようなプレフィックスで登録し、内部でサブパスを処理
		mux.Handle("/v1/dashboard/", finalDashboardHandler)
	}

	// --- その他のカスタムルート (Routes スライスを使う場合) ---
	for _, route := range deps.Routes {
		allMiddlewares := make([]middleware.Middleware, 0, len(route.Middlewares)+len(deps.GlobalMiddlewares))
		allMiddlewares = append(allMiddlewares, route.Middlewares...)      // Route specific
		allMiddlewares = append(allMiddlewares, deps.GlobalMiddlewares...) // Global
		finalHandler := middleware.Chain(route.Handler, allMiddlewares...)
		mux.Handle(route.Path, finalHandler)
	}

	// --- グローバルミドルウェアのみを適用するエンドポイント ---

	if deps.AuthHandler != nil {
		// AuthHandler の各メソッドを http.HandlerFunc として登録
		// ActivateDevice, RefreshToken, Logout が func(w http.ResponseWriter, r *http.Request) のシグネチャを持つ前提
		// また、AuthHandler.ServeHTTP でメインのルーティングが行われるのではなく、これらのメソッドが直接エンドポイントとなる想定
		// これは以前のスナップショットの動作に合わせるため。
		// もし AuthHandler.ServeHTTP が /v1/auth/* を処理するなら、そのように変更する必要がある。

		// Check if the specific handler methods exist and are intended to be routed this way.
		// This assumes AuthHandler does not have a ServeHTTP that dispatches these itself for /v1/auth/*.
		mux.Handle("/v1/auth/device", middleware.Chain(http.HandlerFunc(deps.AuthHandler.ActivateDevice), deps.GlobalMiddlewares...))
		mux.Handle("/v1/auth/refresh", middleware.Chain(http.HandlerFunc(deps.AuthHandler.RefreshToken), deps.GlobalMiddlewares...))
		mux.Handle("/v1/auth/logout", middleware.Chain(http.HandlerFunc(deps.AuthHandler.Logout), deps.GlobalMiddlewares...))
	}

	if deps.PingHandler != nil {
		// /ping エンドポイントにもグローバルミドルウェアを適用
		mux.Handle("/ping", middleware.Chain(deps.PingHandler, deps.GlobalMiddlewares...))
	}

	return mux
}
