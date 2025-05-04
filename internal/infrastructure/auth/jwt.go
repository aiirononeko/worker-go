package auth

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"

	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	domainAuth "github.com/aiirononeko/bulktrack-api/internal/domain/auth"
)

// jwtService は JWTService の EdDSA (Ed25519) 実装です。
type jwtService struct {
	privateKey      crypto.PrivateKey // ed25519.PrivateKey
	publicKey       crypto.PublicKey  // ed25519.PublicKey
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// NewJWTService は PEM 形式 (改行なしも許容) のキー文字列から新しい jwtService を初期化します。
func NewJWTService(privKeyPEM, pubKeyPEM string, accessTokenTTL, refreshTokenTTL time.Duration) (domainAuth.JWTService, error) {

	// 秘密鍵のパース (改行なし PEM 対応)
	privKeyDataBase64 := extractBase64FromPEM(privKeyPEM, "PRIVATE KEY")
	if privKeyDataBase64 == "" {
		return nil, errors.New("failed to extract base64 data from private key PEM")
	}
	privKeyBytes, err := base64.StdEncoding.DecodeString(privKeyDataBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 private key data: %w", err)
	}
	privKey, err := x509.ParsePKCS8PrivateKey(privKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key from decoded data: %w", err)
	}
	if _, ok := privKey.(ed25519.PrivateKey); !ok {
		return nil, errors.New("private key is not an Ed25519 private key")
	}

	// 公開鍵のパース (改行なし PEM 対応)
	pubKeyDataBase64 := extractBase64FromPEM(pubKeyPEM, "PUBLIC KEY")
	if pubKeyDataBase64 == "" {
		return nil, errors.New("failed to extract base64 data from public key PEM")
	}
	pubKeyBytes, err := base64.StdEncoding.DecodeString(pubKeyDataBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 public key data: %w", err)
	}
	pubKey, err := x509.ParsePKIXPublicKey(pubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key from decoded data: %w", err)
	}
	if _, ok := pubKey.(ed25519.PublicKey); !ok {
		return nil, errors.New("public key is not an Ed25519 public key")
	}

	return &jwtService{
		privateKey:      privKey,
		publicKey:       pubKey,
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
	}, nil
}

// extractBase64FromPEM は PEM 風文字列 (改行なし許容) から Base64 データ部分を抽出します。
func extractBase64FromPEM(pemString, keyType string) string {
	prefix := fmt.Sprintf("-----BEGIN %s-----", keyType)
	suffix := fmt.Sprintf("-----END %s-----", keyType)

	trimmed := strings.TrimSpace(pemString)
	if !strings.HasPrefix(trimmed, prefix) || !strings.HasSuffix(trimmed, suffix) {
		return ""
	}

	base64Data := strings.TrimPrefix(trimmed, prefix)
	base64Data = strings.TrimSuffix(base64Data, suffix)
	base64Data = strings.ReplaceAll(base64Data, "\n", "") // 念のため改行も除去
	base64Data = strings.ReplaceAll(base64Data, "\r", "")
	base64Data = strings.TrimSpace(base64Data)

	return base64Data
}

// GenerateTokens は新しいアクセストークンとリフレッシュトークンを EdDSA で生成します。
// リフレッシュトークンには JTI を含めます。
func (s *jwtService) GenerateTokens(ctx context.Context, uid string) (string, string, string, time.Time, error) {
	// Access Token
	accessExpiresAt := time.Now().Add(s.accessTokenTTL)
	accessClaims := jwt.MapClaims{
		"uid": uid,
		"exp": jwt.NewNumericDate(accessExpiresAt),
		"iat": jwt.NewNumericDate(time.Now()),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodEdDSA, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.privateKey)
	if err != nil {
		return "", "", "", time.Time{}, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Refresh Token (with JTI)
	refreshExpiresAt := time.Now().Add(s.refreshTokenTTL)
	refreshTokenJTI := uuid.NewString() // JTI を生成
	refreshClaims := jwt.MapClaims{
		"uid": uid,
		"jti": refreshTokenJTI, // JTI をクレームに追加
		"exp": jwt.NewNumericDate(refreshExpiresAt),
		"iat": jwt.NewNumericDate(time.Now()),
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodEdDSA, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(s.privateKey)
	if err != nil {
		return "", "", "", time.Time{}, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return accessTokenString, refreshTokenString, refreshTokenJTI, accessExpiresAt, nil
}

// VerifyToken は EdDSA 署名されたトークンを検証し、クレーム情報を返します。
func (s *jwtService) VerifyToken(ctx context.Context, tokenString string) (*domainAuth.VerifiedTokenClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 署名アルゴリズムが EdDSA であることを確認
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		// 公開鍵を返す
		return s.publicKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			// TODO: ドメインエラー ErrTokenExpired に変換
			return nil, fmt.Errorf("token expired: %w", err)
		}
		// TODO: ドメインエラー ErrInvalidToken に変換
		return nil, fmt.Errorf("invalid token parse error: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		uid, uidOK := claims["uid"].(string)
		jti, jtiOK := claims["jti"].(string) // jti も取得・検証
		_ = jtiOK                            // linter の未使用エラーを回避 (意図的にチェックしないため)

		if !uidOK {
			// TODO: ドメインエラー ErrInvalidToken
			return nil, errors.New("invalid token: uid claim is missing or not a string")
		}
		// jti はリフレッシュトークン検証時に必須だが、アクセストークン検証時はなくても良い場合がある。
		// ここでは jti がなくてもエラーにしないが、呼び出し元で必要ならチェックする。

		return &domainAuth.VerifiedTokenClaims{
			UID: uid,
			JTI: jti, // jti がなくても空文字列が入る
		}, nil
	} else {
		// TODO: ドメインエラー ErrInvalidToken に変換
		return nil, errors.New("invalid token (claims parsing or validation failed)")
	}
}
