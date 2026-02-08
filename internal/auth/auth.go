package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
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
type JWTManager struct {
	secret     []byte
	cookieName string
}

type sessionInfoContextKey struct{}

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

// GenerateToken 签发 JWT
func (m *JWTManager) GenerateToken(user *models.User) (string, error) {
	claims := jwt.MapClaims{
		"sub":  user.ID,
		"name": user.Username,
		"role": user.Role,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(tokenTTL).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// ParseToken 解析并验证 JWT
func (m *JWTManager) ParseToken(tokenStr string) (*SessionInfo, error) {
	parsed, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidCredentials
		}
		return m.secret, nil
	})
	if err != nil || !parsed.Valid {
		return nil, ErrInvalidCredentials
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidCredentials
	}
	id, _ := claims["sub"].(float64)
	username, _ := claims["name"].(string)
	role, _ := claims["role"].(string)
	return &SessionInfo{
		UserID:   int64(id),
		Username: username,
		Role:     role,
	}, nil
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
