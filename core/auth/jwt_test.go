package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var testKey = []byte("test-signing-key")

type customClaims struct {
	jwt.RegisteredClaims
	Role string `json:"role"`
}

func TestGenerateAndParseJwtToken(t *testing.T) {
	j := NewJwt(Jwt[any]{Issuer: "tituskit", SigningKey: testKey})

	token, err := j.GenerateJwtToken("user-001", []string{"app"}, time.Hour)
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}

	subject, audience, err := j.ParseJwtToken(token)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if subject != "user-001" {
		t.Errorf("subject = %q, 期望 user-001", subject)
	}
	if len(audience) != 1 || audience[0] != "app" {
		t.Errorf("audience = %v, 期望 [app]", audience)
	}
}

func TestParseRejectsWrongKey(t *testing.T) {
	issuer := NewJwt(Jwt[any]{Issuer: "tituskit", SigningKey: testKey})
	token, err := issuer.GenerateJwtToken("user-001", nil, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	other := NewJwt(Jwt[any]{Issuer: "tituskit", SigningKey: []byte("another-key")})
	if _, _, err := other.ParseJwtToken(token); err == nil {
		t.Fatal("使用不同密钥应解析失败")
	}
}

func TestParseRejectsExpiredToken(t *testing.T) {
	j := NewJwt(Jwt[any]{Issuer: "tituskit", SigningKey: testKey})
	token, err := j.GenerateJwtToken("user-001", nil, -time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	if _, _, err := j.ParseJwtToken(token); !errors.Is(err, jwt.ErrTokenExpired) {
		t.Fatalf("期望 ErrTokenExpired, 实际 %v", err)
	}
}

// 解析必须锁定 HS256：keyfunc 只返回 SigningKey，不锁定算法时
// 任何 HMAC 家族算法（HS384/HS512）签发的 token 都会被接受。
func TestParseRejectsNonHS256Algorithms(t *testing.T) {
	j := NewJwt(Jwt[any]{Issuer: "tituskit", SigningKey: testKey})

	claims := jwt.RegisteredClaims{
		Issuer:    "tituskit",
		Subject:   "user-001",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}

	for _, method := range []jwt.SigningMethod{jwt.SigningMethodHS384, jwt.SigningMethodHS512} {
		t.Run(method.Alg(), func(t *testing.T) {
			token, err := jwt.NewWithClaims(method, claims).SignedString(testKey)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := j.ParseJwtToken(token); err == nil {
				t.Fatalf("%s 签发的 token 不应被接受", method.Alg())
			}
		})
	}
}

// 其它签发方签发的 token 不应被接受。
func TestParseRejectsForeignIssuer(t *testing.T) {
	j := NewJwt(Jwt[any]{Issuer: "tituskit", SigningKey: testKey})

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "evil",
		Subject:   "user-001",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}).SignedString(testKey)
	if err != nil {
		t.Fatal(err)
	}

	if _, _, err := j.ParseJwtToken(token); err == nil {
		t.Fatal("issuer 不匹配的 token 不应被接受")
	}
}

// Issuer 为空时不校验 issuer，只校验签名与算法。
func TestParseWithoutIssuer(t *testing.T) {
	j := NewJwt(Jwt[any]{SigningKey: testKey})

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   "user-001",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}).SignedString(testKey)
	if err != nil {
		t.Fatal(err)
	}

	subject, _, err := j.ParseJwtToken(token)
	if err != nil {
		t.Fatalf("未设置 Issuer 时不应校验 issuer: %v", err)
	}
	if subject != "user-001" {
		t.Errorf("subject = %q", subject)
	}
}

func TestCustomClaimsRoundTrip(t *testing.T) {
	j := NewJwt(Jwt[customClaims]{Issuer: "tituskit", SigningKey: testKey})

	claims := &customClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "tituskit",
			Subject:   "user-001",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		Role: "admin",
	}

	token, err := j.GenerateJwtTokenWithClaims(claims)
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}

	parsed, err := j.ParseJwtTokenWithClaims(token, &customClaims{})
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	got, ok := parsed.(*customClaims)
	if !ok {
		t.Fatalf("claims 类型 = %T, 期望 *customClaims", parsed)
	}
	if got.Role != "admin" {
		t.Errorf("Role = %q, 期望 admin", got.Role)
	}
}

// 不满足 jwt.Claims 接口的类型应返回错误而不是 panic。
func TestGenerateJwtTokenWithInvalidClaims(t *testing.T) {
	type notClaims struct{ Foo string }

	j := NewJwt(Jwt[notClaims]{SigningKey: testKey})

	if _, err := j.GenerateJwtTokenWithClaims(&notClaims{Foo: "bar"}); err == nil {
		t.Fatal("非 jwt.Claims 类型应返回错误")
	}
	if _, err := j.ParseJwtTokenWithClaims("whatever", &notClaims{}); err == nil {
		t.Fatal("非 jwt.Claims 类型应返回错误")
	}
}

func TestGetRegisteredClaimsDefault(t *testing.T) {
	j := NewJwt(Jwt[any]{Issuer: "tituskit", SigningKey: testKey})

	claims := j.GetRegisteredClaimsDefault(time.Hour)
	if claims.Issuer != "tituskit" {
		t.Errorf("Issuer = %q", claims.Issuer)
	}
	if claims.ID == "" {
		t.Error("ID 不应为空")
	}
	if claims.ExpiresAt == nil || !claims.ExpiresAt.After(time.Now()) {
		t.Error("ExpiresAt 应为未来时间")
	}
}

// 用 MapClaims 构造一个签名合法、但声明类型非法的 token。
func signMapClaims(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(testKey)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// sub 声明类型非法时应返回错误，而不是静默返回空 subject。
func TestParseJwtTokenMalformedSubject(t *testing.T) {
	j := NewJwt(Jwt[any]{Issuer: "tituskit", SigningKey: testKey})

	token := signMapClaims(t, jwt.MapClaims{
		"iss": "tituskit",
		"sub": 123, // 非字符串
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	_, _, err := j.ParseJwtToken(token)
	if err == nil {
		t.Fatal("sub 类型非法时应返回错误")
	}
	if err.Error() != "jwt token subject error" {
		t.Errorf("错误信息 = %q", err.Error())
	}
}

// aud 声明类型非法时应返回错误。
func TestParseJwtTokenMalformedAudience(t *testing.T) {
	j := NewJwt(Jwt[any]{Issuer: "tituskit", SigningKey: testKey})

	token := signMapClaims(t, jwt.MapClaims{
		"iss": "tituskit",
		"sub": "user-001",
		"aud": []any{"app", 123}, // 数组元素含非字符串
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	_, _, err := j.ParseJwtToken(token)
	if err == nil {
		t.Fatal("aud 类型非法时应返回错误")
	}
	if err.Error() != "jwt token audience error" {
		t.Errorf("错误信息 = %q", err.Error())
	}
}

// 缺少 sub 声明时 MapClaims 返回空字符串而不是错误，这里固定该行为。
func TestParseJwtTokenMissingSubject(t *testing.T) {
	j := NewJwt(Jwt[any]{Issuer: "tituskit", SigningKey: testKey})

	token := signMapClaims(t, jwt.MapClaims{
		"iss": "tituskit",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	subject, _, err := j.ParseJwtToken(token)
	if err != nil {
		t.Fatalf("缺少 sub 不应返回错误: %v", err)
	}
	if subject != "" {
		t.Errorf("subject = %q, 期望空字符串", subject)
	}
}

// 自定义 claims 的解析路径同样锁定 HS256。
func TestParseJwtTokenWithClaimsRejectsNonHS256(t *testing.T) {
	j := NewJwt(Jwt[customClaims]{Issuer: "tituskit", SigningKey: testKey})

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.RegisteredClaims{
		Issuer:    "tituskit",
		Subject:   "user-001",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}).SignedString(testKey)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := j.ParseJwtTokenWithClaims(token, &customClaims{}); err == nil {
		t.Fatal("HS512 签发的 token 不应被接受")
	}
}

// 自定义 claims 的解析路径同样校验 issuer。
func TestParseJwtTokenWithClaimsRejectsForeignIssuer(t *testing.T) {
	j := NewJwt(Jwt[customClaims]{Issuer: "tituskit", SigningKey: testKey})

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "evil",
		Subject:   "user-001",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}).SignedString(testKey)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := j.ParseJwtTokenWithClaims(token, &customClaims{}); err == nil {
		t.Fatal("issuer 不匹配的 token 不应被接受")
	}
}

// 自定义 claims 路径也应拒绝过期 token。
func TestParseJwtTokenWithClaimsRejectsExpired(t *testing.T) {
	j := NewJwt(Jwt[customClaims]{Issuer: "tituskit", SigningKey: testKey})

	claims := &customClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "tituskit",
			Subject:   "user-001",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}
	token, err := j.GenerateJwtTokenWithClaims(claims)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := j.ParseJwtTokenWithClaims(token, &customClaims{}); !errors.Is(err, jwt.ErrTokenExpired) {
		t.Fatalf("期望 ErrTokenExpired, 实际 %v", err)
	}
}

// 使用不同密钥签发的 token 在 claims 路径也应被拒绝。
func TestParseJwtTokenWithClaimsRejectsWrongKey(t *testing.T) {
	j := NewJwt(Jwt[customClaims]{Issuer: "tituskit", SigningKey: testKey})
	other := NewJwt(Jwt[customClaims]{Issuer: "tituskit", SigningKey: []byte("another-key")})

	token, err := other.GenerateJwtTokenWithClaims(&customClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "tituskit",
			Subject:   "user-001",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := j.ParseJwtTokenWithClaims(token, &customClaims{}); err == nil {
		t.Fatal("使用不同密钥应解析失败")
	}
}

// 自签发的 token 在 claims 路径应能正常往返。
func TestParseJwtTokenWithClaimsRoundTrip(t *testing.T) {
	j := NewJwt(Jwt[customClaims]{Issuer: "tituskit", SigningKey: testKey})

	claims := &customClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "tituskit",
			Subject:   "user-001",
			Audience:  jwt.ClaimStrings{"app"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		Role: "admin",
	}
	token, err := j.GenerateJwtTokenWithClaims(claims)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := j.ParseJwtTokenWithClaims(token, &customClaims{})
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if got := parsed.(*customClaims); got.Subject != "user-001" || got.Role != "admin" {
		t.Errorf("claims = %+v", got)
	}
}
