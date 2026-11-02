package api

import (
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"anntodo/pkg/core"
	mw "anntodo/pkg/middleware"
)

// Server holds the HTTP layer dependencies.
type Server struct {
	App     *core.App
	Limiter *mw.LoginLimiter
	Web     fs.FS // built frontend (may be nil in tests)
}

// Router builds the Gin engine with every route.
func (s *Server) Router() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), mw.Logger(), mw.SecurityHeaders())
	_ = r.SetTrustedProxies(s.App.Cfg.Server.TrustedProxies)
	r.MaxMultipartMemory = 4 << 20

	api := r.Group("/api/v1", mw.CSRF())
	api.GET("/system/status", s.systemStatus)
	api.POST("/auth/setup", s.setup)
	api.POST("/auth/login", s.login)
	api.POST("/auth/register", s.register)
	api.POST("/auth/logout", s.logout)

	auth := api.Group("", mw.Auth(s.App))
	auth.GET("/auth/me", s.me)

	auth.GET("/cards", s.listCards)
	auth.POST("/cards", s.createCard)
	auth.PUT("/cards/order", s.reorderCards)
	auth.PUT("/cards/:id", s.updateCard)
	auth.DELETE("/cards/:id", s.deleteCard)

	auth.GET("/items", s.listItems)
	auth.POST("/items", s.createItem)
	auth.PUT("/items/order", s.reorderItems)
	auth.PUT("/items/quadrant-order", s.reorderQuadrants)
	auth.PUT("/items/:id", s.updateItem)
	auth.POST("/items/:id/complete", s.completeItem)
	auth.POST("/items/:id/move", s.moveItem)
	auth.DELETE("/items/:id", s.deleteItem)

	auth.GET("/settings", s.getSettings)
	auth.GET("/mcp/token", s.getMCPToken)
	auth.POST("/mcp/token", s.generateMCPToken)
	auth.DELETE("/mcp/token", s.revokeMCPToken)
	auth.PUT("/settings", s.patchSettings)

	auth.PUT("/account/profile", s.updateProfile)
	auth.PUT("/account/password", s.changePassword)
	auth.POST("/account/avatar", s.uploadAvatar)
	auth.DELETE("/account/avatar", s.deleteAvatar)
	auth.GET("/account/sessions", s.listSessions)
	auth.DELETE("/account/sessions/:id", s.revokeSession)
	auth.GET("/users/:id/avatar", s.getAvatar)

	auth.GET("/data/export", s.exportData)
	auth.POST("/data/import", s.importData)
	auth.GET("/trash", s.getTrash)
	auth.DELETE("/trash", s.emptyTrash)
	auth.POST("/trash/:type/:id/restore", s.restoreTrash)
	auth.DELETE("/trash/:type/:id", s.purgeTrash)

	admin := auth.Group("/admin", mw.Admin())
	admin.GET("/users", s.adminListUsers)
	admin.POST("/users", s.adminCreateUser)
	admin.PUT("/users/:id", s.adminUpdateUser)
	admin.POST("/users/:id/password", s.adminResetPassword)
	admin.DELETE("/users/:id", s.adminDeleteUser)
	admin.GET("/settings", s.adminGetSettings)
	admin.PUT("/settings", s.adminUpdateSettings)

	// MCP server for AI clients (Streamable HTTP, bearer token auth).
	mcp := r.Group("/mcp", mcpCORS())
	mcp.POST("", s.mcpPost)
	mcp.OPTIONS("", func(*gin.Context) {})
	mcp.GET("", s.mcpNotAllowed)
	mcp.DELETE("", s.mcpNotAllowed)

	r.NoRoute(s.static)
	return r
}

// ---------- helpers ----------

func fail(c *gin.Context, err error) { mw.AbortError(c, err) }

func bind(c *gin.Context, v any) bool {
	if err := c.ShouldBindJSON(v); err != nil {
		fail(c, core.BadRequest("invalid JSON body"))
		return false
	}
	return true
}

func paramID(c *gin.Context, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || id == 0 {
		fail(c, core.ErrNotFound)
		return 0, false
	}
	return id, true
}

func uid(c *gin.Context) uint64 { return mw.CurrentUser(c).ID }

func ok(c *gin.Context, v any) { c.JSON(http.StatusOK, v) }

func noContent(c *gin.Context) { c.Status(http.StatusNoContent) }

// ---------- static frontend ----------

func (s *Server) static(c *gin.Context) {
	p := c.Request.URL.Path
	if strings.HasPrefix(p, "/api/") {
		fail(c, core.ErrNotFound)
		return
	}
	if s.Web == nil || (c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead) {
		c.Status(http.StatusNotFound)
		return
	}
	name := strings.TrimPrefix(p, "/")
	if name != "" && !strings.Contains(name, "..") {
		if st, err := fs.Stat(s.Web, name); err == nil && !st.IsDir() {
			switch {
			case strings.HasPrefix(name, "assets/"):
				c.Header("Cache-Control", "public, max-age=31536000, immutable")
			case name == "sw.js" || name == "manifest.webmanifest":
				c.Header("Cache-Control", "no-cache")
				if strings.HasSuffix(name, ".webmanifest") {
					// Not in Go's built-in MIME table on every platform.
					c.Header("Content-Type", "application/manifest+json")
				}
			default:
				c.Header("Cache-Control", "public, max-age=3600")
			}
			c.FileFromFS(name, http.FS(s.Web))
			return
		}
	}
	// SPA fallback: unknown paths render index.html.
	data, err := fs.ReadFile(s.Web, "index.html")
	if err != nil {
		c.String(http.StatusNotFound, "frontend not built")
		return
	}
	c.Header("Cache-Control", "no-cache")
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}
