package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

	user := &models.User{ID: 1, Username: "bad", Role: "user"}
	token, err := m.GenerateToken(user)
	if err != nil {
		t.Fatalf("GenerateToken error: %v", err)
	}

	// 篡改签名，应该被拒绝
	tampered := token[:len(token)-1] + "x"
	if _, err := m.ParseToken(tampered); err == nil {
		t.Fatalf("expected ParseToken to fail for tampered signature")
	}
}

func TestJWTManager_ParseToken_InvalidSigningMethod(t *testing.T) {
	m := NewJWTManager([]byte("secret-key-123456"))

	claims := jwtClaims{
		Subject:   1,
		Username:  "bad",
		Role:      "user",
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}
	token := buildTokenForTest(jwtHeader{Alg: "RS256", Typ: "JWT"}, claims, m.secret)

	if _, err := m.ParseToken(token); err == nil {
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

	claims, err := verifyJWT(token, m.secret, time.Now())
	if err != nil {
		t.Fatalf("verifyJWT error: %v", err)
	}
	if claims.ExpiresAt <= claims.IssuedAt {
		t.Fatalf("invalid claims exp=%d iat=%d", claims.ExpiresAt, claims.IssuedAt)
	}

	expTime := time.Unix(claims.ExpiresAt, 0)
	delta := time.Until(expTime)
	if delta <= 0 {
		t.Fatalf("token already expired")
	}
	if delta < tokenTTL-time.Minute || delta > tokenTTL+time.Minute {
		t.Fatalf("token ttl drift too large: got %v, want around %v", delta, tokenTTL)
	}
}

func TestJWTManager_ParseToken_Expired(t *testing.T) {
	m := NewJWTManager([]byte("expired-secret-key-123"))
	claims := jwtClaims{
		Subject:   1,
		Username:  "u",
		Role:      "r",
		IssuedAt:  time.Now().Add(-2 * time.Hour).Unix(),
		ExpiresAt: time.Now().Add(-time.Hour).Unix(),
	}
	token := buildTokenForTest(jwtHeader{Alg: "HS256", Typ: "JWT"}, claims, m.secret)

	if _, err := m.ParseToken(token); err == nil {
		t.Fatalf("expected expired token to be rejected")
	}
}

func TestSessionFromContext(t *testing.T) {
	if info := SessionFromContext(nil); info != nil {
		t.Fatalf("expected nil from nil context")
	}

	ctx := context.Background()
	if info := SessionFromContext(ctx); info != nil {
		t.Fatalf("expected nil when no session in context")
	}

	want := &SessionInfo{UserID: 9, Username: "admin", Role: "admin"}
	ctx = context.WithValue(ctx, sessionInfoContextKey{}, want)
	got := SessionFromContext(ctx)
	if got == nil || got.UserID != 9 || got.Username != "admin" || got.Role != "admin" {
		t.Fatalf("unexpected session info from context: %+v", got)
	}
}

func buildTokenForTest(header jwtHeader, claims jwtClaims, secret []byte) string {
	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)
	headerPart := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsPart := base64.RawURLEncoding.EncodeToString(claimsJSON)
	signingInput := headerPart + "." + claimsPart
	signature := signHS256(signingInput, secret)
	return signingInput + "." + signature
}
