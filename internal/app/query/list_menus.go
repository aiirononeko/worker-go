package query

import (
	"context"

	"github.com/aiirononeko/bulktrack-api/internal/app/dto"
	"github.com/aiirononeko/bulktrack-api/internal/domain/menu" // Domain レイヤーのリポジトリIFとエンティティ
)

// ListMenusQueryService はメニュー一覧取得ユースケースのインターフェースです。
type ListMenusQueryService interface {
	Execute(ctx context.Context, userID string) ([]dto.MenuDTO, error)
}

// listMenusQueryServiceImpl は ListMenusQueryService の実装です。
type listMenusQueryServiceImpl struct {
	menuRepo menu.MenuRepository // MenuRepository インターフェースに依存
}

// NewListMenusQueryService は ListMenusQueryService の新しいインスタンスを生成します。
func NewListMenusQueryService(mr menu.MenuRepository) ListMenusQueryService {
	return &listMenusQueryServiceImpl{menuRepo: mr}
}

// Execute はメニュー一覧取得のユースケースを実行します。
func (s *listMenusQueryServiceImpl) Execute(ctx context.Context, userID string) ([]dto.MenuDTO, error) {
	// 1. リポジトリを呼び出してドメインオブジェクトを取得
	menus, err := s.menuRepo.ListMenusByUserId(ctx, userID)
	if err != nil {
		// エラーログ、エラーラップなど
		return nil, err // TODO: アプリケーションエラーに変換
	}

	// 2. ドメインオブジェクトを DTO に変換
	menuDTOs := make([]dto.MenuDTO, 0, len(menus))
	for _, m := range menus {
		menuDTOs = append(menuDTOs, dto.MenuDTO{
			ID:          m.ID,
			Name:        m.Name,
			Description: m.Description, // ドメインの型が *string ならそのまま代入
			SortOrder:   m.SortOrder,
			CreatedAt:   m.CreatedAt,
			UpdatedAt:   m.UpdatedAt,
		})
	}

	return menuDTOs, nil
}
