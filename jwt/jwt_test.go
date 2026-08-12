package jwt_test

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/bitdlv/gokit/jwt"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHS256_RoundTrip(t *testing.T) {
	tok, err := jwt.GenerateHS256("s3cr3t", time.Now().Unix(), 3600,
		jwt.K("userId", "1001"), jwt.K("userPhone", "13800000000"))
	require.NoError(t, err)
	c, err := jwt.ValidateHS256(tok, "s3cr3t")
	require.NoError(t, err)
	assert.Equal(t, "1001", c.UserID)
	assert.Equal(t, "13800000000", c.Phone)
}

func TestHS256_WrongSecret(t *testing.T) {
	tok, err := jwt.GenerateHS256("s3cr3t", time.Now().Unix(), 3600)
	require.NoError(t, err)
	_, err = jwt.ValidateHS256(tok, "wrong")
	assert.Error(t, err)
}

func TestHS256_Expired(t *testing.T) {
	tok, err := jwt.GenerateHS256("s3cr3t", time.Now().Unix()-7200, 3600) // exp = -1h
	require.NoError(t, err)
	_, err = jwt.ValidateHS256(tok, "s3cr3t")
	assert.Error(t, err)
}

func TestAlgorithmMismatch_HSvsRS(t *testing.T) {
	// 用 HS256 签发，声明 RS256 校验 → 家族不匹配，必须拒绝（防御家族混淆攻击）。
	tok, err := jwt.GenerateHS256("s3cr3t", time.Now().Unix(), 3600)
	require.NoError(t, err)

	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	_, err = jwt.Parse(tok, jwt.Config{Algorithm: jwt.RS256, PublicKey: &pk.PublicKey})
	assert.ErrorIs(t, err, jwt.ErrInvalidToken)
}

func TestRS256_RoundTrip(t *testing.T) {
	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	claims := &jwt.StandardClaims{
		UserID: "u1",
		RegisteredClaims: jwtv5.RegisteredClaims{
			ExpiresAt: jwtv5.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	tok, err := jwt.Sign(jwt.Config{Algorithm: jwt.RS256, PrivateKey: pk}, claims)
	require.NoError(t, err)
	got, err := jwt.ParseStandard(tok, jwt.Config{Algorithm: jwt.RS256, PublicKey: &pk.PublicKey})
	require.NoError(t, err)
	assert.Equal(t, "u1", got.UserID)
}

func TestES256_RoundTrip(t *testing.T) {
	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tok, err := jwt.Sign(jwt.Config{Algorithm: jwt.ES256, PrivateKey: pk}, jwtv5.MapClaims{
		"userId": "u2",
		"exp":    time.Now().Add(time.Hour).Unix(),
	})
	require.NoError(t, err)
	m, err := jwt.Parse(tok, jwt.Config{Algorithm: jwt.ES256, PublicKey: &pk.PublicKey})
	require.NoError(t, err)
	assert.Equal(t, "u2", m["userId"])
}

func TestEdDSA_RoundTrip(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	tok, err := jwt.Sign(jwt.Config{Algorithm: jwt.EdDSA, PrivateKey: priv}, jwtv5.MapClaims{
		"userId": "u3",
		"exp":    time.Now().Add(time.Hour).Unix(),
	})
	require.NoError(t, err)
	m, err := jwt.Parse(tok, jwt.Config{Algorithm: jwt.EdDSA, PublicKey: pub})
	require.NoError(t, err)
	assert.Equal(t, "u3", m["userId"])
}

func TestUnsupportedAlgorithm(t *testing.T) {
	_, err := jwt.Sign(jwt.Config{Algorithm: "none", Secret: []byte("x")}, jwtv5.MapClaims{})
	assert.ErrorIs(t, err, jwt.ErrUnsupportedAlgorithm)
}

func TestMissingKey(t *testing.T) {
	_, err := jwt.Sign(jwt.Config{Algorithm: jwt.HS256}, jwtv5.MapClaims{})
	assert.ErrorIs(t, err, jwt.ErrMissingKey)
}
