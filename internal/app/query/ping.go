package query

import (
	"context"
	"log"
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
	log.Printf("INFO: Executing Ping query.")
	result := PingResult{
		Message:   "pong",
		Timestamp: time.Now(),
	}
	log.Printf("INFO: Ping query executed successfully. Message: %s, Timestamp: %s", result.Message, result.Timestamp)
	return result
}
