package query

import (
	"context"
	"log/slog"
	"time"
)

// PingResult は Ping の結果を保持します。
type PingResult struct {
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// PingQueryService は Ping ユースケースのインターフェースです。
type PingQueryService interface {
	Execute(ctx context.Context) PingResult
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

// Execute は Ping のユースケースを実行します。
func (s *pingQueryServiceImpl) Execute(ctx context.Context) PingResult {
	slog.InfoContext(ctx, "Executing Ping query.")
	now := time.Now()
	result := PingResult{
		Message:   "pong",
		Timestamp: now,
	}
	// ログに含める情報を構造化
	slog.InfoContext(ctx, "Ping query executed successfully",
		slog.String("message", result.Message),
		slog.Time("timestamp", result.Timestamp),
	)
	return result
}
