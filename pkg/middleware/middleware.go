package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"anntodo/pkg/core"
	"anntodo/pkg/models"
)

const (
	SessionCookie = "anntodo_session"
	CSRFCookie    = "anntodo_csrf"
	CSRFHeader    = "X-CSRF-Token"

	ctxUser    = "anntodo.user"
	ctxSession = "anntodo.session"
)

// SecureRequest reports whether the client reached us over HTTPS.
func SecureRequest(c *gin.Context) bool {
	return c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
}

// AbortError writes a JSON error response.
func AbortError(c *gin.Context, err error) {
	if e, ok := core.AsError(err); ok {
		c.AbortWithStatusJSON(e.Status, gin.H{"error": gin.H{"code": e.Code, "message": e.Message}})
		return
	}
	slog.Error("request failed", "path", c.Request.URL.Path, "err", err)
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "internal", "message": "internal server error"}})
}

// CSRF implements the double submit cookie pattern: a random token is set in
// a readable cookie and every state changing request must echo it in a header.
func CSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(CSRFCookie)
		if err != nil || len(token) != 64 {
			b := make([]byte, 32)
			_, _ = rand.Read(b)
			token = hex.EncodeToString(b)
			http.SetCookie(c.Writer, &http.Cookie{Name: CSRFCookie, Value: token, Path: "/", MaxAge: 365 * 24 * 3600,
				SameSite: http.SameSiteLaxMode, Secure: SecureRequest(c)})
			c.Request.AddCookie(&http.Cookie{Name: CSRFCookie, Value: token})
		}
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			h := c.GetHeader(CSRFHeader)
			if h == "" || subtle.ConstantTimeCompare([]byte(h), []byte(token)) != 1 {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "csrf_failed", "message": "invalid CSRF token, reload the page"}})
				return
			}
		}
		c.Next()
	}
}

// Auth requires a valid session cookie.
func Auth(app *core.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, _ := c.Cookie(SessionCookie)
		s, u, err := app.ValidateSession(token, c.ClientIP())
		if err != nil {
			AbortError(c, err)
			return
		}
		c.Set(ctxUser, u)
		c.Set(ctxSession, s)
		c.Next()
	}
}

// Admin requires the signed in user to be an administrator. Use after Auth.
func Admin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if CurrentUser(c).Role != models.RoleAdmin {
			AbortError(c, core.ErrForbidden)
			return
		}
		c.Next()
	}
}

func CurrentUser(c *gin.Context) *models.User {
	v, _ := c.Get(ctxUser)
	u, _ := v.(*models.User)
	return u
}

func CurrentSession(c *gin.Context) *models.Session {
	v, _ := c.Get(ctxSession)
	s, _ := v.(*models.Session)
	return s
}

// Logger logs every request with log/slog.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		status := c.Writer.Status()
		lvl := slog.LevelDebug
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			lvl = slog.LevelInfo
		}
		if status >= 500 {
			lvl = slog.LevelError
		}
		slog.Log(c.Request.Context(), lvl, "http", "method", c.Request.Method, "path", c.Request.URL.Path,
			"status", status, "duration", time.Since(start).String(), "ip", c.ClientIP())
	}
}

// SecurityHeaders adds conservative browser security headers.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		c.Next()
	}
}

// LoginLimiter throttles failed login attempts per IP and per account.
type LoginLimiter struct {
	mu       sync.Mutex
	failures map[string][]time.Time
	window   time.Duration
	perIP    int
	perUser  int
}

func NewLoginLimiter() *LoginLimiter {
	l := &LoginLimiter{failures: map[string][]time.Time{}, window: 15 * time.Minute, perIP: 20, perUser: 5}
	go func() {
		for range time.Tick(5 * time.Minute) {
			l.cleanup()
		}
	}()
	return l
}

func (l *LoginLimiter) recent(key string, t time.Time) []time.Time {
	list := l.failures[key]
	i := 0
	for i < len(list) && t.Sub(list[i]) > l.window {
		i++
	}
	list = list[i:]
	if len(list) == 0 {
		delete(l.failures, key)
	} else {
		l.failures[key] = list
	}
	return list
}

// Allowed returns false and the wait time when the caller must back off.
func (l *LoginLimiter) Allowed(ip, user string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	t := time.Now()
	check := func(key string, limit int) (bool, time.Duration) {
		list := l.recent(key, t)
		if len(list) >= limit {
			return false, l.window - t.Sub(list[0])
		}
		return true, 0
	}
	if ok, d := check("ip:"+ip, l.perIP); !ok {
		return false, d
	}
	return check("user:"+strings.ToLower(user), l.perUser)
}

func (l *LoginLimiter) Fail(ip, user string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	t := time.Now()
	l.failures["ip:"+ip] = append(l.failures["ip:"+ip], t)
	k := "user:" + strings.ToLower(user)
	l.failures[k] = append(l.failures[k], t)
}

func (l *LoginLimiter) Success(user string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.failures, "user:"+strings.ToLower(user))
}

func (l *LoginLimiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()
	t := time.Now()
	for k := range l.failures {
		l.recent(k, t)
	}
}
