package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aiirononeko/bulktrack-api/internal/app/query"
	"github.com/aiirononeko/bulktrack-api/internal/interface/http/handler"
)

// --- モックの実装 ---

// mockPingQueryService は query.PingQueryService のモック実装です。
type mockPingQueryService struct {
	RetResult query.PingResult // Execute メソッドが返す PingResult
}

// Execute はモック用の Execute メソッドです。設定された値を返します。
func (m *mockPingQueryService) Execute(ctx context.Context) query.PingResult {
	return m.RetResult
}

// --- テスト関数 ---

func TestPingHandler_ServeHTTP(t *testing.T) {
	// --- テストケースの定義 ---
	now := time.Now()
	tests := []struct {
		name           string           // テストケース名
		mockRetResult  query.PingResult // モックが返す PingResult
		expectedStatus int              // 期待されるHTTPステータスコード
		expectedBody   string           // 期待されるレスポンスボディ (JSON文字列)
	}{
		{
			name: "Success",
			mockRetResult: query.PingResult{
				Message:   "pong from mock",
				Timestamp: now,
			},
			expectedStatus: http.StatusOK,
			// expectedBody は mockRetResult をJSONエンコードしたものになる
		},
		// Executeはエラーを返さないので、Service Errorケースは削除
	}

	// --- 各テストケースの実行 ---
	for _, tt := range tests {
		// expectedBody を動的に生成
		expectedBodyBytes, _ := json.Marshal(tt.mockRetResult)
		tt.expectedBody = string(expectedBodyBytes) + "\n" // http.ResponseWriterはしばしば改行を追加

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange (準備)
			mockService := &mockPingQueryService{ // モックを作成し、期待値を設定
				RetResult: tt.mockRetResult,
			}
			pingHandler := handler.NewPingHandler(mockService)       // モックを注入してハンドラを作成
			req := httptest.NewRequest(http.MethodGet, "/ping", nil) // テストリクエスト作成
			rec := httptest.NewRecorder()                            // レスポンスレコーダー作成

			// Act (実行)
			pingHandler.ServeHTTP(rec, req) // ハンドラを実行

			// Assert (検証)
			if rec.Code != tt.expectedStatus {
				t.Errorf("unexpected status code: got %d, want %d", rec.Code, tt.expectedStatus)
			}
			if rec.Body.String() != tt.expectedBody {
				t.Errorf("unexpected response body: got %q, want %q", rec.Body.String(), tt.expectedBody)
			}
		})
	}
}
