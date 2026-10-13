package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Jwt 是 JWT 签发与解析器。签名算法固定为 HS256。
type Jwt[T any] struct {
	Issuer     string
	SigningKey []byte
}

// NewJwt 创建 Jwt 实例
func NewJwt[T any](cfg Jwt[T]) *Jwt[T] {
	return &Jwt[T]{
		Issuer:     cfg.Issuer,
		SigningKey: cfg.SigningKey,
	}
}

// parseOptions 返回解析时统一使用的校验选项。
//
// 必须用 WithValidMethods 固定签名算法：keyfunc 只返回 SigningKey、不看
// token.Method，不锁定算法的话 HS384/HS512 等同族算法签发的 token 也会被接受。
// Issuer 非空时一并校验，避免接受其它签发方签发的 token。
func (j *Jwt[T]) parseOptions() []jwt.ParserOption {
	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	}
	if j.Issuer != "" {
		opts = append(opts, jwt.WithIssuer(j.Issuer))
	}
	return opts
}

// 签发token
func (j *Jwt[T]) GenerateJwtToken(subject string, audience []string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    j.Issuer,                         // 签名的发行者
		Subject:   subject,                          // 主题
		Audience:  audience,                         // 受众
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)), // 到期时间
		NotBefore: jwt.NewNumericDate(now),          // 不早于 声明标识了 JWT 必须被接受处理的时间
		IssuedAt:  jwt.NewNumericDate(now),          // 签发时间
		ID:        uuid.New().String(),              // 唯一标识
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.SigningKey)
}

// 解析token
func (j *Jwt[T]) ParseJwtToken(tokenString string) (string, []string, error) {
	jwtToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return j.SigningKey, nil
	}, j.parseOptions()...)
	if err != nil {
		return "", nil, err
	}
	if jwtToken == nil {
		return "", nil, errors.New("jwt token nil")
	}
	if !jwtToken.Valid {
		return "", nil, errors.New("jwt token invalid")
	}

	subject, err := jwtToken.Claims.GetSubject()
	if err != nil {
		return "", nil, errors.New("jwt token subject error")
	}
	audience, err := jwtToken.Claims.GetAudience()
	if err != nil {
		return "", nil, errors.New("jwt token audience error")
	}
	return subject, audience, nil
}

// 获取默认的注册声明
func (j *Jwt[T]) GetRegisteredClaimsDefault(ttl time.Duration) *jwt.RegisteredClaims {
	now := time.Now()
	return &jwt.RegisteredClaims{
		Issuer:    j.Issuer,                         // 签名的发行者
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)), // 到期时间
		NotBefore: jwt.NewNumericDate(now),          // 不早于 声明标识了 JWT 必须被接受处理的时间
		IssuedAt:  jwt.NewNumericDate(now),          // 签发时间
		ID:        uuid.New().String(),              // 唯一标识
	}
}

// 通过claims签发token
func (j *Jwt[T]) GenerateJwtTokenWithClaims(claims *T) (string, error) {
	jwtClaims, ok := any(claims).(jwt.Claims)
	if !ok {
		return "", errors.New("jwt token claims assert failed")
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims)
	return token.SignedString(j.SigningKey)
}

// 通过claims解析token
func (j *Jwt[T]) ParseJwtTokenWithClaims(tokenString string, claims *T) (jwt.Claims, error) {
	jwtClaims, ok := any(claims).(jwt.Claims)
	if !ok {
		return nil, errors.New("jwt token claims assert failed")
	}
	token, err := jwt.ParseWithClaims(tokenString, jwtClaims, func(token *jwt.Token) (interface{}, error) {
		return j.SigningKey, nil
	}, j.parseOptions()...)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("jwt token invalid")
	}
	return token.Claims, nil
}
