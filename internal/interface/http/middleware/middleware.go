package middleware

import "net/http"

// Middleware は http.Handler をラップして新しい http.Handler を返す関数の型です。
type Middleware func(http.Handler) http.Handler

// Chain は複数のミドルウェアを順番に適用するヘルパー関数です。
// middlewares の最初のミドルウェアが最も外側に、最後のミドルウェアが最も内側 (ハンドラに近い側) に適用されます。
// 例: Chain(handler, mw1, mw2, mw3) は mw1(mw2(mw3(handler))) と等価です。
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	// スライスの逆順で適用していくことで、指定された順序（外側から内側へ）でラップされるようにする
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
