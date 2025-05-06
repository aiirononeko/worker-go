package query

import (
	"context"
	"fmt"
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
	// コンテキストから認証情報を取得
	uid, ok := ctx.Value(middleware.UIDKey).(string)
	if !ok || uid == "" {
		// UIDが取得できない場合は認証エラーとして扱う (通常は認証ミドルウェアでブロックされるはず)
		return nil, fmt.Errorf("authentication required: UID not found in context")
	}

	// UIDから deviceID を抽出 (例: "device:xxxx" -> "xxxx")
	// ここでは単純なプレフィックス除去。user: も考慮する場合はより詳細なパースが必要。
	var idToQuery string
	if strings.HasPrefix(uid, "device:") {
		idToQuery = strings.TrimPrefix(uid, "device:")
	} else if strings.HasPrefix(uid, "user:") {
		// 現状 menus テーブルは device_id しか持たないため、user: の場合はエラーとするか、
		// devices テーブルを引いて関連する device_id を全て取得するなどの処理が必要。
		// ここでは、user:xxx の場合はまだサポートされていないエラーとする。
		return nil, fmt.Errorf("listing menus by user_id is not yet supported, use device_id")
	} else {
		return nil, fmt.Errorf("invalid UID format in context: %s", uid)
	}

	if idToQuery == "" {
		return nil, fmt.Errorf("failed to extract actual ID from UID: %s", uid)
	}

	// 1. リポジトリを呼び出してドメインオブジェクトを取得 (メソッド名と引数を変更)
	menus, err := s.menuRepo.ListMenusByDeviceId(ctx, idToQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to list menus: %w", err)
	}

	// 2. ドメインオブジェクトを DTO に変換
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

	return menuDTOs, nil
}
