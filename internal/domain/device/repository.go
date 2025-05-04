package device

import "context"

// DeviceRepository は Device エンティティの永続化を抽象化するインターフェースです。
type DeviceRepository interface {
	// Save はデバイス情報を保存します。
	// 存在しない場合は新規作成、存在する場合は更新する責務を持ちます。
	// 実装によっては INSERT OR UPDATE や Find/Save の組み合わせになります。
	Save(ctx context.Context, device *Device) error

	// FindByID は指定された ID のデバイスを取得します。
	// 見つからない場合は、特定のドメインエラー (例: ErrDeviceNotFound) または nil とエラーを返すことが期待されます。
	FindByID(ctx context.Context, id string) (*Device, error)

	// TODO: 必要に応じて他のメソッド (例: FindByUserID) を追加
}

// TODO: ドメイン固有のエラー (例: ErrDeviceNotFound) を定義
