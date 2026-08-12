// Package jwt 提供通用的 JWT 生成与校验能力，支持 HMAC / RSA / ECDSA / EdDSA 多种常见算法。
//
// 设计目标：
//   - 一套 API 覆盖对称（HS256/384/512）与非对称（RS/PS/ES/EdDSA）算法；
//   - 通过 Config.Algorithm 声明算法，Sign 时严格校验密钥类型与算法家族匹配；
//   - Parse 阶段强制校验 token 的 alg header 属于允许的家族，防御 alg=none / 家族混淆攻击；
//   - 提供 StandardClaims 作为 idx 项目 CustomClaims 的等价物，方便迁移。
//
// 使用示例：
//
//	// HS256 生成
//	tok, _ := jwt.Sign(jwt.Config{Algorithm: jwt.HS256, Secret: []byte("s3cr3t")},
//	    &jwt.StandardClaims{UserID: "1001", Phone: "13800000000",
//	        RegisteredClaims: jwtv5.RegisteredClaims{ExpiresAt: jwtv5.NewNumericDate(time.Now().Add(time.Hour))}})
//
//	// HS256 校验
//	claims, err := jwt.Parse(tok, jwt.Config{Algorithm: jwt.HS256, Secret: []byte("s3cr3t")})
//
//	// 快捷函数（兼容 idx.NewJwtToken / ValidateToken 语义）
//	tok, _ := jwt.GenerateHS256("s3cr3t", time.Now().Unix(), 3600,
//	    jwt.KV("userId", "1001"), jwt.KV("userPhone", "13800000000"))
//	claims, err := jwt.ValidateHS256(tok, "s3cr3t")
package jwt

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"errors"
	"fmt"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
)

// Algorithm 枚举支持的签名算法。
type Algorithm string

const (
	HS256 Algorithm = "HS256"
	HS384 Algorithm = "HS384"
	HS512 Algorithm = "HS512"

	RS256 Algorithm = "RS256"
	RS384 Algorithm = "RS384"
	RS512 Algorithm = "RS512"

	PS256 Algorithm = "PS256"
	PS384 Algorithm = "PS384"
	PS512 Algorithm = "PS512"

	ES256 Algorithm = "ES256"
	ES384 Algorithm = "ES384"
	ES512 Algorithm = "ES512"

	EdDSA Algorithm = "EdDSA"
)

// Family 返回算法家族，用于 Parse 阶段强制校验 token 的 alg header 与配置一致。
func (a Algorithm) Family() string {
	switch a {
	case HS256, HS384, HS512:
		return "HMAC"
	case RS256, RS384, RS512:
		return "RSA"
	case PS256, PS384, PS512:
		return "RSA-PSS"
	case ES256, ES384, ES512:
		return "ECDSA"
	case EdDSA:
		return "EdDSA"
	}
	return ""
}

// method 将 Algorithm 映射为 jwtv5.SigningMethod；不支持的算法返回 nil。
func (a Algorithm) method() jwtv5.SigningMethod {
	return jwtv5.GetSigningMethod(string(a))
}

// Config 声明签名与校验所需的算法与密钥。
//
// 对称算法（HS*）：仅需 Secret（[]byte 或 string，Sign/Parse 通用）；
// 非对称算法：Sign 用 PrivateKey，Parse 用 PublicKey。
type Config struct {
	Algorithm  Algorithm
	Secret     []byte           // HS* 共享密钥
	PrivateKey crypto.PrivateKey // 非对称签名密钥（*rsa.PrivateKey / *ecdsa.PrivateKey / ed25519.PrivateKey）
	PublicKey  crypto.PublicKey  // 非对称校验密钥（*rsa.PublicKey / *ecdsa.PublicKey / ed25519.PublicKey）
}

// StandardClaims 是 idx 项目 CustomClaims 的等价物，兼容既有 header 契约。
type StandardClaims struct {
	UserID string `json:"userId"`
	Phone  string `json:"userPhone"`
	jwtv5.RegisteredClaims
}

// KV 用于 GenerateHS* 快捷函数构建自定义 claim。
type KV struct {
	Key string
	Val any
}

// K 构造 KV 的语法糖。
func K(key string, val any) KV { return KV{Key: key, Val: val} }

// ─────────────────────── Sign / Parse ────────────────────────────────────────

var (
	ErrUnsupportedAlgorithm = errors.New("jwt: unsupported algorithm")
	ErrMissingKey           = errors.New("jwt: missing signing key")
	ErrInvalidKeyType       = errors.New("jwt: invalid key type for algorithm")
	ErrAlgorithmMismatch    = errors.New("jwt: token algorithm does not match expected family")
	ErrInvalidToken         = errors.New("jwt: invalid or expired token")
)

// Sign 根据 cfg 生成 token 字符串。
func Sign(cfg Config, claims jwtv5.Claims) (string, error) {
	m := cfg.Algorithm.method()
	if m == nil {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedAlgorithm, cfg.Algorithm)
	}
	key, err := signKey(cfg)
	if err != nil {
		return "", err
	}
	tok := jwtv5.NewWithClaims(m, claims)
	return tok.SignedString(key)
}

// Parse 校验 token 并把 payload 解析进一个新的 MapClaims。
// 更常见的用法是 ParseInto（直接解析进具体的 claims 结构体）。
func Parse(tokenStr string, cfg Config) (jwtv5.MapClaims, error) {
	claims := jwtv5.MapClaims{}
	if _, err := parseInto(tokenStr, cfg, claims); err != nil {
		return nil, err
	}
	return claims, nil
}

// ParseInto 将 token payload 解析进用户提供的 claims 结构体。
func ParseInto(tokenStr string, cfg Config, out jwtv5.Claims) error {
	_, err := parseInto(tokenStr, cfg, out)
	return err
}

// ParseStandard 校验并返回 StandardClaims（等价于 idx.ValidateToken）。
func ParseStandard(tokenStr string, cfg Config) (*StandardClaims, error) {
	c := &StandardClaims{}
	if err := ParseInto(tokenStr, cfg, c); err != nil {
		return nil, err
	}
	return c, nil
}

func parseInto(tokenStr string, cfg Config, out jwtv5.Claims) (*jwtv5.Token, error) {
	if cfg.Algorithm == "" {
		return nil, fmt.Errorf("%w: empty algorithm", ErrUnsupportedAlgorithm)
	}
	expectedFamily := cfg.Algorithm.Family()
	if expectedFamily == "" {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedAlgorithm, cfg.Algorithm)
	}

	token, err := jwtv5.ParseWithClaims(tokenStr, out, func(t *jwtv5.Token) (any, error) {
		// 严格校验 token header 中的 alg 与配置声明一致（家族级），防御 alg=none / 家族混淆。
		if got := Algorithm(t.Method.Alg()).Family(); got != expectedFamily {
			return nil, fmt.Errorf("%w: got %s, expected family %s", ErrAlgorithmMismatch, t.Method.Alg(), expectedFamily)
		}
		return verifyKey(cfg)
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if !token.Valid {
		return nil, ErrInvalidToken
	}
	return token, nil
}

func signKey(cfg Config) (any, error) {
	switch cfg.Algorithm.Family() {
	case "HMAC":
		if len(cfg.Secret) == 0 {
			return nil, ErrMissingKey
		}
		return cfg.Secret, nil
	case "RSA", "RSA-PSS":
		if cfg.PrivateKey == nil {
			return nil, ErrMissingKey
		}
		if _, ok := cfg.PrivateKey.(*rsa.PrivateKey); !ok {
			return nil, ErrInvalidKeyType
		}
		return cfg.PrivateKey, nil
	case "ECDSA":
		if cfg.PrivateKey == nil {
			return nil, ErrMissingKey
		}
		if _, ok := cfg.PrivateKey.(*ecdsa.PrivateKey); !ok {
			return nil, ErrInvalidKeyType
		}
		return cfg.PrivateKey, nil
	case "EdDSA":
		if cfg.PrivateKey == nil {
			return nil, ErrMissingKey
		}
		if _, ok := cfg.PrivateKey.(ed25519.PrivateKey); !ok {
			return nil, ErrInvalidKeyType
		}
		return cfg.PrivateKey, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrUnsupportedAlgorithm, cfg.Algorithm)
}

func verifyKey(cfg Config) (any, error) {
	switch cfg.Algorithm.Family() {
	case "HMAC":
		if len(cfg.Secret) == 0 {
			return nil, ErrMissingKey
		}
		return cfg.Secret, nil
	case "RSA", "RSA-PSS":
		if cfg.PublicKey == nil {
			return nil, ErrMissingKey
		}
		if _, ok := cfg.PublicKey.(*rsa.PublicKey); !ok {
			return nil, ErrInvalidKeyType
		}
		return cfg.PublicKey, nil
	case "ECDSA":
		if cfg.PublicKey == nil {
			return nil, ErrMissingKey
		}
		if _, ok := cfg.PublicKey.(*ecdsa.PublicKey); !ok {
			return nil, ErrInvalidKeyType
		}
		return cfg.PublicKey, nil
	case "EdDSA":
		if cfg.PublicKey == nil {
			return nil, ErrMissingKey
		}
		if _, ok := cfg.PublicKey.(ed25519.PublicKey); !ok {
			return nil, ErrInvalidKeyType
		}
		return cfg.PublicKey, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrUnsupportedAlgorithm, cfg.Algorithm)
}

// ─────────────────────── HS256 兼容层（对应 idx.NewJwtToken/ValidateToken）─────

// GenerateHS256 复刻 idx 项目 middlewares.NewJwtToken 的行为：
// iat 为签发时间（秒），ttlSeconds 为有效期（秒），extras 追加到 MapClaims。
func GenerateHS256(secret string, iat, ttlSeconds int64, extras ...KV) (string, error) {
	claims := jwtv5.MapClaims{
		"iat": iat,
		"exp": iat + ttlSeconds,
	}
	for _, kv := range extras {
		claims[kv.Key] = kv.Val
	}
	return Sign(Config{Algorithm: HS256, Secret: []byte(secret)}, claims)
}

// ValidateHS256 复刻 idx 项目 middlewares.ValidateToken 的行为，返回 StandardClaims。
func ValidateHS256(tokenStr, secret string) (*StandardClaims, error) {
	return ParseStandard(tokenStr, Config{Algorithm: HS256, Secret: []byte(secret)})
}

// Now 便于测试注入时间；生产使用 time.Now()。保留为包级变量以支持覆写。
var Now = func() time.Time { return time.Now() }
