package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"

	"anntodo/pkg/core"
	mw "anntodo/pkg/middleware"
)

const maxAvatarSize = 2 << 20

func (s *Server) avatarDir() string { return filepath.Join(s.App.Cfg.Storage.DataDir, "avatars") }

func (s *Server) removeAvatarFile(name string) {
	if name == "" || filepath.Base(name) != name {
		return
	}
	_ = os.Remove(filepath.Join(s.avatarDir(), name))
}

func (s *Server) updateProfile(c *gin.Context) {
	var in core.ProfileInput
	if !bind(c, &in) {
		return
	}
	u, err := s.App.UpdateProfile(uid(c), in)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, viewUser(u))
}

func (s *Server) changePassword(c *gin.Context) {
	var in struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if !bind(c, &in) {
		return
	}
	if err := s.App.ChangePassword(uid(c), mw.CurrentSession(c).ID, in.OldPassword, in.NewPassword); err != nil {
		fail(c, err)
		return
	}
	noContent(c)
}

func (s *Server) uploadAvatar(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAvatarSize+64*1024)
	fh, err := c.FormFile("avatar")
	if err != nil || fh.Size > maxAvatarSize {
		fail(c, core.ErrInvalidAvatar)
		return
	}
	f, err := fh.Open()
	if err != nil {
		fail(c, core.ErrInvalidAvatar)
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxAvatarSize+1))
	if err != nil || len(data) > maxAvatarSize {
		fail(c, core.ErrInvalidAvatar)
		return
	}
	ext := map[string]string{"image/png": ".png", "image/jpeg": ".jpg", "image/gif": ".gif", "image/webp": ".webp"}[http.DetectContentType(data)]
	if ext == "" {
		fail(c, core.ErrInvalidAvatar)
		return
	}
	if err := os.MkdirAll(s.avatarDir(), 0o750); err != nil {
		fail(c, err)
		return
	}
	rnd := make([]byte, 8)
	_, _ = rand.Read(rnd)
	name := fmt.Sprintf("%d-%s%s", uid(c), hex.EncodeToString(rnd), ext)
	if err := os.WriteFile(filepath.Join(s.avatarDir(), name), data, 0o640); err != nil {
		fail(c, err)
		return
	}
	old := mw.CurrentUser(c).Avatar
	if err := s.App.SetAvatar(uid(c), name); err != nil {
		fail(c, err)
		return
	}
	s.removeAvatarFile(old)
	u, err := s.App.GetUser(uid(c))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, viewUser(u))
}

func (s *Server) deleteAvatar(c *gin.Context) {
	old := mw.CurrentUser(c).Avatar
	if err := s.App.SetAvatar(uid(c), ""); err != nil {
		fail(c, err)
		return
	}
	s.removeAvatarFile(old)
	u, err := s.App.GetUser(uid(c))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, viewUser(u))
}

func (s *Server) getAvatar(c *gin.Context) {
	id, good := paramID(c, "id")
	if !good {
		return
	}
	u, err := s.App.GetUser(id)
	if err != nil || u.Avatar == "" || filepath.Base(u.Avatar) != u.Avatar {
		fail(c, core.ErrNotFound)
		return
	}
	c.Header("Cache-Control", "private, max-age=86400")
	c.File(filepath.Join(s.avatarDir(), u.Avatar))
}

func (s *Server) listSessions(c *gin.Context) {
	list, err := s.App.ListSessions(uid(c), mw.CurrentSession(c).ID)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, list)
}

func (s *Server) revokeSession(c *gin.Context) {
	if err := s.App.RevokeSession(uid(c), c.Param("id")); err != nil {
		fail(c, err)
		return
	}
	noContent(c)
}

func (s *Server) exportData(c *gin.Context) {
	data, err := s.App.Export(uid(c))
	if err != nil {
		fail(c, err)
		return
	}
	name := "anntodo-export-" + time.Now().Format("20060102-150405") + ".json"
	c.Header("Content-Disposition", `attachment; filename="`+name+`"`)
	c.JSON(http.StatusOK, data)
}

func (s *Server) importData(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 20<<20)
	var data core.ExportData
	if err := json.NewDecoder(c.Request.Body).Decode(&data); err != nil {
		fail(c, core.ErrInvalidImport)
		return
	}
	res, err := s.App.Import(uid(c), &data)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, res)
}

func (s *Server) getTrash(c *gin.Context) {
	t, err := s.App.GetTrash(uid(c))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, t)
}

func (s *Server) emptyTrash(c *gin.Context) {
	if err := s.App.EmptyTrash(uid(c)); err != nil {
		fail(c, err)
		return
	}
	noContent(c)
}

func trashTarget(c *gin.Context) (string, uint64, bool) {
	typ := c.Param("type")
	if typ != "card" && typ != "item" {
		fail(c, core.ErrNotFound)
		return "", 0, false
	}
	id, good := paramID(c, "id")
	return typ, id, good
}

func (s *Server) restoreTrash(c *gin.Context) {
	typ, id, good := trashTarget(c)
	if !good {
		return
	}
	var err error
	if typ == "card" {
		err = s.App.RestoreCard(uid(c), id)
	} else {
		err = s.App.RestoreItem(uid(c), id)
	}
	if err != nil {
		fail(c, err)
		return
	}
	noContent(c)
}

func (s *Server) purgeTrash(c *gin.Context) {
	typ, id, good := trashTarget(c)
	if !good {
		return
	}
	var err error
	if typ == "card" {
		err = s.App.PurgeCard(uid(c), id)
	} else {
		err = s.App.PurgeItem(uid(c), id)
	}
	if err != nil {
		fail(c, err)
		return
	}
	noContent(c)
}

// ---------- admin ----------

func (s *Server) adminListUsers(c *gin.Context) {
	users, err := s.App.ListUsers()
	if err != nil {
		fail(c, err)
		return
	}
	out := make([]userView, 0, len(users))
	for i := range users {
		out = append(out, viewUser(&users[i]))
	}
	ok(c, out)
}

func (s *Server) adminCreateUser(c *gin.Context) {
	var in core.UserInput
	if !bind(c, &in) {
		return
	}
	u, err := s.App.AdminCreateUser(in)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, viewUser(u))
}

func (s *Server) adminUpdateUser(c *gin.Context) {
	id, good := paramID(c, "id")
	if !good {
		return
	}
	var in core.AdminUserUpdate
	if !bind(c, &in) {
		return
	}
	u, err := s.App.AdminUpdateUser(uid(c), id, in)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, viewUser(u))
}

func (s *Server) adminResetPassword(c *gin.Context) {
	id, good := paramID(c, "id")
	if !good {
		return
	}
	var in struct {
		Password string `json:"password"`
	}
	if !bind(c, &in) {
		return
	}
	if err := s.App.AdminResetPassword(id, in.Password); err != nil {
		fail(c, err)
		return
	}
	noContent(c)
}

func (s *Server) adminDeleteUser(c *gin.Context) {
	id, good := paramID(c, "id")
	if !good {
		return
	}
	avatar, err := s.App.AdminDeleteUser(uid(c), id)
	if err != nil {
		fail(c, err)
		return
	}
	s.removeAvatarFile(avatar)
	noContent(c)
}

func (s *Server) adminGetSettings(c *gin.Context) {
	st, err := s.App.GetSystemSettings()
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, st)
}

func (s *Server) adminUpdateSettings(c *gin.Context) {
	var in core.SystemSettings
	if !bind(c, &in) {
		return
	}
	st, err := s.App.UpdateSystemSettings(in)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, st)
}
