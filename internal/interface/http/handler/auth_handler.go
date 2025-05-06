package handler

import (
	"encoding/json"
	"net/http"
	"time"

	appCmd "github.com/aiirononeko/bulktrack-api/internal/app/command" // Application 層を import
)

// AuthHandler は認証関連のエンドポイントを扱います。
type AuthHandler struct {
	activateDeviceCmd appCmd.ActivateDeviceHandler
	refreshTokenCmd   appCmd.RefreshTokenHandler
	logoutCmd         appCmd.LogoutHandler // LogoutHandler への依存を追加
}

// NewAuthHandler は新しい AuthHandler を初期化します。
func NewAuthHandler(activateDeviceCmd appCmd.ActivateDeviceHandler, refreshTokenCmd appCmd.RefreshTokenHandler, logoutCmd appCmd.LogoutHandler) *AuthHandler {
	return &AuthHandler{
		activateDeviceCmd: activateDeviceCmd,
		refreshTokenCmd:   refreshTokenCmd,
		logoutCmd:         logoutCmd, // 依存を初期化
	}
}

// ActivateDevice はデバイスをアクティベートし、トークンを返します。
// POST /v1/auth/device
func (h *AuthHandler) ActivateDevice(w http.ResponseWriter, r *http.Request) {
	deviceID := r.Header.Get("X-Device-Id")
	if deviceID == "" {
		http.Error(w, "X-Device-Id header is required", http.StatusBadRequest)
		return
	}

	// Application 層のコマンドを呼び出す
	cmd := appCmd.ActivateDeviceCommand{DeviceID: deviceID}
	result, err := h.activateDeviceCmd.Handle(r.Context(), cmd)
	if err != nil {
		http.Error(w, "Failed to activate device", http.StatusInternalServerError)
		return
	}

	// レスポンスを構築
	resp := struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"` // Access token の有効期間 (秒)
	}{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    int64(time.Until(result.ExpiresAt).Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// RefreshTokenRequest はリフレッシュトークンリクエストのボディを表します。
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshTokenResponse はリフレッシュ成功時のレスポンスを表します。
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // Access token の有効期間 (秒)
}

// RefreshToken は受け取ったリフレッシュトークンを検証し、新しいトークンペアを返します。
// POST /v1/auth/refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	// 1. リクエストボディをデコード
	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.RefreshToken == "" {
		http.Error(w, "refresh_token is required", http.StatusBadRequest)
		return
	}

	// 2. Application 層の RefreshTokenCommand を呼び出す
	cmd := appCmd.RefreshTokenCommand{RefreshToken: req.RefreshToken}
	result, err := h.refreshTokenCmd.Handle(r.Context(), cmd)
	if err != nil {
		http.Error(w, "Failed to refresh token", http.StatusInternalServerError)
		return
	}

	// 3. 成功レスポンスを返す
	resp := RefreshTokenResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    int64(time.Until(result.ExpiresAt).Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// LogoutRequest はログアウトリクエストのボディを表します。
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Logout は受け取ったリフレッシュトークンを無効化します。
// POST /v1/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// 1. リクエストボディをデコード
	var req LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.RefreshToken == "" {
		http.Error(w, "refresh_token is required", http.StatusBadRequest)
		return
	}

	// 2. Application 層の LogoutCommand を呼び出す
	cmd := appCmd.LogoutCommand{RefreshToken: req.RefreshToken}
	if err := h.logoutCmd.Handle(r.Context(), cmd); err != nil {
		// エラーの種類に応じてクライアントに返すステータスコードを検討する
		// 例えば、トークンが無効だった場合は 400 Bad Request や 401 Unauthorized など
		// KV操作失敗は 500 Internal Server Error
		// ここでは詳細なエラーハンドリングは省略し、一律 500 を返す
		http.Error(w, "Failed to logout", http.StatusInternalServerError)
		return
	}

	// 3. 成功レスポンス (No Content)
	w.WriteHeader(http.StatusNoContent)
}
