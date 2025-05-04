package command

import (
	"context"
	"fmt"
	"time"

	domainAuth "github.com/aiirononeko/bulktrack-api/internal/domain/auth"
	"github.com/aiirononeko/bulktrack-api/internal/domain/device"
)

// ActivateDeviceCommand はデバイスアクティベートの入力データを保持します。
type ActivateDeviceCommand struct {
	DeviceID string
}

// ActivateDeviceResult はデバイスアクティベートの結果を保持します。
type ActivateDeviceResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time // Access token の有効期限
}

// ActivateDeviceHandler はデバイスアクティベートのユースケースを実行します。
type ActivateDeviceHandler interface {
	Handle(ctx context.Context, cmd ActivateDeviceCommand) (*ActivateDeviceResult, error)
}

// --- Implementation ---

type activateDeviceHandler struct {
	deviceRepo       device.DeviceRepository
	jwtService       domainAuth.JWTService
	refreshTokenRepo domainAuth.RefreshTokenRepository
	refreshTokenTTL  time.Duration
}

// NewActivateDeviceHandler は新しい activateDeviceHandler を初期化します。
func NewActivateDeviceHandler(deviceRepo device.DeviceRepository, jwtService domainAuth.JWTService, refreshTokenRepo domainAuth.RefreshTokenRepository, refreshTokenTTL time.Duration) ActivateDeviceHandler {
	return &activateDeviceHandler{
		deviceRepo:       deviceRepo,
		jwtService:       jwtService,
		refreshTokenRepo: refreshTokenRepo,
		refreshTokenTTL:  refreshTokenTTL,
	}
}

func (h *activateDeviceHandler) Handle(ctx context.Context, cmd ActivateDeviceCommand) (*ActivateDeviceResult, error) {
	// DeviceID のバリデーション (簡易)
	if cmd.DeviceID == "" {
		return nil, fmt.Errorf("invalid device ID")
	}

	// 1. デバイスを検索または作成
	existingDevice, err := h.deviceRepo.FindByID(ctx, cmd.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to find device: %w", err)
	}

	var dev *device.Device
	if existingDevice == nil {
		dev = device.NewDevice(cmd.DeviceID, nil)
	} else {
		dev = existingDevice
		dev.UpdateLastSeen()
	}

	// 2. デバイス情報を保存
	if err := h.deviceRepo.Save(ctx, dev); err != nil {
		return nil, fmt.Errorf("failed to save device: %w", err)
	}

	// 3. JWT を生成
	uid := fmt.Sprintf("device:%s", dev.ID)
	accessToken, refreshToken, refreshTokenJTI, expiresAt, err := h.jwtService.GenerateTokens(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// ★★★ 追加: 生成したリフレッシュトークン情報を KV に保存 ★★★
	if err := h.refreshTokenRepo.Save(ctx, refreshTokenJTI, uid, h.refreshTokenTTL); err != nil {
		// KV への保存失敗は致命的ではないかもしれないが、リフレッシュが機能しなくなる
		// ログを出力し、エラーを返しされ、場合によっては成功としてトークンを返すことも検討可
		fmt.Printf("Warning: failed to save refresh token to KV (jti: %s, uid: %s): %v\n", refreshTokenJTI, uid, err)
		return nil, fmt.Errorf("failed to save refresh token state: %w", err)
	}
	// ★★★ ここまで ★★★

	// 4. 結果を返す
	return &ActivateDeviceResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}
