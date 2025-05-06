package middleware

import (
	"net/http"
	"strconv"
	"strings"
)

// CORSOptions はCORSミドルウェアの設定を保持します。
// 今後の拡張のために構造体として定義しますが、まずはハードコードされた値を使用します。
type CORSOptions struct {
	AllowedOrigins   []string // 例: ["http://localhost:3000", "https://app.example.com"]
	AllowedMethods   []string // 例: ["GET", "POST", "PUT", "DELETE"]
	AllowedHeaders   []string // 例: ["Authorization", "Content-Type"]
	AllowCredentials bool
	MaxAge           int // 秒単位
}

// DefaultCORSOptions はデフォルトのCORS設定を返します。
// セキュリティを考慮し、本番環境ではより厳格な設定にすることを推奨します。
func DefaultCORSOptions() CORSOptions {
	return CORSOptions{
		AllowedOrigins: []string{"*"}, // すべてのオリジンを許可 (開発用)
		AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions, http.MethodPatch},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-Requested-With",
			"X-Device-Id", // カスタムヘッダー
			// 他にクライアントが送信する可能性のあるヘッダーを追加
		},
		AllowCredentials: true, // CookieやAuthorizationヘッダーなどの認証情報を含むリクエストを許可
		MaxAge:           3600, // 1時間
	}
}

// CORS はCORSヘッダーを処理するミドルウェアです。
func CORS(next http.Handler) http.Handler {
	return CORSWithOptions(next, DefaultCORSOptions())
}

// CORSWithOptions は指定されたオプションでCORSミドルウェアを返します。
func CORSWithOptions(next http.Handler, opts CORSOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allowed Origins の設定
		origin := r.Header.Get("Origin")
		allowedOrigin := ""
		if len(opts.AllowedOrigins) == 0 || opts.AllowedOrigins[0] == "*" {
			allowedOrigin = origin   // 任意のオリジンを許可する場合は、リクエストされたオリジンをそのまま返すのが一般的
			if allowedOrigin == "" { // オリジンヘッダがない場合 (同一オリジンリクエストなど)
				allowedOrigin = "*"
			}
		} else {
			for _, o := range opts.AllowedOrigins {
				if o == origin {
					allowedOrigin = origin
					break
				}
			}
		}
		if allowedOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		}

		if opts.AllowCredentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// プリフライトリクエスト (OPTIONS) の処理
		if r.Method == http.MethodOptions {
			if len(opts.AllowedMethods) > 0 {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(opts.AllowedMethods, ", "))
			}
			if len(opts.AllowedHeaders) > 0 {
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(opts.AllowedHeaders, ", "))
			}
			if opts.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", strconv.Itoa(opts.MaxAge)) // strconv.Itoa を使用
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// 通常のリクエストの場合は次のハンドラへ
		next.ServeHTTP(w, r)
	})
}
