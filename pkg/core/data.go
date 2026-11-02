package core

import (
	"time"

	"gorm.io/gorm"

	"anntodo/pkg/models"
	"anntodo/pkg/version"
)

const exportFormat = 1

type ExportItem struct {
	Title             string     `json:"title"`
	Note              string     `json:"note"`
	DueDate           *string    `json:"due_date"`
	DueTime           *string    `json:"due_time"`
	Important         bool       `json:"important"`
	Urgent            bool       `json:"urgent"`
	Completed         bool       `json:"completed"`
	CompletedAt       *time.Time `json:"completed_at"`
	SortOrder         int64      `json:"sort_order"`
	QuadrantSortOrder int64      `json:"quadrant_sort_order"`
	CreatedAt         time.Time  `json:"created_at"`
}

type ExportCard struct {
	Name      string       `json:"name"`
	Color     string       `json:"color"`
	SortOrder int64        `json:"sort_order"`
	CreatedAt time.Time    `json:"created_at"`
	Items     []ExportItem `json:"items"`
}

type ExportData struct {
	App        string         `json:"app"`
	Format     int            `json:"format"`
	Version    string         `json:"version"`
	ExportedAt time.Time      `json:"exported_at"`
	Settings   map[string]any `json:"settings"`
	Cards      []ExportCard   `json:"cards"`
}

func (a *App) Export(userID uint64) (*ExportData, error) {
	cards, err := a.ListCards(userID)
	if err != nil {
		return nil, err
	}
	items, err := a.ListItems(userID, ItemFilter{})
	if err != nil {
		return nil, err
	}
	settings, err := a.GetUserSettings(userID)
	if err != nil {
		return nil, err
	}
	delete(settings, "current_card_id")
	delete(settings, "mcp_token") // legacy plaintext token, never export
	byCard := map[uint64][]ExportItem{}
	for _, it := range items {
		byCard[it.CardID] = append(byCard[it.CardID], ExportItem{Title: it.Title, Note: it.Note, DueDate: it.DueDate, DueTime: it.DueTime,
			Important: it.Important, Urgent: it.Urgent, Completed: it.Completed, CompletedAt: it.CompletedAt,
			SortOrder: it.SortOrder, QuadrantSortOrder: it.QuadrantSortOrder, CreatedAt: it.CreatedAt})
	}
	out := &ExportData{App: "AnnTodo", Format: exportFormat, Version: version.Version, ExportedAt: now(), Settings: settings, Cards: []ExportCard{}}
	for _, c := range cards {
		ec := ExportCard{Name: c.Name, Color: c.Color, SortOrder: c.SortOrder, CreatedAt: c.CreatedAt, Items: byCard[c.ID]}
		if ec.Items == nil {
			ec.Items = []ExportItem{}
		}
		out.Cards = append(out.Cards, ec)
	}
	return out, nil
}

type ImportResult struct {
	Cards int `json:"cards"`
	Items int `json:"items"`
}

// Import appends the cards of an export file after the user's existing cards.
// Existing data is never overwritten. Preferences from the file are applied.
func (a *App) Import(userID uint64, data *ExportData) (*ImportResult, error) {
	if data == nil || data.App != "AnnTodo" || data.Format != exportFormat {
		return nil, ErrInvalidImport
	}
	res := &ImportResult{}
	err := a.DB.Transaction(func(tx *gorm.DB) error {
		var max *int64
		if err := tx.Model(&models.Card{}).Where("owner_id = ? AND deleted_at IS NULL", userID).
			Select("MAX(sort_order)").Scan(&max).Error; err != nil {
			return err
		}
		base := int64(0)
		if max != nil {
			base = *max + 1
		}
		for ci, ec := range data.Cards {
			name, err := cleanText(ec.Name, 100, "card name", true)
			if err != nil {
				return ErrInvalidImport
			}
			if !validColor(ec.Color) {
				ec.Color = ""
			}
			card := &models.Card{OwnerID: userID, Name: name, Color: ec.Color, SortOrder: base + int64(ci)}
			if !ec.CreatedAt.IsZero() {
				card.CreatedAt = ec.CreatedAt.UTC()
			}
			if err := tx.Create(card).Error; err != nil {
				return err
			}
			res.Cards++
			for _, ei := range ec.Items {
				title, err := cleanText(ei.Title, 500, "title", true)
				if err != nil {
					return ErrInvalidImport
				}
				note, err := cleanText(ei.Note, 10000, "note", false)
				if err != nil {
					return ErrInvalidImport
				}
				dd, dt := emptyToNil(ei.DueDate), emptyToNil(ei.DueTime)
				if validateDue(dd, dt) != nil {
					return ErrInvalidImport
				}
				it := &models.Item{CardID: card.ID, OwnerID: userID, Title: title, Note: note, DueDate: dd, DueTime: dt,
					Important: ei.Important, Urgent: ei.Urgent, Completed: ei.Completed,
					SortOrder: ei.SortOrder, QuadrantSortOrder: ei.QuadrantSortOrder}
				if ei.Completed {
					t := now()
					if ei.CompletedAt != nil {
						t = ei.CompletedAt.UTC()
					}
					it.CompletedAt = &t
				}
				if !ei.CreatedAt.IsZero() {
					it.CreatedAt = ei.CreatedAt.UTC()
				}
				if err := tx.Create(it).Error; err != nil {
					return err
				}
				res.Items++
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(data.Settings) > 0 {
		delete(data.Settings, "current_card_id")
		delete(data.Settings, "mcp_token")
		if _, err := a.PatchUserSettings(userID, data.Settings); err != nil {
			return nil, err
		}
	}
	return res, nil
}
