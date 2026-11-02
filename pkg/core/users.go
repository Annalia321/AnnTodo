package core

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"anntodo/pkg/models"
)

var (
	dummyOnce sync.Once
	dummyVal  []byte
)

// dummyHash is compared against when the user does not exist so that a
// failed login takes about as long as a wrong password.
func dummyHash() []byte {
	dummyOnce.Do(func() {
		dummyVal, _ = bcrypt.GenerateFromPassword([]byte("anntodo-dummy-password"), bcrypt.DefaultCost)
	})
	return dummyVal
}

// ---------- system state ----------

const (
	SysAllowRegister = "allow_register"
	SysSiteName      = "site_name"
)

type SystemSettings struct {
	AllowRegister bool   `json:"allow_register"`
	SiteName      string `json:"site_name"`
}

func (a *App) Initialized() (bool, error) {
	var n int64
	if err := a.DB.Model(&models.User{}).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

func (a *App) GetSystemSettings() (SystemSettings, error) {
	s := SystemSettings{SiteName: "AnnTodo"}
	var rows []models.SystemSetting
	if err := a.DB.Find(&rows).Error; err != nil {
		return s, err
	}
	for _, r := range rows {
		switch r.Key {
		case SysAllowRegister:
			s.AllowRegister = r.Value == "true"
		case SysSiteName:
			if r.Value != "" {
				s.SiteName = r.Value
			}
		}
	}
	return s, nil
}

func (a *App) UpdateSystemSettings(in SystemSettings) (SystemSettings, error) {
	name, err := cleanText(in.SiteName, 64, "site name", false)
	if err != nil {
		return SystemSettings{}, err
	}
	if name == "" {
		name = "AnnTodo"
	}
	allow := "false"
	if in.AllowRegister {
		allow = "true"
	}
	err = a.DB.Transaction(func(tx *gorm.DB) error {
		for k, v := range map[string]string{SysAllowRegister: allow, SysSiteName: name} {
			if err := tx.Save(&models.SystemSetting{Key: k, Value: v}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return SystemSettings{}, err
	}
	return a.GetSystemSettings()
}

// ---------- users ----------

type UserInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

func hashPassword(p string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	return string(h), err
}

func (a *App) createUser(tx *gorm.DB, in UserInput) (*models.User, error) {
	username, err := normalizeUsername(in.Username)
	if err != nil {
		return nil, err
	}
	if err := validatePassword(in.Password); err != nil {
		return nil, err
	}
	email, err := normalizeEmail(in.Email)
	if err != nil {
		return nil, err
	}
	nickname, err := cleanText(in.Nickname, 64, "nickname", false)
	if err != nil {
		return nil, err
	}
	if nickname == "" {
		nickname = username
	}
	role := in.Role
	if role != models.RoleAdmin {
		role = models.RoleUser
	}
	if err := a.checkUnique(tx, 0, username, email); err != nil {
		return nil, err
	}
	hash, err := hashPassword(in.Password)
	if err != nil {
		return nil, err
	}
	u := &models.User{Username: username, Email: email, Nickname: nickname, PasswordHash: hash, Role: role, Status: models.StatusActive}
	if err := tx.Create(u).Error; err != nil {
		return nil, err
	}
	return u, nil
}

func (a *App) checkUnique(tx *gorm.DB, selfID uint64, username string, email *string) error {
	var n int64
	if username != "" {
		if err := tx.Model(&models.User{}).Where("username = ? AND id <> ?", username, selfID).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return ErrUsernameTaken
		}
	}
	if email != nil {
		if err := tx.Model(&models.User{}).Where("email = ? AND id <> ?", *email, selfID).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return ErrEmailTaken
		}
	}
	return nil
}

// Setup creates the first administrator. It fails once any user exists.
func (a *App) Setup(in UserInput) (*models.User, error) {
	var user *models.User
	err := a.DB.Transaction(func(tx *gorm.DB) error {
		var n int64
		if err := tx.Model(&models.User{}).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return ErrAlreadyInitialized
		}
		in.Role = models.RoleAdmin
		u, err := a.createUser(tx, in)
		user = u
		return err
	})
	return user, err
}

// Register creates a normal user when self registration is enabled.
func (a *App) Register(in UserInput) (*models.User, error) {
	ok, err := a.Initialized()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotInitialized
	}
	sys, err := a.GetSystemSettings()
	if err != nil {
		return nil, err
	}
	if !sys.AllowRegister {
		return nil, ErrRegistrationClosed
	}
	in.Role = models.RoleUser
	var user *models.User
	err = a.DB.Transaction(func(tx *gorm.DB) error {
		u, err := a.createUser(tx, in)
		user = u
		return err
	})
	return user, err
}

// Authenticate checks a username (or email) and password.
func (a *App) Authenticate(login, password string) (*models.User, error) {
	login = strings.ToLower(strings.TrimSpace(login))
	if login == "" || password == "" {
		return nil, ErrInvalidCredentials
	}
	var u models.User
	q := a.DB.Where("username = ?", login)
	if strings.Contains(login, "@") {
		q = a.DB.Where("username = ? OR email = ?", login, login)
	}
	if err := q.First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Spend comparable time to make user enumeration harder.
			_ = bcrypt.CompareHashAndPassword(dummyHash(), []byte(password))
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return nil, ErrInvalidCredentials
	}
	if u.Status != models.StatusActive {
		return nil, ErrUserDisabled
	}
	return &u, nil
}

func (a *App) GetUser(id uint64) (*models.User, error) {
	var u models.User
	if err := a.DB.First(&u, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

type ProfileInput struct {
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
}

func (a *App) UpdateProfile(userID uint64, in ProfileInput) (*models.User, error) {
	nickname, err := cleanText(in.Nickname, 64, "nickname", true)
	if err != nil {
		return nil, err
	}
	email, err := normalizeEmail(in.Email)
	if err != nil {
		return nil, err
	}
	if err := a.checkUnique(a.DB, userID, "", email); err != nil {
		return nil, err
	}
	if err := a.DB.Model(&models.User{}).Where("id = ?", userID).
		Updates(map[string]any{"nickname": nickname, "email": email, "updated_at": now()}).Error; err != nil {
		return nil, err
	}
	return a.GetUser(userID)
}

// ChangePassword verifies the old password, stores the new one and signs out
// every other session of the user.
func (a *App) ChangePassword(userID uint64, currentSession, oldPw, newPw string) error {
	u, err := a.GetUser(userID)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(oldPw)) != nil {
		return ErrWrongPassword
	}
	if err := validatePassword(newPw); err != nil {
		return err
	}
	hash, err := hashPassword(newPw)
	if err != nil {
		return err
	}
	return a.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.User{}).Where("id = ?", userID).
			Updates(map[string]any{"password_hash": hash, "updated_at": now()}).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ? AND id <> ?", userID, currentSession).Delete(&models.Session{}).Error
	})
}

func (a *App) SetAvatar(userID uint64, file string) error {
	return a.DB.Model(&models.User{}).Where("id = ?", userID).
		Updates(map[string]any{"avatar": file, "updated_at": now()}).Error
}

// ---------- admin: user management ----------

func (a *App) ListUsers() ([]models.User, error) {
	var users []models.User
	err := a.DB.Order("id").Find(&users).Error
	return users, err
}

func (a *App) AdminCreateUser(in UserInput) (*models.User, error) {
	var user *models.User
	err := a.DB.Transaction(func(tx *gorm.DB) error {
		u, err := a.createUser(tx, in)
		user = u
		return err
	})
	return user, err
}

type AdminUserUpdate struct {
	Nickname *string `json:"nickname"`
	Email    *string `json:"email"`
	Role     *string `json:"role"`
	Status   *string `json:"status"`
}

func (a *App) AdminUpdateUser(actorID, userID uint64, in AdminUserUpdate) (*models.User, error) {
	u, err := a.GetUser(userID)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{"updated_at": now()}
	if in.Nickname != nil {
		n, err := cleanText(*in.Nickname, 64, "nickname", true)
		if err != nil {
			return nil, err
		}
		updates["nickname"] = n
	}
	if in.Email != nil {
		e, err := normalizeEmail(*in.Email)
		if err != nil {
			return nil, err
		}
		if err := a.checkUnique(a.DB, userID, "", e); err != nil {
			return nil, err
		}
		updates["email"] = e
	}
	if in.Role != nil && *in.Role != u.Role {
		if actorID == userID {
			return nil, ErrCannotModifySelf
		}
		if *in.Role != models.RoleAdmin && *in.Role != models.RoleUser {
			return nil, BadRequest("invalid role")
		}
		updates["role"] = *in.Role
	}
	disable := false
	if in.Status != nil && *in.Status != u.Status {
		if actorID == userID {
			return nil, ErrCannotModifySelf
		}
		if *in.Status != models.StatusActive && *in.Status != models.StatusDisabled {
			return nil, BadRequest("invalid status")
		}
		updates["status"] = *in.Status
		disable = *in.Status == models.StatusDisabled
	}
	err = a.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
			return err
		}
		if disable {
			return tx.Where("user_id = ?", userID).Delete(&models.Session{}).Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return a.GetUser(userID)
}

func (a *App) AdminResetPassword(userID uint64, pw string) error {
	if _, err := a.GetUser(userID); err != nil {
		return err
	}
	if err := validatePassword(pw); err != nil {
		return err
	}
	hash, err := hashPassword(pw)
	if err != nil {
		return err
	}
	return a.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.User{}).Where("id = ?", userID).
			Updates(map[string]any{"password_hash": hash, "updated_at": now()}).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ?", userID).Delete(&models.Session{}).Error
	})
}

// AdminDeleteUser permanently removes a user and all of their data.
// It returns the avatar file name so the caller can delete it.
func (a *App) AdminDeleteUser(actorID, userID uint64) (string, error) {
	if actorID == userID {
		return "", ErrCannotModifySelf
	}
	u, err := a.GetUser(userID)
	if err != nil {
		return "", err
	}
	err = a.DB.Transaction(func(tx *gorm.DB) error {
		for _, m := range []any{&models.Item{}, &models.Card{}} {
			if err := tx.Where("owner_id = ?", userID).Delete(m).Error; err != nil {
				return err
			}
		}
		for _, m := range []any{&models.Session{}, &models.UserSetting{}, &models.MCPToken{}} {
			if err := tx.Where("user_id = ?", userID).Delete(m).Error; err != nil {
				return err
			}
		}
		return tx.Delete(&models.User{}, userID).Error
	})
	return u.Avatar, err
}

// ---------- sessions ----------

// SessionToken is returned to the client as an HttpOnly cookie. Only its
// SHA-256 hash is stored in the database.
func newToken() (token, id string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	token = hex.EncodeToString(b)
	return token, HashToken(token), nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (a *App) sessionLifetime(remember bool) time.Duration {
	if remember {
		return time.Duration(a.Cfg.Session.RememberDays) * 24 * time.Hour
	}
	return time.Duration(a.Cfg.Session.DefaultHours) * time.Hour
}

// CreateSession signs a user in and returns the raw cookie token.
func (a *App) CreateSession(u *models.User, remember bool, device, ip string) (string, *models.Session, error) {
	token, id, err := newToken()
	if err != nil {
		return "", nil, err
	}
	t := now()
	if len(device) > 255 {
		device = device[:255]
	}
	if len(ip) > 64 {
		ip = ip[:64]
	}
	s := &models.Session{ID: id, UserID: u.ID, Device: device, IP: ip, Remember: remember,
		ExpiresAt: t.Add(a.sessionLifetime(remember)), CreatedAt: t, LastActiveAt: t}
	err = a.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(s).Error; err != nil {
			return err
		}
		return tx.Model(&models.User{}).Where("id = ?", u.ID).Update("last_login_at", t).Error
	})
	if err != nil {
		return "", nil, err
	}
	return token, s, nil
}

// ValidateSession resolves a cookie token to an active session and user.
func (a *App) ValidateSession(token, ip string) (*models.Session, *models.User, error) {
	if token == "" {
		return nil, nil, ErrUnauthorized
	}
	var s models.Session
	if err := a.DB.Where("id = ?", HashToken(token)).First(&s).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrUnauthorized
		}
		return nil, nil, err
	}
	t := now()
	if t.After(s.ExpiresAt) {
		a.DB.Delete(&s)
		return nil, nil, ErrUnauthorized
	}
	u, err := a.GetUser(s.UserID)
	if err != nil {
		if err == ErrNotFound {
			return nil, nil, ErrUnauthorized
		}
		return nil, nil, err
	}
	if u.Status != models.StatusActive {
		return nil, nil, ErrUnauthorized
	}
	// Refresh activity at most once a minute; normal sessions slide forward.
	if t.Sub(s.LastActiveAt) > time.Minute {
		updates := map[string]any{"last_active_at": t}
		if ip != "" && len(ip) <= 64 {
			updates["ip"] = ip
		}
		if !s.Remember {
			s.ExpiresAt = t.Add(a.sessionLifetime(false))
			updates["expires_at"] = s.ExpiresAt
		}
		a.DB.Model(&models.Session{}).Where("id = ?", s.ID).Updates(updates)
	}
	return &s, u, nil
}

func (a *App) DeleteSessionByToken(token string) error {
	return a.DB.Where("id = ?", HashToken(token)).Delete(&models.Session{}).Error
}

type SessionInfo struct {
	ID           string    `json:"id"`
	Device       string    `json:"device"`
	IP           string    `json:"ip"`
	Current      bool      `json:"current"`
	CreatedAt    time.Time `json:"created_at"`
	LastActiveAt time.Time `json:"last_active_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func (a *App) ListSessions(userID uint64, currentID string) ([]SessionInfo, error) {
	var rows []models.Session
	if err := a.DB.Where("user_id = ? AND expires_at > ?", userID, now()).Order("last_active_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]SessionInfo, 0, len(rows))
	for _, s := range rows {
		out = append(out, SessionInfo{ID: s.ID[:16], Device: s.Device, IP: s.IP, Current: s.ID == currentID,
			CreatedAt: s.CreatedAt, LastActiveAt: s.LastActiveAt, ExpiresAt: s.ExpiresAt})
	}
	return out, nil
}

// RevokeSession removes one of the user's sessions identified by the 16 char
// prefix returned from ListSessions.
func (a *App) RevokeSession(userID uint64, prefix string) error {
	if len(prefix) != 16 {
		return ErrNotFound
	}
	if _, err := hex.DecodeString(prefix); err != nil {
		return ErrNotFound
	}
	res := a.DB.Where("user_id = ? AND id LIKE ?", userID, prefix+"%").Delete(&models.Session{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// PurgeExpiredSessions removes stale sessions.
func (a *App) PurgeExpiredSessions() error {
	return a.DB.Where("expires_at < ?", now()).Delete(&models.Session{}).Error
}
