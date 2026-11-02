package core

import (
	"encoding/json"
	"errors"
	"strings"

	"gorm.io/gorm"

	"anntodo/pkg/models"
)

// OptString distinguishes an absent JSON field from an explicit null.
type OptString struct {
	Set   bool
	Value *string
}

func (o *OptString) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		o.Value = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	s = strings.TrimSpace(s)
	if s == "" {
		o.Value = nil
	} else {
		o.Value = &s
	}
	return nil
}

type ItemInput struct {
	CardID    uint64  `json:"card_id"`
	Title     string  `json:"title"`
	Note      string  `json:"note"`
	DueDate   *string `json:"due_date"`
	DueTime   *string `json:"due_time"`
	Important bool    `json:"important"`
	Urgent    bool    `json:"urgent"`
	Position  string  `json:"position"` // "top" or "bottom" (default)
}

type ItemPatch struct {
	CardID    *uint64   `json:"card_id"`
	Title     *string   `json:"title"`
	Note      *string   `json:"note"`
	DueDate   OptString `json:"due_date"`
	DueTime   OptString `json:"due_time"`
	Important *bool     `json:"important"`
	Urgent    *bool     `json:"urgent"`
}

type ItemFilter struct {
	CardID uint64
	From   string // YYYY-MM-DD, inclusive
	To     string // YYYY-MM-DD, inclusive
}

func emptyToNil(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}

func validateDue(date, tm *string) error {
	if date != nil && !validDate(*date) {
		return BadRequest("invalid due_date")
	}
	if tm != nil {
		if !validTime(*tm) {
			return BadRequest("invalid due_time")
		}
		if date == nil {
			return BadRequest("due_time requires due_date")
		}
	}
	return nil
}

func (a *App) ListItems(userID uint64, f ItemFilter) ([]models.Item, error) {
	items := []models.Item{}
	q := a.DB.Where("owner_id = ? AND deleted_at IS NULL", userID)
	if f.CardID != 0 {
		q = q.Where("card_id = ?", f.CardID)
	}
	if f.From != "" || f.To != "" {
		if !validDate(f.From) || !validDate(f.To) {
			return nil, BadRequest("from and to must be YYYY-MM-DD")
		}
		// Widen by a day on each side so that UTC storage never hides an item
		// near the edge of the user's local range; the client filters exactly.
		from, _ := parseDate(f.From)
		to, _ := parseDate(f.To)
		from = from.AddDate(0, 0, -1)
		to = to.AddDate(0, 0, 2)
		q = q.Where("(completed = ? AND due_date >= ? AND due_date <= ?) OR (completed = ? AND completed_at >= ? AND completed_at < ?)",
			false, from.Format("2006-01-02"), to.Format("2006-01-02"), true, from, to)
	}
	err := q.Order("sort_order, id").Find(&items).Error
	return items, err
}

func (a *App) getItem(tx *gorm.DB, userID, itemID uint64) (*models.Item, error) {
	var it models.Item
	err := tx.Where("id = ? AND owner_id = ? AND deleted_at IS NULL", itemID, userID).First(&it).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &it, err
}

func maxOrder(tx *gorm.DB, column string, cardID uint64, fn string) (int64, bool, error) {
	var v *int64
	err := tx.Model(&models.Item{}).Where("card_id = ? AND deleted_at IS NULL", cardID).
		Select(fn + "(" + column + ")").Scan(&v).Error
	if err != nil || v == nil {
		return 0, false, err
	}
	return *v, true, nil
}

func nextOrder(tx *gorm.DB, column string, cardID uint64) (int64, error) {
	v, ok, err := maxOrder(tx, column, cardID, "MAX")
	if err != nil || !ok {
		return 0, err
	}
	return v + 1, nil
}

func (a *App) CreateItem(userID uint64, in ItemInput) (*models.Item, error) {
	title, err := cleanText(in.Title, 500, "title", true)
	if err != nil {
		return nil, err
	}
	note, err := cleanText(in.Note, 10000, "note", false)
	if err != nil {
		return nil, err
	}
	dueDate, dueTime := emptyToNil(in.DueDate), emptyToNil(in.DueTime)
	if err := validateDue(dueDate, dueTime); err != nil {
		return nil, err
	}
	var it *models.Item
	err = a.DB.Transaction(func(tx *gorm.DB) error {
		if _, err := a.getCard(tx, userID, in.CardID); err != nil {
			return err
		}
		var order int64
		if in.Position == "top" {
			v, ok, err := maxOrder(tx, "sort_order", in.CardID, "MIN")
			if err != nil {
				return err
			}
			if ok {
				order = v - 1
			}
		} else {
			if order, err = nextOrder(tx, "sort_order", in.CardID); err != nil {
				return err
			}
		}
		qOrder, err := nextOrder(tx, "quadrant_sort_order", in.CardID)
		if err != nil {
			return err
		}
		it = &models.Item{CardID: in.CardID, OwnerID: userID, Title: title, Note: note, DueDate: dueDate, DueTime: dueTime,
			Important: in.Important, Urgent: in.Urgent, SortOrder: order, QuadrantSortOrder: qOrder}
		return tx.Create(it).Error
	})
	return it, err
}

func (a *App) UpdateItem(userID, itemID uint64, p ItemPatch) (*models.Item, error) {
	err := a.DB.Transaction(func(tx *gorm.DB) error {
		it, err := a.getItem(tx, userID, itemID)
		if err != nil {
			return err
		}
		updates := map[string]any{"updated_at": now()}
		if p.Title != nil {
			t, err := cleanText(*p.Title, 500, "title", true)
			if err != nil {
				return err
			}
			updates["title"] = t
		}
		if p.Note != nil {
			n, err := cleanText(*p.Note, 10000, "note", false)
			if err != nil {
				return err
			}
			updates["note"] = n
		}
		dueDate, dueTime := it.DueDate, it.DueTime
		if p.DueDate.Set {
			dueDate = p.DueDate.Value
			if dueDate == nil {
				dueTime = nil // clearing the date also clears the time
			}
		}
		if p.DueTime.Set {
			dueTime = p.DueTime.Value
		}
		if p.DueDate.Set || p.DueTime.Set {
			if err := validateDue(dueDate, dueTime); err != nil {
				return err
			}
			updates["due_date"] = dueDate
			updates["due_time"] = dueTime
		}
		quadrantChanged := false
		if p.Important != nil && *p.Important != it.Important {
			updates["important"] = *p.Important
			quadrantChanged = true
		}
		if p.Urgent != nil && *p.Urgent != it.Urgent {
			updates["urgent"] = *p.Urgent
			quadrantChanged = true
		}
		targetCard := it.CardID
		if p.CardID != nil && *p.CardID != it.CardID {
			if _, err := a.getCard(tx, userID, *p.CardID); err != nil {
				return err
			}
			targetCard = *p.CardID
			order, err := nextOrder(tx, "sort_order", targetCard)
			if err != nil {
				return err
			}
			updates["card_id"] = targetCard
			updates["sort_order"] = order
			quadrantChanged = true
		}
		if quadrantChanged {
			q, err := nextOrder(tx, "quadrant_sort_order", targetCard)
			if err != nil {
				return err
			}
			updates["quadrant_sort_order"] = q
		}
		return tx.Model(&models.Item{}).Where("id = ?", it.ID).Updates(updates).Error
	})
	if err != nil {
		return nil, err
	}
	return a.getItem(a.DB, userID, itemID)
}

// SetCompleted marks an item done (recording the time) or not done.
func (a *App) SetCompleted(userID, itemID uint64, done bool) (*models.Item, error) {
	it, err := a.getItem(a.DB, userID, itemID)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{"completed": done, "updated_at": now()}
	if done {
		if !it.Completed {
			updates["completed_at"] = now()
		}
	} else {
		updates["completed_at"] = nil
	}
	if err := a.DB.Model(&models.Item{}).Where("id = ?", it.ID).Updates(updates).Error; err != nil {
		return nil, err
	}
	return a.getItem(a.DB, userID, itemID)
}

func (a *App) MoveItem(userID, itemID, cardID uint64) (*models.Item, error) {
	return a.UpdateItem(userID, itemID, ItemPatch{CardID: &cardID})
}

func (a *App) DeleteItem(userID, itemID uint64) error {
	it, err := a.getItem(a.DB, userID, itemID)
	if err != nil {
		return err
	}
	return a.DB.Model(&models.Item{}).Where("id = ?", it.ID).
		Updates(map[string]any{"deleted_at": now(), "deleted_with_card": false}).Error
}

type OrderList struct {
	CardID uint64   `json:"card_id"`
	IDs    []uint64 `json:"ids"`
}

// ReorderItems applies the new order of one or more cards in one transaction.
// Items listed under a different card are moved to that card.
func (a *App) ReorderItems(userID uint64, lists []OrderList) error {
	return a.DB.Transaction(func(tx *gorm.DB) error {
		for _, l := range lists {
			if _, err := a.getCard(tx, userID, l.CardID); err != nil {
				return err
			}
			for i, id := range l.IDs {
				res := tx.Model(&models.Item{}).Where("id = ? AND owner_id = ? AND deleted_at IS NULL", id, userID).
					Updates(map[string]any{"card_id": l.CardID, "sort_order": i, "updated_at": now()})
				if res.Error != nil {
					return res.Error
				}
				if res.RowsAffected == 0 {
					return ErrNotFound
				}
			}
		}
		return nil
	})
}

// QuadrantCell lists the order of one quadrant cell. When Important / Urgent
// are given they are written to every listed item; when omitted only the
// order changes (used when some items are only displayed as urgent).
type QuadrantCell struct {
	Important *bool    `json:"important"`
	Urgent    *bool    `json:"urgent"`
	IDs       []uint64 `json:"ids"`
}

// ReorderQuadrants stores the order inside quadrant cells and updates the
// important / urgent flags of the items listed in each cell.
func (a *App) ReorderQuadrants(userID uint64, cells []QuadrantCell) error {
	return a.DB.Transaction(func(tx *gorm.DB) error {
		for _, c := range cells {
			for i, id := range c.IDs {
				fields := map[string]any{"quadrant_sort_order": i, "updated_at": now()}
				if c.Important != nil {
					fields["important"] = *c.Important
				}
				if c.Urgent != nil {
					fields["urgent"] = *c.Urgent
				}
				res := tx.Model(&models.Item{}).Where("id = ? AND owner_id = ? AND deleted_at IS NULL", id, userID).
					Updates(fields)
				if res.Error != nil {
					return res.Error
				}
				if res.RowsAffected == 0 {
					return ErrNotFound
				}
			}
		}
		return nil
	})
}
