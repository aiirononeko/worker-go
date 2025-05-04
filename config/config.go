package config

import (
	"fmt" // log パッケージを import
	"strconv"
	"time"

	"github.com/joho/godotenv" // .env ファイル読み込み用 (オプション)

	"github.com/aiirononeko/bulktrack-api/internal/platform/env" // プラットフォーム固有の env 読み込みを追加
)

// Config はアプリケーション設定を保持します。
type Config struct {
	Env              string        // 環境 (development, production など)
	JWTPrivateKeyPEM string        // EdDSA/ES256 秘密鍵 (PEM形式)
	JWTPublicKeyPEM  string        // EdDSA/ES256 公開鍵 (PEM形式)
	AccessTokenTTL   time.Duration // アクセストークンの有効期間
	RefreshTokenTTL  time.Duration // リフレッシュトークンの有効期間
	// D1 またはローカル SQLite の DSN (Data Source Name)
	// Workers 環境では Binding 名が使われることが多い
	DatabaseDSN string
}

// Load は環境変数 (.env ファイルを含む) から設定を読み込みます。
func Load() (*Config, error) {
	// .env ファイルを読み込む (開発環境用)
	// godotenv は os パッケージに影響を与えるため、プラットフォームの Getenv より先に呼ぶ
	_ = godotenv.Load() // エラーは無視しても良い

	accessTokenTTLStr := getEnv("ACCESS_TOKEN_TTL_MINUTES", "15") // デフォルト15分
	refreshTokenTTLStr := getEnv("REFRESH_TOKEN_TTL_DAYS", "30")  // デフォルト30日

	accessTokenTTL, err := strconv.Atoi(accessTokenTTLStr)
	if err != nil {
		accessTokenTTL = 15 // パース失敗時はデフォルト値
	}

	refreshTokenTTL, err := strconv.Atoi(refreshTokenTTLStr)
	if err != nil {
		refreshTokenTTL = 30 // パース失敗時はデフォルト値
	}

	pemPrivKey := getEnv("JWT_PRIVATE_KEY", "") // wrangler secret から読み込む想定
	pemPubKey := getEnv("JWT_PUBLIC_KEY", "")   // wrangler secret から読み込む想定

	// ★★★ デバッグログ削除 ★★★
	// log.Printf("[Config] Loaded JWT_PRIVATE_KEY (length %d):\n%s", len(pemPrivKey), pemPrivKey)
	// log.Printf("[Config] Loaded JWT_PUBLIC_KEY (length %d):\n%s", len(pemPubKey), pemPubKey)
	// ★★★ ここまで ★★★

	cfg := &Config{
		Env:              getEnv("APP_ENV", "development"),
		JWTPrivateKeyPEM: pemPrivKey,
		JWTPublicKeyPEM:  pemPubKey,
		AccessTokenTTL:   time.Duration(accessTokenTTL) * time.Minute,
		RefreshTokenTTL:  time.Duration(refreshTokenTTL) * 24 * time.Hour,
		DatabaseDSN:      getEnv("DATABASE_DSN", "./bulktrack-local.db"), // ローカル開発用デフォルト
	}

	// 鍵は必須
	if cfg.JWTPrivateKeyPEM == "" {
		return nil, fmt.Errorf("environment variable JWT_PRIVATE_KEY is required")
	}
	if cfg.JWTPublicKeyPEM == "" {
		return nil, fmt.Errorf("environment variable JWT_PUBLIC_KEY is required")
	}

	return cfg, nil
}

// getEnv は環境変数を取得し、なければデフォルト値を返します。
// 内部で platform/env を使用し、ビルドターゲットに応じて適切な実装を呼び出します。
func getEnv(key, defaultValue string) string {
	value := env.Getenv(key) // platform/env の Getenv を使用
	if value == "" {
		return defaultValue
	}
	return value
}
