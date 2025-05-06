package middleware

import (
	"context"
	"net/http"
	"strings"

	domainAuth "github.com/aiirononeko/bulktrack-api/internal/domain/auth"
)

// AuthContextKey はコンテキストに認証情報を格納するためのキーの型です。
type AuthContextKey string

const (
	// UIDKey はコンテキストに保存されるユーザー/デバイス識別子のキーです。
	UIDKey AuthContextKey = "uid"
)

// RequireAuth は JWT 認証を要求するミドルウェアです。
// JWTService を受け取り、トークンの検証を行います。
func RequireAuth(jwtService domainAuth.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawToken := r.Header.Get("Authorization")
			if rawToken == "" {
				http.Error(w, "Authorization header is required", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(rawToken, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, "Authorization header format must be Bearer {token}", http.StatusUnauthorized)
				return
			}
			tokenString := parts[1]

			claims, err := jwtService.VerifyToken(r.Context(), tokenString)
			if err != nil {
				// エラーの種類によってログレベルを変えたり、より詳細なエラーメッセージを返すことも検討
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			if claims.UID == "" {
				// UIDが空の場合は不正なトークンとして扱う（通常はVerifyToken内でチェックされるべきだが念のため）
				http.Error(w, "Invalid token: UID missing", http.StatusUnauthorized)
				return
			}

			// 認証情報をコンテキストに保存
			ctx := context.WithValue(r.Context(), UIDKey, claims.UID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
