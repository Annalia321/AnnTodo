package core

import (
	"errors"

	"gorm.io/gorm"

	"anntodo/pkg/models"
)

type CardInput struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type CardPatch struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
}

func (a *App) ListCards(userID uint64) ([]models.Card, error) {
	cards := []models.Card{}
	err := a.DB.Where("owner_id = ? AND deleted_at IS NULL", userID).Order("sort_order, id").Find(&cards).Error
	return cards, err
}

func (a *App) getCard(tx *gorm.DB, userID, cardID uint64) (*models.Card, error) {
	var c models.Card
	err := tx.Where("id = ? AND owner_id = ? AND deleted_at IS NULL", cardID, userID).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &c, err
}

func (a *App) CreateCard(userID uint64, in CardInput) (*models.Card, error) {
	name, err := cleanText(in.Name, 100, "name", true)
	if err != nil {
		return nil, err
	}
	if !validColor(in.Color) {
		return nil, BadRequest("invalid color")
	}
	var c *models.Card
	err = a.DB.Transaction(func(tx *gorm.DB) error {
		var max *int64
		if err := tx.Model(&models.Card{}).Where("owner_id = ? AND deleted_at IS NULL", userID).
			Select("MAX(sort_order)").Scan(&max).Error; err != nil {
			return err
		}
		order := int64(0)
		if max != nil {
			order = *max + 1
		}
		c = &models.Card{OwnerID: userID, Name: name, Color: in.Color, SortOrder: order}
		return tx.Create(c).Error
	})
	return c, err
}

func (a *App) UpdateCard(userID, cardID uint64, in CardPatch) (*models.Card, error) {
	c, err := a.getCard(a.DB, userID, cardID)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{"updated_at": now()}
	if in.Name != nil {
		name, err := cleanText(*in.Name, 100, "name", true)
		if err != nil {
			return nil, err
		}
		updates["name"] = name
	}
	if in.Color != nil {
		if !validColor(*in.Color) {
			return nil, BadRequest("invalid color")
		}
		updates["color"] = *in.Color
	}
	if err := a.DB.Model(c).Updates(updates).Error; err != nil {
		return nil, err
	}
	return a.getCard(a.DB, userID, cardID)
}

// DeleteCard moves a card and all of its items to the trash.
func (a *App) DeleteCard(userID, cardID uint64) error {
	return a.DB.Transaction(func(tx *gorm.DB) error {
		c, err := a.getCard(tx, userID, cardID)
		if err != nil {
			return err
		}
		t := now()
		if err := tx.Model(&models.Item{}).Where("card_id = ? AND deleted_at IS NULL", c.ID).
			Updates(map[string]any{"deleted_at": t, "deleted_with_card": true}).Error; err != nil {
			return err
		}
		return tx.Model(c).Update("deleted_at", t).Error
	})
}

// ReorderCards stores a new card order. ids must contain the user's cards.
func (a *App) ReorderCards(userID uint64, ids []uint64) error {
	return a.DB.Transaction(func(tx *gorm.DB) error {
		for i, id := range ids {
			res := tx.Model(&models.Card{}).Where("id = ? AND owner_id = ? AND deleted_at IS NULL", id, userID).
				Update("sort_order", i)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return ErrNotFound
			}
		}
		return nil
	})
}
