package core

import (
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"anntodo/pkg/models"
)

const maxSettingsSize = 64 * 1024

func parseDate(s string) (time.Time, error) { return time.Parse("2006-01-02", s) }

// GetUserSettings returns the stored preference object (possibly empty).
// Defaults live in the frontend so that new options need no migration.
func (a *App) GetUserSettings(userID uint64) (map[string]any, error) {
	var row models.UserSetting
	err := a.DB.Where("user_id = ?", userID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if err := json.Unmarshal([]byte(row.Settings), &out); err != nil {
		return map[string]any{}, nil
	}
	return out, nil
}

// PatchUserSettings merges patch into the stored settings. A null value
// removes the key.
func (a *App) PatchUserSettings(userID uint64, patch map[string]any) (map[string]any, error) {
	var result map[string]any
	err := a.DB.Transaction(func(tx *gorm.DB) error {
		cur := map[string]any{}
		var row models.UserSetting
		err := tx.Where("user_id = ?", userID).First(&row).Error
		if err == nil {
			_ = json.Unmarshal([]byte(row.Settings), &cur)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		for k, v := range patch {
			if len(k) > 64 {
				return BadRequest("setting key too long")
			}
			if v == nil {
				delete(cur, k)
			} else {
				cur[k] = v
			}
		}
		b, err := json.Marshal(cur)
		if err != nil {
			return err
		}
		if len(b) > maxSettingsSize {
			return BadRequest("settings too large")
		}
		result = cur
		return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}}, DoUpdates: clause.AssignmentColumns([]string{"settings", "updated_at"})}).
			Create(&models.UserSetting{UserID: userID, Settings: string(b), UpdatedAt: now()}).Error
	})
	return result, err
}
