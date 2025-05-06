package query

import (
	"context"
	"fmt"
	"log"

	"github.com/aiirononeko/bulktrack-api/internal/app/dto"
	"github.com/aiirononeko/bulktrack-api/internal/domain/menu"
)

// ListMenusQueryService はメニュー一覧取得ユースケースのインターフェースです。
type ListMenusQueryService interface {
	// Execute retrieves menus for a given device ID.
	// deviceID should be the pure UUID string, without any prefixes.
	Execute(ctx context.Context, deviceID string) ([]dto.MenuDTO, error)
}

// listMenusQueryServiceImpl は ListMenusQueryService の実装です。
type listMenusQueryServiceImpl struct {
	menuRepo menu.MenuRepository
}

// NewListMenusQueryService は ListMenusQueryService の新しいインスタンスを生成します。
func NewListMenusQueryService(mr menu.MenuRepository) ListMenusQueryService {
	return &listMenusQueryServiceImpl{menuRepo: mr}
}

// Execute はメニュー一覧取得のユースケースを実行します。
// deviceID はプレフィックスなしの純粋なUUID文字列である必要があります。
func (s *listMenusQueryServiceImpl) Execute(ctx context.Context, deviceID string) ([]dto.MenuDTO, error) {
	log.Printf("INFO: Starting to execute ListMenus query for DeviceID: %s", deviceID)

	if deviceID == "" {
		log.Printf("ERROR: DeviceID is empty in ListMenus query execution.")
		return nil, fmt.Errorf("invalid argument: deviceID cannot be empty")
	}

	log.Printf("INFO: Calling MenuRepository.ListMenusByDeviceId with DeviceID: %s", deviceID)

	menus, err := s.menuRepo.ListMenusByDeviceId(ctx, deviceID)
	if err != nil {
		log.Printf("ERROR: Failed to list menus for DeviceID %s from repository: %v", deviceID, err)
		return nil, fmt.Errorf("failed to list menus for deviceID %s: %w", deviceID, err)
	}

	menuDTOs := make([]dto.MenuDTO, 0, len(menus))
	for _, m := range menus {
		menuDTOs = append(menuDTOs, dto.MenuDTO{
			ID:          m.ID, // Assuming m.ID is already uuid.UUID
			Name:        m.Name,
			Description: m.Description,
			SortOrder:   m.SortOrder,
			CreatedAt:   m.CreatedAt,
			UpdatedAt:   m.UpdatedAt,
		})
	}

	log.Printf("INFO: Successfully executed ListMenus query for DeviceID: %s. Found %d menus.", deviceID, len(menuDTOs))
	return menuDTOs, nil
}
