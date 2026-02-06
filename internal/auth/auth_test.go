package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestNewJWTManager_DefaultSecretFallback(t *testing.T) {
	m := NewJWTManager([]byte("short"))
	if string(m.secret) == "short" {
		t.Fatalf("expected short secret to be replaced with default")
	}
	if m.cookieName == "" {
		t.Fatalf("expected cookieName to be set")
	}
}

func TestJWTManager_GenerateAndParseToken(t *testing.T) {
	m := NewJWTManager([]byte("this-is-a-very-secret-key"))

	user := &models.User{
		ID:       42,
		Username: "test-user",
		Role:     "admin",
	}

	token, err := m.GenerateToken(user)
	if err != nil {
		t.Fatalf("GenerateToken error: %v", err)
	}
	if token == "" {
		t.Fatalf("expected non-empty token")
	}

	info, err := m.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken error: %v", err)
	}
	if info == nil {
		t.Fatalf("expected non-nil SessionInfo")
	}
	if info.UserID != user.ID || info.Username != user.Username || info.Role != user.Role {
		t.Fatalf("parsed session info mismatch: %+v", info)
	}
}

func TestJWTManager_ParseToken_Invalid(t *testing.T) {
	m := NewJWTManager([]byte("secret-key-123456"))

	// 使用错误算法构造一个 token，应该被拒绝
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub":  float64(1),
		"name": "bad",
	})
	signed, _ := token.SignedString([]byte("other"))

	if _, err := m.ParseToken(signed); err == nil {
		t.Fatalf("expected ParseToken to fail for invalid signing method")
	}
}

func TestExtractToken_FromAuthorizationHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer test-token-123")

	got := extractToken(req, defaultCookieName)
	if got != "test-token-123" {
		t.Fatalf("expected token from header, got %q", got)
	}
}

func TestExtractToken_FromCookie(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  defaultCookieName,
		Value: "cookie-token",
	})

	got := extractToken(req, defaultCookieName)
	if got != "cookie-token" {
		t.Fatalf("expected token from cookie, got %q", got)
	}
}

func TestJWTManager_SetCookieAndGetSession(t *testing.T) {
	m := NewJWTManager([]byte("another-secret-key-123"))
	user := &models.User{
		ID:       7,
		Username: "bob",
		Role:     "user",
	}
	token, err := m.GenerateToken(user)
	if err != nil {
		t.Fatalf("GenerateToken error: %v", err)
	}

	// 先通过 setCookie 写入响应，再把 cookie 搬到请求里
	rr := httptest.NewRecorder()
	m.setCookie(rr, token)
	resp := rr.Result()
	defer resp.Body.Close()

	var cookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == defaultCookieName {
			cookie = c
			break
		}
	}
	if cookie == nil {
		t.Fatalf("expected jwt cookie to be set")
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookie)

	info, err := m.GetSession(req)
	if err != nil {
		t.Fatalf("GetSession error: %v", err)
	}
	if info == nil {
		t.Fatalf("expected non-nil session")
	}
	if info.Username != user.Username {
		t.Fatalf("expected username %q, got %q", user.Username, info.Username)
	}
}

func TestJWTManager_RequireAuth_UnauthorizedAPI(t *testing.T) {
	m := NewJWTManager([]byte("secret-1234567890"))
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := m.RequireAuth(next)
	req := httptest.NewRequest("GET", "/api/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if called {
		t.Fatalf("next handler should not be called without session")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestJWTManager_RequireAdmin_ForbiddenForNonAdmin(t *testing.T) {
	m := NewJWTManager([]byte("secret-1234567890"))

	// 手工构造一个 user 角色的 token
	user := &models.User{
		ID:       1,
		Username: "user",
		Role:     "user",
	}
	token, err := m.GenerateToken(user)
	if err != nil {
		t.Fatalf("GenerateToken error: %v", err)
	}

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})
	handler := m.RequireAdmin(next)

	req := httptest.NewRequest("GET", "/api/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if called {
		t.Fatalf("next handler should not be called for non-admin user")
	}
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rr.Code)
	}
}

// 简单检查 token 过期字段设置是否大致正确（不过度依赖时间）
func TestJWTManager_TokenTTL(t *testing.T) {
	m := NewJWTManager([]byte("ttl-secret-key-123456"))
	user := &models.User{ID: 1, Username: "u", Role: "r"}
	token, err := m.GenerateToken(user)
	if err != nil {
		t.Fatalf("GenerateToken error: %v", err)
	}

	parsed, err := jwt.Parse(token, func(tk *jwt.Token) (interface{}, error) {
		return m.secret, nil
	})
	if err != nil || !parsed.Valid {
		t.Fatalf("token not valid: %v", err)
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatalf("unexpected claims type")
	}
	exp, ok := claims["exp"].(float64)
	if !ok {
		t.Fatalf("exp claim missing or wrong type")
	}
	expTime := time.Unix(int64(exp), 0)
	if time.Until(expTime) <= 0 {
		t.Fatalf("token already expired")
	}
}

