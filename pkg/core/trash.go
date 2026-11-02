package core

import (
	"time"

	"gorm.io/gorm"

	"anntodo/pkg/models"
)

const TrashRetention = 30 * 24 * time.Hour

type TrashCard struct {
	ID        uint64    `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	ItemCount int64     `json:"item_count"`
	DeletedAt time.Time `json:"deleted_at"`
}

type TrashItem struct {
	ID        uint64    `json:"id"`
	Title     string    `json:"title"`
	CardID    uint64    `json:"card_id"`
	CardName  string    `json:"card_name"`
	DeletedAt time.Time `json:"deleted_at"`
}

type Trash struct {
	Cards []TrashCard `json:"cards"`
	Items []TrashItem `json:"items"`
}

// GetTrash lists deleted cards and individually deleted items whose card still exists.
func (a *App) GetTrash(userID uint64) (*Trash, error) {
	t := &Trash{Cards: []TrashCard{}, Items: []TrashItem{}}
	var cards []models.Card
	if err := a.DB.Where("owner_id = ? AND deleted_at IS NOT NULL", userID).Order("deleted_at DESC").Find(&cards).Error; err != nil {
		return nil, err
	}
	for _, c := range cards {
		var n int64
		if err := a.DB.Model(&models.Item{}).Where("card_id = ? AND deleted_with_card = ?", c.ID, true).Count(&n).Error; err != nil {
			return nil, err
		}
		t.Cards = append(t.Cards, TrashCard{ID: c.ID, Name: c.Name, Color: c.Color, ItemCount: n, DeletedAt: *c.DeletedAt})
	}
	type row struct {
		ID        uint64
		Title     string
		CardID    uint64
		CardName  string
		DeletedAt time.Time
	}
	var rows []row
	err := a.DB.Table("items").
		Select("items.id, items.title, items.card_id, cards.name AS card_name, items.deleted_at").
		Joins("JOIN cards ON cards.id = items.card_id").
		Where("items.owner_id = ? AND items.deleted_at IS NOT NULL AND items.deleted_with_card = ? AND cards.deleted_at IS NULL", userID, false).
		Order("items.deleted_at DESC").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		t.Items = append(t.Items, TrashItem(r))
	}
	return t, nil
}

func (a *App) RestoreCard(userID, cardID uint64) error {
	return a.DB.Transaction(func(tx *gorm.DB) error {
		var c models.Card
		if err := tx.Where("id = ? AND owner_id = ? AND deleted_at IS NOT NULL", cardID, userID).First(&c).Error; err != nil {
			return ErrNotFound
		}
		var max *int64
		if err := tx.Model(&models.Card{}).Where("owner_id = ? AND deleted_at IS NULL", userID).
			Select("MAX(sort_order)").Scan(&max).Error; err != nil {
			return err
		}
		order := int64(0)
		if max != nil {
			order = *max + 1
		}
		if err := tx.Model(&c).Updates(map[string]any{"deleted_at": nil, "sort_order": order, "updated_at": now()}).Error; err != nil {
			return err
		}
		return tx.Model(&models.Item{}).Where("card_id = ? AND deleted_with_card = ?", c.ID, true).
			Updates(map[string]any{"deleted_at": nil, "deleted_with_card": false}).Error
	})
}

func (a *App) RestoreItem(userID, itemID uint64) error {
	return a.DB.Transaction(func(tx *gorm.DB) error {
		var it models.Item
		if err := tx.Where("id = ? AND owner_id = ? AND deleted_at IS NOT NULL AND deleted_with_card = ?", itemID, userID, false).First(&it).Error; err != nil {
			return ErrNotFound
		}
		if _, err := a.getCard(tx, userID, it.CardID); err != nil {
			return err
		}
		order, err := nextOrder(tx, "sort_order", it.CardID)
		if err != nil {
			return err
		}
		return tx.Model(&it).Updates(map[string]any{"deleted_at": nil, "sort_order": order, "updated_at": now()}).Error
	})
}

func (a *App) PurgeCard(userID, cardID uint64) error {
	return a.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Where("id = ? AND owner_id = ? AND deleted_at IS NOT NULL", cardID, userID).Delete(&models.Card{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		return tx.Where("card_id = ?", cardID).Delete(&models.Item{}).Error
	})
}

func (a *App) PurgeItem(userID, itemID uint64) error {
	res := a.DB.Where("id = ? AND owner_id = ? AND deleted_at IS NOT NULL", itemID, userID).Delete(&models.Item{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// EmptyTrash permanently removes everything in the user's trash.
func (a *App) EmptyTrash(userID uint64) error {
	return a.DB.Transaction(func(tx *gorm.DB) error {
		sub := tx.Model(&models.Card{}).Select("id").Where("owner_id = ? AND deleted_at IS NOT NULL", userID)
		if err := tx.Where("card_id IN (?)", sub).Delete(&models.Item{}).Error; err != nil {
			return err
		}
		if err := tx.Where("owner_id = ? AND deleted_at IS NOT NULL", userID).Delete(&models.Item{}).Error; err != nil {
			return err
		}
		return tx.Where("owner_id = ? AND deleted_at IS NOT NULL", userID).Delete(&models.Card{}).Error
	})
}

// PurgeOldTrash removes trash older than the retention period for all users.
func (a *App) PurgeOldTrash() error {
	cutoff := now().Add(-TrashRetention)
	return a.DB.Transaction(func(tx *gorm.DB) error {
		sub := tx.Model(&models.Card{}).Select("id").Where("deleted_at IS NOT NULL AND deleted_at < ?", cutoff)
		if err := tx.Where("card_id IN (?)", sub).Delete(&models.Item{}).Error; err != nil {
			return err
		}
		if err := tx.Where("deleted_at IS NOT NULL AND deleted_at < ?", cutoff).Delete(&models.Item{}).Error; err != nil {
			return err
		}
		return tx.Where("deleted_at IS NOT NULL AND deleted_at < ?", cutoff).Delete(&models.Card{}).Error
	})
}
