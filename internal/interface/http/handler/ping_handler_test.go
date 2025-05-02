package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aiirononeko/bulktrack-api/internal/interface/http/handler"
)

// --- モックの実装 ---

// mockPingQueryService は query.PingQueryService のモック実装です。
type mockPingQueryService struct {
	RetMsg string // Ping メソッドが返すメッセージ
	RetErr error  // Ping メソッドが返すエラー
}

// Ping はモック用の Ping メソッドです。設定された値を返します。
func (m *mockPingQueryService) Ping(ctx context.Context) (string, error) {
	return m.RetMsg, m.RetErr
}

// --- テスト関数 ---

func TestPingHandler_ServeHTTP(t *testing.T) {
	// --- テストケースの定義 (テーブル駆動テストにするとより良い) ---
	tests := []struct {
		name           string // テストケース名
		mockRetMsg     string // モックが返すメッセージ
		mockRetErr     error  // モックが返すエラー
		expectedStatus int    // 期待されるHTTPステータスコード
		expectedBody   string // 期待されるレスポンスボディ
	}{
		{
			name:           "Success",
			mockRetMsg:     "Pong from mock", // モックからの返り値だと分かるように
			mockRetErr:     nil,
			expectedStatus: http.StatusOK,
			expectedBody:   "Pong from mock",
		},
		{
			name:           "Service Error",
			mockRetMsg:     "",
			mockRetErr:     errors.New("mock service error"), // 何らかのエラー
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Internal Server Error\n", // http.Error は改行を追加する
		},
	}

	// --- 各テストケースの実行 ---
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange (準備)
			mockService := &mockPingQueryService{ // モックを作成し、期待値を設定
				RetMsg: tt.mockRetMsg,
				RetErr: tt.mockRetErr,
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
