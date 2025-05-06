package middleware

import (
	"log"
	"net/http"
	"time"
)

// responseWriter は http.ResponseWriter をラップしてステータスコードをキャプチャします。
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// LoggingMiddleware はリクエストの情報をログに出力するミドルウェアです。
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// ステータスコードをキャプチャするために ResponseWriter をラップ
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK} // デフォルトは 200 OK

		// 次のハンドラを実行
		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		log.Printf(
			"%s %s %s %d %s %s",
			r.Method,
			r.RequestURI,
			r.Proto,
			rw.status, // キャプチャしたステータスコード
			duration,
			r.UserAgent(),
		)
	})
}
