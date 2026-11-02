package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"anntodo/pkg/core"
	mw "anntodo/pkg/middleware"
	"anntodo/pkg/models"
	"anntodo/pkg/version"
)

type userView struct {
	ID          uint64  `json:"id"`
	Username    string  `json:"username"`
	Email       *string `json:"email"`
	Nickname    string  `json:"nickname"`
	Role        string  `json:"role"`
	Status      string  `json:"status"`
	HasAvatar   bool    `json:"has_avatar"`
	AvatarURL   string  `json:"avatar_url"`
	LastLoginAt any     `json:"last_login_at"`
	CreatedAt   any     `json:"created_at"`
}

func viewUser(u *models.User) userView {
	v := userView{ID: u.ID, Username: u.Username, Email: u.Email, Nickname: u.Nickname, Role: u.Role, Status: u.Status,
		HasAvatar: u.Avatar != "", LastLoginAt: u.LastLoginAt, CreatedAt: u.CreatedAt}
	if u.Avatar != "" {
		v.AvatarURL = "/api/v1/users/" + strconv.FormatUint(u.ID, 10) + "/avatar?v=" + strconv.FormatInt(u.UpdatedAt.Unix(), 10)
	}
	return v
}

func (s *Server) systemStatus(c *gin.Context) {
	initialized, err := s.App.Initialized()
	if err != nil {
		fail(c, err)
		return
	}
	sys, err := s.App.GetSystemSettings()
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"initialized": initialized, "allow_register": sys.AllowRegister, "site_name": sys.SiteName, "version": version.Version})
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Remember bool   `json:"remember"`
}

func (s *Server) startSession(c *gin.Context, u *models.User, remember bool) {
	token, sess, err := s.App.CreateSession(u, remember, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		fail(c, err)
		return
	}
	cookie := &http.Cookie{Name: mw.SessionCookie, Value: token, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: mw.SecureRequest(c)}
	if remember {
		cookie.Expires = sess.ExpiresAt
		cookie.MaxAge = int(sess.ExpiresAt.Sub(sess.CreatedAt).Seconds())
	}
	http.SetCookie(c.Writer, cookie)
	fresh, err := s.App.GetUser(u.ID)
	if err != nil {
		fail(c, err)
		return
	}
	settings, err := s.App.GetUserSettings(u.ID)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"user": viewUser(fresh), "settings": settings})
}

func (s *Server) setup(c *gin.Context) {
	var in core.UserInput
	if !bind(c, &in) {
		return
	}
	u, err := s.App.Setup(in)
	if err != nil {
		fail(c, err)
		return
	}
	s.startSession(c, u, false)
}

func (s *Server) register(c *gin.Context) {
	var in core.UserInput
	if !bind(c, &in) {
		return
	}
	u, err := s.App.Register(in)
	if err != nil {
		fail(c, err)
		return
	}
	s.startSession(c, u, false)
}

func (s *Server) login(c *gin.Context) {
	var in loginRequest
	if !bind(c, &in) {
		return
	}
	ip := c.ClientIP()
	if allowed, wait := s.Limiter.Allowed(ip, in.Username); !allowed {
		c.Header("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": gin.H{"code": "too_many_attempts",
			"message": "too many failed attempts, try again later", "retry_after": int(wait.Seconds()) + 1}})
		return
	}
	u, err := s.App.Authenticate(in.Username, in.Password)
	if err != nil {
		if err == core.ErrInvalidCredentials {
			s.Limiter.Fail(ip, in.Username)
		}
		fail(c, err)
		return
	}
	s.Limiter.Success(in.Username)
	s.startSession(c, u, in.Remember)
}

func (s *Server) logout(c *gin.Context) {
	if token, err := c.Cookie(mw.SessionCookie); err == nil && token != "" {
		_ = s.App.DeleteSessionByToken(token)
	}
	http.SetCookie(c.Writer, &http.Cookie{Name: mw.SessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: mw.SecureRequest(c)})
	noContent(c)
}

func (s *Server) me(c *gin.Context) {
	settings, err := s.App.GetUserSettings(uid(c))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"user": viewUser(mw.CurrentUser(c)), "settings": settings})
}
