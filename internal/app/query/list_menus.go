package query

import (
	"context"
	"errors"
	"log/slog"

	"github.com/aiirononeko/bulktrack-api/internal/app/apperror"
	"github.com/aiirononeko/bulktrack-api/internal/app/dto"
	"github.com/aiirononeko/bulktrack-api/internal/domain/entity"
	"github.com/aiirononeko/bulktrack-api/internal/domain/menu"
)

// ListMenusQueryService はメニュー一覧取得ユースケースのインターフェースです。
type ListMenusQueryService interface {
	// Execute retrieves menus for a given device ID.
	// deviceID should be the pure UUID string, without any prefixes.
	Execute(ctx context.Context, deviceID entity.DeviceID) ([]dto.MenuDTO, error)
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
func (s *listMenusQueryServiceImpl) Execute(ctx context.Context, deviceID entity.DeviceID) ([]dto.MenuDTO, error) {
	slog.InfoContext(ctx, "Starting ListMenus query execution", slog.String("deviceID", deviceID.String()))

	if deviceID.IsZero() {
		slog.WarnContext(ctx, "DeviceID is empty in ListMenus query execution")
		return nil, apperror.NewErrBadRequest("DeviceID cannot be empty for ListMenus query", "")
	}

	slog.InfoContext(ctx, "Calling MenuRepository.ListMenusByDeviceId", slog.String("deviceID", deviceID.String()))
	menus, err := s.menuRepo.ListMenusByDeviceId(ctx, deviceID)
	if err != nil {
		var nfErr *apperror.ErrNotFound
		var internalErr *apperror.ErrInternal

		if errors.As(err, &nfErr) {
			slog.InfoContext(ctx, "No menus found for deviceID via repository", slog.String("deviceID", deviceID.String()), slog.Any("error", err))
			return nil, nfErr
		} else if errors.As(err, &internalErr) {
			slog.ErrorContext(ctx, "Internal error from menu repository while listing menus",
				slog.String("deviceID", deviceID.String()),
				slog.Any("error", err),
			)
			return nil, internalErr
		} else {
			slog.ErrorContext(ctx, "Unexpected error from menu repository while listing menus",
				slog.String("deviceID", deviceID.String()),
				slog.Any("original_error", err.Error()),
			)
			return nil, apperror.NewErrInternal("Failed to list menus due to an unexpected repository error", err)
		}
	}

	menuDTOs := make([]dto.MenuDTO, 0, len(menus))
	for _, m := range menus {
		menuDTOs = append(menuDTOs, dto.MenuDTO{
			ID:          m.ID.String(),
			Name:        m.Name,
			Description: m.Description,
			SortOrder:   m.SortOrder,
			CreatedAt:   m.CreatedAt,
			UpdatedAt:   m.UpdatedAt,
		})
	}

	slog.InfoContext(ctx, "Successfully executed ListMenus query",
		slog.String("deviceID", deviceID.String()),
		slog.Int("menu_count", len(menuDTOs)),
	)
	return menuDTOs, nil
}
