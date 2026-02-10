package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/pwdutil"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
	ErrPasswordMismatch   = errors.New("password mismatch")
)

const (
	defaultCookieName = "gogw_jwt"
	tokenTTL          = 7 * 24 * time.Hour
)

// JWTManager 管理 JWT 签发与验证
// 采用标准 HS256(JWT) 格式，使用标准库实现，避免额外依赖。
type JWTManager struct {
	secret     []byte
	cookieName string
}

type sessionInfoContextKey struct{}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type jwtClaims struct {
	Subject   int64  `json:"sub"`
	Username  string `json:"name"`
	Role      string `json:"role"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

func NewJWTManager(secretKey []byte) *JWTManager {
	if len(secretKey) < 16 {
		secretKey = []byte("gogw-default-secret-please-change")
	}
	return &JWTManager{
		secret:     secretKey,
		cookieName: defaultCookieName,
	}
}

// Login 用户登录，返回 JWT
func (m *JWTManager) Login(w http.ResponseWriter, r *http.Request, username, password string) (string, error) {
	user, err := database.GetUserByUsername(username)
	if err != nil {
		return "", ErrInvalidCredentials
	}

	// 验证密码
	if !pwdutil.Compare(password, user.Password) {
		return "", ErrInvalidCredentials
	}

	token, err := m.GenerateToken(user)
	if err != nil {
		return "", err
	}
	m.setCookie(w, token)
	return token, nil
}

// Logout 用户登出
func (m *JWTManager) Logout(w http.ResponseWriter, r *http.Request) error {
	http.SetCookie(w, &http.Cookie{
		Name:     m.cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

// GetSession 获取会话信息（从 Authorization Bearer 或 Cookie）
func (m *JWTManager) GetSession(r *http.Request) (*SessionInfo, error) {
	tokenStr := extractToken(r, m.cookieName)
	if tokenStr == "" {
		return nil, nil
	}
	return m.ParseToken(tokenStr)
}

// RequireAuth 需要认证中间件
func (m *JWTManager) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info, err := m.GetSession(r)
		if err != nil || info == nil {
			// API 请求返回 401，页面请求跳转登录
			if strings.HasPrefix(r.URL.Path, "/api") {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
			} else {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
			}
			return
		}

		ctx := context.WithValue(r.Context(), sessionInfoContextKey{}, info)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin 需要管理员权限中间件
func (m *JWTManager) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info, err := m.GetSession(r)
		if err != nil || info == nil || info.Role != "admin" {
			if strings.HasPrefix(r.URL.Path, "/api") {
				http.Error(w, "forbidden", http.StatusForbidden)
			} else {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
			}
			return
		}

		ctx := context.WithValue(r.Context(), sessionInfoContextKey{}, info)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// SessionInfo 会话信息
type SessionInfo struct {
	UserID   int64
	Username string
	Role     string
}

// GenerateToken 签发 JWT(HS256)
func (m *JWTManager) GenerateToken(user *models.User) (string, error) {
	now := time.Now()
	claims := jwtClaims{
		Subject:   user.ID,
		Username:  user.Username,
		Role:      user.Role,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(tokenTTL).Unix(),
	}
	return signJWT(claims, m.secret)
}

// ParseToken 解析并验证 JWT
func (m *JWTManager) ParseToken(tokenStr string) (*SessionInfo, error) {
	claims, err := verifyJWT(tokenStr, m.secret, time.Now())
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	return &SessionInfo{
		UserID:   claims.Subject,
		Username: claims.Username,
		Role:     claims.Role,
	}, nil
}

func signJWT(claims jwtClaims, secret []byte) (string, error) {
	header := jwtHeader{Alg: "HS256", Typ: "JWT"}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	headerPart := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsPart := base64.RawURLEncoding.EncodeToString(claimsJSON)
	signingInput := headerPart + "." + claimsPart
	signature := signHS256(signingInput, secret)
	return signingInput + "." + signature, nil
}

func verifyJWT(token string, secret []byte, now time.Time) (jwtClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return jwtClaims{}, ErrInvalidCredentials
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return jwtClaims{}, ErrInvalidCredentials
	}
	var header jwtHeader
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return jwtClaims{}, ErrInvalidCredentials
	}
	if header.Alg != "HS256" || header.Typ != "JWT" {
		return jwtClaims{}, ErrInvalidCredentials
	}

	signatureGiven, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return jwtClaims{}, ErrInvalidCredentials
	}
	signingInput := parts[0] + "." + parts[1]
	signatureExpected := signHS256Bytes(signingInput, secret)
	if !hmac.Equal(signatureGiven, signatureExpected) {
		return jwtClaims{}, ErrInvalidCredentials
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return jwtClaims{}, ErrInvalidCredentials
	}
	var claims jwtClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return jwtClaims{}, ErrInvalidCredentials
	}

	if claims.ExpiresAt <= 0 || now.Unix() >= claims.ExpiresAt {
		return jwtClaims{}, ErrInvalidCredentials
	}

	return claims, nil
}

func signHS256(input string, secret []byte) string {
	return base64.RawURLEncoding.EncodeToString(signHS256Bytes(input, secret))
}

func signHS256Bytes(input string, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(input))
	return mac.Sum(nil)
}

func extractToken(r *http.Request, cookieName string) string {
	// Authorization: Bearer <token>
	authz := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
		return strings.TrimSpace(authz[7:])
	}
	if c, err := r.Cookie(cookieName); err == nil {
		return c.Value
	}
	return ""
}

func (m *JWTManager) setCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     m.cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(tokenTTL.Seconds()),
	})
}

// ChangePassword 修改密码
func ChangePassword(userID int64, oldPassword, newPassword string) error {
	user, err := database.GetUserByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	// 验证旧密码
	if !pwdutil.Compare(oldPassword, user.Password) {
		return ErrPasswordMismatch
	}

	// 更新密码
	user.Password = pwdutil.Hash(newPassword)
	return database.UpdateUser(user)
}

// ResetPassword 重置密码（管理员功能）
func ResetPassword(userID int64, newPassword string) error {
	user, err := database.GetUserByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	user.Password = pwdutil.Hash(newPassword)
	return database.UpdateUser(user)
}

func SessionFromContext(ctx context.Context) *SessionInfo {
	if ctx == nil {
		return nil
	}
	info, _ := ctx.Value(sessionInfoContextKey{}).(*SessionInfo)
	return info
}
