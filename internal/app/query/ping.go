package query

import (
	"context"
)

// PingQueryService は Ping の問い合わせ処理を提供するインターフェースです。
type PingQueryService interface {
	Ping(ctx context.Context) (string, error)
}

// pingQueryServiceImpl は PingQueryService の実装です。
// 今回は依存関係がないためフィールドは空です。
type pingQueryServiceImpl struct {
	// ここに依存する Repository などのインターフェースをフィールドとして持つことが多い
}

// NewPingQueryService は PingQueryService の新しいインスタンスを生成します。
func NewPingQueryService() PingQueryService {
	return &pingQueryServiceImpl{}
}

// Ping は Ping の問い合わせ処理を実行します。
func (s *pingQueryServiceImpl) Ping(ctx context.Context) (string, error) {
	return "Pong", nil
}
