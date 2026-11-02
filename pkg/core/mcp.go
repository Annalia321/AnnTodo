package core

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"anntodo/pkg/models"
)

// MCPTokenPrefix marks AnnTodo MCP access tokens.
const MCPTokenPrefix = "sk-"

// MCPTokenInfo describes the user's MCP token without revealing it.
type MCPTokenInfo struct {
	Exists     bool       `json:"exists"`
	Hint       string     `json:"hint"`
	CreatedAt  *time.Time `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
}

func tokenInfo(t *models.MCPToken) *MCPTokenInfo {
	if t == nil {
		return &MCPTokenInfo{}
	}
	created := t.CreatedAt
	return &MCPTokenInfo{Exists: true, Hint: t.Hint, CreatedAt: &created, LastUsedAt: t.LastUsedAt}
}

func validTimezone(tz string) bool {
	if tz == "" || len(tz) > 64 {
		return false
	}
	_, err := time.LoadLocation(tz)
	return err == nil
}

// GetMCPToken reports whether the user has an MCP token.
func (a *App) GetMCPToken(userID uint64) (*MCPTokenInfo, error) {
	var t models.MCPToken
	err := a.DB.Where("user_id = ?", userID).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return tokenInfo(nil), nil
	}
	if err != nil {
		return nil, err
	}
	return tokenInfo(&t), nil
}

// GenerateMCPToken creates a new token and replaces the previous one. The
// plaintext token is returned only here; the database keeps its hash.
// tz is the browser timezone, used when the user follows the browser timezone.
func (a *App) GenerateMCPToken(userID uint64, tz string) (string, *MCPTokenInfo, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, err
	}
	token := MCPTokenPrefix + hex.EncodeToString(b)
	if !validTimezone(tz) {
		tz = ""
	}
	row := &models.MCPToken{ID: HashToken(token), UserID: userID, Hint: token[:10], Timezone: tz, CreatedAt: now()}
	err := a.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&models.MCPToken{}).Error; err != nil {
			return err
		}
		return tx.Create(row).Error
	})
	if err != nil {
		return "", nil, err
	}
	// Older builds stored the token in plain text in the user settings.
	_, _ = a.PatchUserSettings(userID, map[string]any{"mcp_token": nil})
	return token, tokenInfo(row), nil
}

// RevokeMCPToken deletes the user's token; AI clients using it lose access.
func (a *App) RevokeMCPToken(userID uint64) error {
	return a.DB.Where("user_id = ?", userID).Delete(&models.MCPToken{}).Error
}

// ValidateMCPToken resolves a bearer token to an active user.
func (a *App) ValidateMCPToken(token string) (*models.User, *models.MCPToken, error) {
	if !strings.HasPrefix(token, MCPTokenPrefix) || len(token) != len(MCPTokenPrefix)+64 {
		return nil, nil, ErrUnauthorized
	}
	var t models.MCPToken
	if err := a.DB.Where("id = ?", HashToken(token)).First(&t).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrUnauthorized
		}
		return nil, nil, err
	}
	u, err := a.GetUser(t.UserID)
	if err != nil {
		if err == ErrNotFound {
			return nil, nil, ErrUnauthorized
		}
		return nil, nil, err
	}
	if u.Status != models.StatusActive {
		return nil, nil, ErrUnauthorized
	}
	ts := now()
	if t.LastUsedAt == nil || ts.Sub(*t.LastUsedAt) > time.Minute {
		a.DB.Model(&models.MCPToken{}).Where("id = ?", t.ID).Update("last_used_at", ts)
		t.LastUsedAt = &ts
	}
	return u, &t, nil
}

// UserLocation returns the timezone used to read and write due times for an
// MCP client: the user's timezone setting, else the browser timezone recorded
// with the token, else the server's local timezone.
func (a *App) UserLocation(userID uint64, tokenTZ string) *time.Location {
	if s, err := a.GetUserSettings(userID); err == nil {
		if tz, _ := s["timezone"].(string); validTimezone(tz) {
			loc, _ := time.LoadLocation(tz)
			return loc
		}
	}
	if validTimezone(tokenTZ) {
		loc, _ := time.LoadLocation(tokenTZ)
		return loc
	}
	return time.Local
}

// SearchItems finds the user's live items whose title or note contains query.
func (a *App) SearchItems(userID uint64, query string, cardID uint64, limit int) ([]models.Item, error) {
	query = strings.TrimSpace(query)
	items := []models.Item{}
	if query == "" {
		return items, nil
	}
	q := a.DB.Where("owner_id = ? AND deleted_at IS NULL", userID)
	if cardID > 0 {
		q = q.Where("card_id = ?", cardID)
	}
	esc := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(strings.ToLower(query))
	pattern := "%" + esc + "%"
	q = q.Where(`(LOWER(title) LIKE ? ESCAPE '\' OR LOWER(note) LIKE ? ESCAPE '\')`, pattern, pattern)
	err := q.Order("completed, sort_order, id").Limit(limit).Find(&items).Error
	return items, err
}

// GetItem returns one live item of the user.
func (a *App) GetItem(userID, itemID uint64) (*models.Item, error) {
	return a.getItem(a.DB, userID, itemID)
}

// CardItemCounts returns [open, completed] live item counts per card.
func (a *App) CardItemCounts(userID uint64) (map[uint64][2]int64, error) {
	var rows []struct {
		CardID    uint64
		Completed bool
		N         int64
	}
	err := a.DB.Model(&models.Item{}).Select("card_id, completed, COUNT(*) AS n").
		Where("owner_id = ? AND deleted_at IS NULL", userID).Group("card_id, completed").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := map[uint64][2]int64{}
	for _, r := range rows {
		v := out[r.CardID]
		if r.Completed {
			v[1] = r.N
		} else {
			v[0] = r.N
		}
		out[r.CardID] = v
	}
	return out, nil
}
