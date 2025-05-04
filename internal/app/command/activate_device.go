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
	deviceRepo device.DeviceRepository
	jwtService domainAuth.JWTService
}

// NewActivateDeviceHandler は新しい activateDeviceHandler を初期化します。
func NewActivateDeviceHandler(deviceRepo device.DeviceRepository, jwtService domainAuth.JWTService) ActivateDeviceHandler {
	return &activateDeviceHandler{
		deviceRepo: deviceRepo,
		jwtService: jwtService,
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
	accessToken, refreshToken, expiresAt, err := h.jwtService.GenerateTokens(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// 4. 結果を返す
	return &ActivateDeviceResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}
