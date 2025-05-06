package query

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aiirononeko/bulktrack-api/internal/app/dto"
	"github.com/aiirononeko/bulktrack-api/internal/domain/menu"
	"github.com/aiirononeko/bulktrack-api/internal/interface/http/middleware" // middleware パッケージをインポートして UIDKey を使用
)

// ListMenusQueryService はメニュー一覧取得ユースケースのインターフェースです。
type ListMenusQueryService interface {
	Execute(ctx context.Context) ([]dto.MenuDTO, error) // userID 引数を削除
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
func (s *listMenusQueryServiceImpl) Execute(ctx context.Context) ([]dto.MenuDTO, error) {
	log.Printf("INFO: Starting to execute ListMenus query.")

	uid, ok := ctx.Value(middleware.UIDKey).(string)
	if !ok || uid == "" {
		log.Printf("ERROR: UID not found in context during ListMenus execution.")
		return nil, fmt.Errorf("authentication required: UID not found in context")
	}
	log.Printf("INFO: ListMenus query called by UID: %s", uid)

	var idToQuery string
	if strings.HasPrefix(uid, "device:") {
		idToQuery = strings.TrimPrefix(uid, "device:")
		log.Printf("INFO: Extracted DeviceID %s from UID %s for ListMenus query.", idToQuery, uid)
	} else if strings.HasPrefix(uid, "user:") {
		userIDToQuery := strings.TrimPrefix(uid, "user:")
		log.Printf("WARN: ListMenus by UserID (%s) is not yet supported. UID was: %s", userIDToQuery, uid)
		return nil, fmt.Errorf("listing menus by user_id (%s) is not yet supported, use device_id", userIDToQuery)
	} else {
		log.Printf("ERROR: Invalid UID format in context during ListMenus: %s", uid)
		return nil, fmt.Errorf("invalid UID format in context: %s", uid)
	}

	if idToQuery == "" {
		log.Printf("ERROR: Failed to extract actual ID from UID %s for ListMenus query.", uid)
		return nil, fmt.Errorf("failed to extract actual ID from UID: %s", uid)
	}

	log.Printf("INFO: Calling MenuRepository.ListMenusByDeviceId with DeviceID: %s", idToQuery)
	menus, err := s.menuRepo.ListMenusByDeviceId(ctx, idToQuery)
	if err != nil {
		log.Printf("ERROR: Failed to list menus for DeviceID %s from repository: %v", idToQuery, err)
		return nil, fmt.Errorf("failed to list menus: %w", err)
	}

	menuDTOs := make([]dto.MenuDTO, 0, len(menus))
	for _, m := range menus {
		menuDTOs = append(menuDTOs, dto.MenuDTO{
			ID:          m.ID,
			Name:        m.Name,
			Description: m.Description,
			SortOrder:   m.SortOrder,
			CreatedAt:   m.CreatedAt,
			UpdatedAt:   m.UpdatedAt,
		})
	}

	log.Printf("INFO: Successfully executed ListMenus query for DeviceID: %s. Found %d menus.", idToQuery, len(menuDTOs))
	return menuDTOs, nil
}
