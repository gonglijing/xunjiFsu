package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/sessions"
	"gogw/internal/database"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
	ErrPasswordMismatch   = errors.New("password mismatch")
)

// SessionManager 会话管理器
type SessionManager struct {
	store     *sessions.CookieStore
	cookieName string
}

// NewSessionManager 创建会话管理器
func NewSessionManager(secretKey []byte) *SessionManager {
	store := sessions.NewCookieStore(secretKey)
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7天
		HttpOnly: true,
	}

	return &SessionManager{
		store:      store,
		cookieName: "gogw_session",
	}
}

// Login 用户登录
func (m *SessionManager) Login(w http.ResponseWriter, r *http.Request, username, password string) error {
	user, err := database.GetUserByUsername(username)
	if err != nil {
		return ErrInvalidCredentials
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return ErrInvalidCredentials
	}

	// 创建会话
	session, err := m.store.Get(r, m.cookieName)
	if err != nil {
		return err
	}

	session.Values["user_id"] = user.ID
	session.Values["username"] = user.Username
	session.Values["role"] = user.Role
	session.Values["login_time"] = time.Now().Unix()

	if err := m.store.Save(r, w, session); err != nil {
		return err
	}

	return nil
}

// Logout 用户登出
func (m *SessionManager) Logout(w http.ResponseWriter, r *http.Request) error {
	session, err := m.store.Get(r, m.cookieName)
	if err != nil {
		return err
	}

	session.Values = make(map[interface{}]interface{})
	return m.store.Save(r, w, session)
}

// GetSession 获取会话信息
func (m *SessionManager) GetSession(r *http.Request) (*SessionInfo, error) {
	session, err := m.store.Get(r, m.cookieName)
	if err != nil {
		return nil, err
	}

	userID, ok := session.Values["user_id"].(int64)
	if !ok {
		return nil, nil
	}

	username, _ := session.Values["username"].(string)
	role, _ := session.Values["role"].(string)

	return &SessionInfo{
		UserID:   userID,
		Username: username,
		Role:     role,
	}, nil
}

// RequireAuth 需要认证中间件
func (m *SessionManager) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := m.store.Get(r, m.cookieName)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		_, ok := session.Values["user_id"].(int64)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireAdmin 需要管理员权限中间件
func (m *SessionManager) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := m.store.Get(r, m.cookieName)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		role, ok := session.Values["role"].(string)
		if !ok || role != "admin" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// SessionInfo 会话信息
type SessionInfo struct {
	UserID   int64
	Username string
	Role     string
}

// hashPassword 密码哈希
func hashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return string(hash)
}

// ChangePassword 修改密码
func ChangePassword(userID int64, oldPassword, newPassword string) error {
	user, err := database.GetUserByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return ErrPasswordMismatch
	}

	// 更新密码
	user.Password = hashPassword(newPassword)
	return database.UpdateUser(user)
}

// ResetPassword 重置密码（管理员功能）
func ResetPassword(userID int64, newPassword string) error {
	user, err := database.GetUserByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	user.Password = hashPassword(newPassword)
	return database.UpdateUser(user)
}
