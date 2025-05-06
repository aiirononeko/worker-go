package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// HTTPErrorResponse is a generic JSON error response body.
// Renamed from ErrorResponse to avoid conflicts if other ErrorResponse types exist elsewhere,
// and to be more specific about its use with HTTP.
type HTTPErrorResponse struct {
	Error   string `json:"error"`             // A high-level error message string.
	Details string `json:"details,omitempty"` // More detailed error information, often the error.Error() string.
}

// SendJSONError はJSON形式でエラーレスポンスを送信するヘルパー関数です。
// この関数は handler パッケージ内で共通して使用されます。
func SendJSONError(w http.ResponseWriter, logger *slog.Logger, message string, statusCode int, details string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	response := HTTPErrorResponse{
		Error:   message,
		Details: details,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// SendJSONError 自体でエラーが発生した場合のフォールバックロギング
		// ここで使用する logger は呼び出し元から渡されたものを使用します。
		logger.Error("Failed to encode error response in SendJSONError", slog.Any("error", err),
			slog.Int("original_status", statusCode),
			slog.String("original_message", message),
			slog.String("original_details", details),
		)
	}
}
