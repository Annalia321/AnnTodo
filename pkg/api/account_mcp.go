package api

import (
	"github.com/gin-gonic/gin"
)

// getMCPToken reports whether the user has an MCP token (never the token itself).
func (s *Server) getMCPToken(c *gin.Context) {
	info, err := s.App.GetMCPToken(uid(c))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, info)
}

// generateMCPToken creates a new token, replacing the previous one. The
// plaintext token is only returned in this response.
func (s *Server) generateMCPToken(c *gin.Context) {
	var in struct {
		Timezone string `json:"timezone"`
	}
	if c.Request.ContentLength != 0 && !bind(c, &in) {
		return
	}
	token, info, err := s.App.GenerateMCPToken(uid(c), in.Timezone)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"token": token, "info": info})
}

// revokeMCPToken deletes the token; AI clients using it lose access at once.
func (s *Server) revokeMCPToken(c *gin.Context) {
	if err := s.App.RevokeMCPToken(uid(c)); err != nil {
		fail(c, err)
		return
	}
	noContent(c)
}
