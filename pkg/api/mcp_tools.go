package api

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"anntodo/pkg/core"
	"anntodo/pkg/models"
)

// ---------- argument helpers ----------

// mcpArgError is an invalid tool argument; its message is shown to the model.
type mcpArgError struct{ msg string }

func (e *mcpArgError) Error() string { return e.msg }

func argErr(format string, a ...any) error { return &mcpArgError{fmt.Sprintf(format, a...)} }

func decodeArgs(raw json.RawMessage, v any) error {
	if err := json.Unmarshal(raw, v); err != nil {
		return argErr("invalid arguments: %v", err)
	}
	return nil
}

// flexID accepts 12, 12.0 or "12": models are not always strict about types.
type flexID uint64

func (f *flexID) UnmarshalJSON(b []byte) error {
	s := strings.Trim(strings.TrimSpace(string(b)), `"`)
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	if v, err := strconv.ParseUint(s, 10, 64); err == nil {
		*f = flexID(v)
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v < 0 || v != math.Trunc(v) || v > 1<<53 {
		return fmt.Errorf("%s is not a valid id", string(b))
	}
	*f = flexID(v)
	return nil
}

// flexBool accepts true/false or "true"/"false".
type flexBool bool

func (f *flexBool) UnmarshalJSON(b []byte) error {
	switch strings.ToLower(strings.Trim(strings.TrimSpace(string(b)), `"`)) {
	case "true", "1":
		*f = true
	case "false", "0", "null", "":
		*f = false
	default:
		return fmt.Errorf("%s is not a boolean", string(b))
	}
	return nil
}

func boolPtr(f *flexBool) *bool {
	if f == nil {
		return nil
	}
	v := bool(*f)
	return &v
}

// optString reads a string-or-null value; empty strings become nil.
func optString(raw json.RawMessage) (*string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, argErr("expected a string or null, got %s", string(raw))
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	return &s, nil
}

func requireID(id flexID, name string) (uint64, error) {
	if id == 0 {
		return 0, argErr("%s is required", name)
	}
	return uint64(id), nil
}

// ---------- dates and times ----------

const (
	layoutDate     = "2006-01-02"
	layoutDateTime = "2006-01-02 15:04"
)

func validLocalDate(d string) bool {
	if len(d) != 10 {
		return false
	}
	_, err := time.Parse(layoutDate, d)
	return err == nil
}

// localDue converts a stored due date/time (UTC when a time is set) to the
// user's timezone. All-day items have no time and are not converted.
func (m *mcpSession) localDue(d, t *string) (*string, *string) {
	if d == nil || t == nil {
		return d, t
	}
	ts, err := time.ParseInLocation(layoutDateTime, *d+" "+*t, time.UTC)
	if err != nil {
		return d, t
	}
	l := ts.In(m.loc)
	ld, lt := l.Format(layoutDate), l.Format("15:04")
	return &ld, &lt
}

// storedDue validates a due date/time given in the user's timezone and
// converts it to the stored form.
func (m *mcpSession) storedDue(d, t *string) (*string, *string, error) {
	if d == nil {
		if t != nil {
			return nil, nil, argErr("due_time requires due_date")
		}
		return nil, nil, nil
	}
	if !validLocalDate(*d) {
		return nil, nil, argErr("due_date must be YYYY-MM-DD, got %q", *d)
	}
	if t == nil {
		return d, nil, nil
	}
	tm := *t
	if len(tm) == 8 && tm[5] == ':' {
		tm = tm[:5] // drop seconds
	}
	ts, err := time.ParseInLocation(layoutDateTime, *d+" "+tm, m.loc)
	if err != nil {
		return nil, nil, argErr("due_time must be HH:MM (24-hour), got %q", *t)
	}
	u := ts.UTC()
	sd, st := u.Format(layoutDate), u.Format("15:04")
	return &sd, &st, nil
}

func (m *mcpSession) ts(t time.Time) string { return t.In(m.loc).Format(time.RFC3339) }

// ---------- output shapes ----------

type mcpCard struct {
	ID             uint64 `json:"id"`
	Name           string `json:"name"`
	Color          string `json:"color"`
	OpenItems      int64  `json:"open_items"`
	CompletedItems int64  `json:"completed_items"`
}

type mcpItem struct {
	ID          uint64  `json:"id"`
	CardID      uint64  `json:"card_id"`
	CardName    string  `json:"card_name"`
	Title       string  `json:"title"`
	Note        string  `json:"note,omitempty"`
	DueDate     *string `json:"due_date"`
	DueTime     *string `json:"due_time"`
	Important   bool    `json:"important"`
	Urgent      bool    `json:"urgent"`
	Completed   bool    `json:"completed"`
	CompletedAt *string `json:"completed_at"`
	CreatedAt   string  `json:"created_at"`
}

func (m *mcpSession) cards() ([]models.Card, map[uint64]string, error) {
	cards, err := m.app.ListCards(m.user.ID)
	if err != nil {
		return nil, nil, err
	}
	names := make(map[uint64]string, len(cards))
	for _, c := range cards {
		names[c.ID] = c.Name
	}
	return cards, names, nil
}

func (m *mcpSession) cardList() ([]mcpCard, error) {
	cards, _, err := m.cards()
	if err != nil {
		return nil, err
	}
	counts, err := m.app.CardItemCounts(m.user.ID)
	if err != nil {
		return nil, err
	}
	out := make([]mcpCard, 0, len(cards))
	for _, c := range cards {
		n := counts[c.ID]
		out = append(out, mcpCard{ID: c.ID, Name: c.Name, Color: c.Color, OpenItems: n[0], CompletedItems: n[1]})
	}
	return out, nil
}

func (m *mcpSession) card(c *models.Card) mcpCard {
	n, _ := m.app.CardItemCounts(m.user.ID)
	return mcpCard{ID: c.ID, Name: c.Name, Color: c.Color, OpenItems: n[c.ID][0], CompletedItems: n[c.ID][1]}
}

func (m *mcpSession) item(it *models.Item, cardName string) mcpItem {
	d, t := m.localDue(it.DueDate, it.DueTime)
	out := mcpItem{ID: it.ID, CardID: it.CardID, CardName: cardName, Title: it.Title, Note: it.Note,
		DueDate: d, DueTime: t, Important: it.Important, Urgent: it.Urgent, Completed: it.Completed,
		CreatedAt: m.ts(it.CreatedAt)}
	if it.CompletedAt != nil {
		s := m.ts(*it.CompletedAt)
		out.CompletedAt = &s
	}
	return out
}

// itemOut renders one item with its card name.
func (m *mcpSession) itemOut(it *models.Item) (mcpItem, error) {
	_, names, err := m.cards()
	if err != nil {
		return mcpItem{}, err
	}
	return m.item(it, names[it.CardID]), nil
}

// ---------- item filtering ----------

type itemFilterArgs struct {
	CardID    flexID    `json:"card_id"`
	Status    string    `json:"status"`
	DueFrom   string    `json:"due_from"`
	DueTo     string    `json:"due_to"`
	Important *flexBool `json:"important"`
	Urgent    *flexBool `json:"urgent"`
	Limit     flexID    `json:"limit"`
}

type itemList struct {
	Items     []mcpItem `json:"items"`
	Total     int       `json:"total"`
	Truncated bool      `json:"truncated"`
}

func (m *mcpSession) filterItems(items []models.Item, f itemFilterArgs, defStatus string) (*itemList, error) {
	status := strings.ToLower(strings.TrimSpace(f.Status))
	if status == "" {
		status = defStatus
	}
	if status != "open" && status != "completed" && status != "all" {
		return nil, argErr("status must be open, completed or all")
	}
	if f.DueFrom != "" && !validLocalDate(f.DueFrom) {
		return nil, argErr("due_from must be YYYY-MM-DD")
	}
	if f.DueTo != "" && !validLocalDate(f.DueTo) {
		return nil, argErr("due_to must be YYYY-MM-DD")
	}
	limit := int(f.Limit)
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	cards, names, err := m.cards()
	if err != nil {
		return nil, err
	}
	if f.CardID != 0 {
		if _, ok := names[uint64(f.CardID)]; !ok {
			return nil, core.ErrNotFound
		}
	}
	rank := make(map[uint64]int, len(cards))
	for i, c := range cards {
		rank[c.ID] = i
	}
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if rank[a.CardID] != rank[b.CardID] {
			return rank[a.CardID] < rank[b.CardID]
		}
		if a.Completed != b.Completed {
			return !a.Completed
		}
		return a.SortOrder < b.SortOrder
	})
	out := &itemList{Items: []mcpItem{}}
	for i := range items {
		it := &items[i]
		if _, live := names[it.CardID]; !live {
			continue
		}
		if f.CardID != 0 && it.CardID != uint64(f.CardID) {
			continue
		}
		if (status == "open" && it.Completed) || (status == "completed" && !it.Completed) {
			continue
		}
		if f.Important != nil && it.Important != bool(*f.Important) {
			continue
		}
		if f.Urgent != nil && it.Urgent != bool(*f.Urgent) {
			continue
		}
		v := m.item(it, names[it.CardID])
		if f.DueFrom != "" || f.DueTo != "" {
			if v.DueDate == nil || (f.DueFrom != "" && *v.DueDate < f.DueFrom) || (f.DueTo != "" && *v.DueDate > f.DueTo) {
				continue
			}
		}
		out.Total++
		if len(out.Items) < limit {
			out.Items = append(out.Items, v)
		} else {
			out.Truncated = true
		}
	}
	return out, nil
}

// ---------- tool implementations ----------

// cardPalette matches the card colors offered by the web app.
var cardPalette = []string{"#1976D2", "#0097A7", "#00897B", "#43A047", "#7CB342", "#F9A825",
	"#FB8C00", "#E53935", "#D81B60", "#8E24AA", "#6D4C41", "#546E7A"}

const trashNote = "moved to the trash; restore_from_trash can bring it back within 30 days"

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}

func toolGetContext(m *mcpSession, _ json.RawMessage) (any, error) {
	cards, err := m.cardList()
	if err != nil {
		return nil, err
	}
	now := time.Now().In(m.loc)
	return map[string]any{
		"user":       map[string]any{"id": m.user.ID, "username": m.user.Username, "nickname": m.user.Nickname},
		"timezone":   m.loc.String(),
		"utc_offset": now.Format("-07:00"),
		"now":        now.Format(time.RFC3339),
		"today":      now.Format(layoutDate),
		"weekday":    now.Weekday().String(),
		"cards":      cards,
	}, nil
}

func toolListCards(m *mcpSession, _ json.RawMessage) (any, error) {
	cards, err := m.cardList()
	if err != nil {
		return nil, err
	}
	return map[string]any{"cards": cards}, nil
}

func toolCreateCard(m *mcpSession, raw json.RawMessage) (any, error) {
	var a struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := decodeArgs(raw, &a); err != nil {
		return nil, err
	}
	color := strings.ToUpper(strings.TrimSpace(a.Color))
	if color == "" {
		cards, _, err := m.cards()
		if err != nil {
			return nil, err
		}
		color = cardPalette[len(cards)%len(cardPalette)]
	}
	c, err := m.app.CreateCard(m.user.ID, core.CardInput{Name: a.Name, Color: color})
	if err != nil {
		return nil, err
	}
	return m.card(c), nil
}

func toolUpdateCard(m *mcpSession, raw json.RawMessage) (any, error) {
	var a struct {
		ID    flexID  `json:"id"`
		Name  *string `json:"name"`
		Color *string `json:"color"`
	}
	if err := decodeArgs(raw, &a); err != nil {
		return nil, err
	}
	id, err := requireID(a.ID, "id")
	if err != nil {
		return nil, err
	}
	if a.Name == nil && a.Color == nil {
		return nil, argErr("nothing to update: give name and/or color")
	}
	if a.Color != nil {
		v := strings.ToUpper(strings.TrimSpace(*a.Color))
		a.Color = &v
	}
	c, err := m.app.UpdateCard(m.user.ID, id, core.CardPatch{Name: a.Name, Color: a.Color})
	if err != nil {
		return nil, err
	}
	return m.card(c), nil
}

func toolDeleteCard(m *mcpSession, raw json.RawMessage) (any, error) {
	var a struct {
		ID flexID `json:"id"`
	}
	if err := decodeArgs(raw, &a); err != nil {
		return nil, err
	}
	id, err := requireID(a.ID, "id")
	if err != nil {
		return nil, err
	}
	if err := m.app.DeleteCard(m.user.ID, id); err != nil {
		return nil, err
	}
	return map[string]any{"deleted": true, "card_id": id, "message": "the card and its items were " + trashNote}, nil
}

func toolListItems(m *mcpSession, raw json.RawMessage) (any, error) {
	var a itemFilterArgs
	if err := decodeArgs(raw, &a); err != nil {
		return nil, err
	}
	items, err := m.app.ListItems(m.user.ID, core.ItemFilter{})
	if err != nil {
		return nil, err
	}
	return m.filterItems(items, a, "open")
}

func toolSearchItems(m *mcpSession, raw json.RawMessage) (any, error) {
	var a struct {
		Query string `json:"query"`
		itemFilterArgs
	}
	if err := decodeArgs(raw, &a); err != nil {
		return nil, err
	}
	if strings.TrimSpace(a.Query) == "" {
		return nil, argErr("query is required")
	}
	items, err := m.app.SearchItems(m.user.ID, a.Query, uint64(a.CardID), 5000)
	if err != nil {
		return nil, err
	}
	return m.filterItems(items, a.itemFilterArgs, "all")
}

func toolGetItem(m *mcpSession, raw json.RawMessage) (any, error) {
	var a struct {
		ID flexID `json:"id"`
	}
	if err := decodeArgs(raw, &a); err != nil {
		return nil, err
	}
	id, err := requireID(a.ID, "id")
	if err != nil {
		return nil, err
	}
	it, err := m.app.GetItem(m.user.ID, id)
	if err != nil {
		return nil, err
	}
	return m.itemOut(it)
}

func toolCreateItem(m *mcpSession, raw json.RawMessage) (any, error) {
	var a struct {
		CardID    flexID   `json:"card_id"`
		Title     string   `json:"title"`
		Note      string   `json:"note"`
		DueDate   *string  `json:"due_date"`
		DueTime   *string  `json:"due_time"`
		Important flexBool `json:"important"`
		Urgent    flexBool `json:"urgent"`
		Position  string   `json:"position"`
	}
	if err := decodeArgs(raw, &a); err != nil {
		return nil, err
	}
	cardID, err := requireID(a.CardID, "card_id")
	if err != nil {
		return nil, err
	}
	d, t, err := m.storedDue(trimPtr(a.DueDate), trimPtr(a.DueTime))
	if err != nil {
		return nil, err
	}
	pos := strings.ToLower(strings.TrimSpace(a.Position))
	if pos == "" {
		if s, err := m.app.GetUserSettings(m.user.ID); err == nil {
			if v, _ := s["new_item_position"].(string); v == "top" {
				pos = "top"
			}
		}
	}
	if pos != "" && pos != "top" && pos != "bottom" {
		return nil, argErr("position must be top or bottom")
	}
	it, err := m.app.CreateItem(m.user.ID, core.ItemInput{CardID: cardID, Title: a.Title, Note: a.Note,
		DueDate: d, DueTime: t, Important: bool(a.Important), Urgent: bool(a.Urgent), Position: pos})
	if err != nil {
		return nil, err
	}
	return m.itemOut(it)
}

func toolUpdateItem(m *mcpSession, raw json.RawMessage) (any, error) {
	var a struct {
		ID        flexID    `json:"id"`
		CardID    *flexID   `json:"card_id"`
		Title     *string   `json:"title"`
		Note      *string   `json:"note"`
		Important *flexBool `json:"important"`
		Urgent    *flexBool `json:"urgent"`
	}
	var present map[string]json.RawMessage
	if err := decodeArgs(raw, &a); err != nil {
		return nil, err
	}
	if err := decodeArgs(raw, &present); err != nil {
		return nil, err
	}
	id, err := requireID(a.ID, "id")
	if err != nil {
		return nil, err
	}
	p := core.ItemPatch{Title: a.Title, Note: a.Note, Important: boolPtr(a.Important), Urgent: boolPtr(a.Urgent)}
	if a.CardID != nil && *a.CardID != 0 {
		v := uint64(*a.CardID)
		p.CardID = &v
	}
	rawDate, dateSet := present["due_date"]
	rawTime, timeSet := present["due_time"]
	if dateSet || timeSet {
		cur, err := m.app.GetItem(m.user.ID, id)
		if err != nil {
			return nil, err
		}
		d, t := m.localDue(cur.DueDate, cur.DueTime)
		if dateSet {
			if d, err = optString(rawDate); err != nil {
				return nil, err
			}
			if d == nil {
				t = nil // clearing the date also clears the time
			}
		}
		if timeSet {
			if t, err = optString(rawTime); err != nil {
				return nil, err
			}
		}
		sd, st, err := m.storedDue(d, t)
		if err != nil {
			return nil, err
		}
		p.DueDate = core.OptString{Set: true, Value: sd}
		p.DueTime = core.OptString{Set: true, Value: st}
	}
	if p.Title == nil && p.Note == nil && p.Important == nil && p.Urgent == nil && p.CardID == nil && !p.DueDate.Set {
		return nil, argErr("nothing to update: give at least one of title, note, due_date, due_time, important, urgent, card_id")
	}
	it, err := m.app.UpdateItem(m.user.ID, id, p)
	if err != nil {
		return nil, err
	}
	return m.itemOut(it)
}

func toolCompleteItem(m *mcpSession, raw json.RawMessage) (any, error) {
	var a struct {
		ID        flexID    `json:"id"`
		Completed *flexBool `json:"completed"`
	}
	if err := decodeArgs(raw, &a); err != nil {
		return nil, err
	}
	id, err := requireID(a.ID, "id")
	if err != nil {
		return nil, err
	}
	done := a.Completed == nil || bool(*a.Completed)
	it, err := m.app.SetCompleted(m.user.ID, id, done)
	if err != nil {
		return nil, err
	}
	return m.itemOut(it)
}

func toolMoveItem(m *mcpSession, raw json.RawMessage) (any, error) {
	var a struct {
		ID     flexID `json:"id"`
		CardID flexID `json:"card_id"`
	}
	if err := decodeArgs(raw, &a); err != nil {
		return nil, err
	}
	id, err := requireID(a.ID, "id")
	if err != nil {
		return nil, err
	}
	cardID, err := requireID(a.CardID, "card_id")
	if err != nil {
		return nil, err
	}
	it, err := m.app.MoveItem(m.user.ID, id, cardID)
	if err != nil {
		return nil, err
	}
	return m.itemOut(it)
}

func toolDeleteItem(m *mcpSession, raw json.RawMessage) (any, error) {
	var a struct {
		ID flexID `json:"id"`
	}
	if err := decodeArgs(raw, &a); err != nil {
		return nil, err
	}
	id, err := requireID(a.ID, "id")
	if err != nil {
		return nil, err
	}
	if err := m.app.DeleteItem(m.user.ID, id); err != nil {
		return nil, err
	}
	return map[string]any{"deleted": true, "item_id": id, "message": "the item was " + trashNote}, nil
}

func toolListTrash(m *mcpSession, _ json.RawMessage) (any, error) {
	t, err := m.app.GetTrash(m.user.ID)
	if err != nil {
		return nil, err
	}
	cards := make([]map[string]any, 0, len(t.Cards))
	for _, c := range t.Cards {
		cards = append(cards, map[string]any{"id": c.ID, "name": c.Name, "color": c.Color, "item_count": c.ItemCount,
			"deleted_at": m.ts(c.DeletedAt), "purged_after": m.ts(c.DeletedAt.Add(core.TrashRetention))})
	}
	items := make([]map[string]any, 0, len(t.Items))
	for _, it := range t.Items {
		items = append(items, map[string]any{"id": it.ID, "title": it.Title, "card_id": it.CardID, "card_name": it.CardName,
			"deleted_at": m.ts(it.DeletedAt), "purged_after": m.ts(it.DeletedAt.Add(core.TrashRetention))})
	}
	return map[string]any{"cards": cards, "items": items}, nil
}

func toolRestoreFromTrash(m *mcpSession, raw json.RawMessage) (any, error) {
	var a struct {
		Type string `json:"type"`
		ID   flexID `json:"id"`
	}
	if err := decodeArgs(raw, &a); err != nil {
		return nil, err
	}
	id, err := requireID(a.ID, "id")
	if err != nil {
		return nil, err
	}
	typ := strings.ToLower(strings.TrimSpace(a.Type))
	switch typ {
	case "card":
		err = m.app.RestoreCard(m.user.ID, id)
	case "item":
		err = m.app.RestoreItem(m.user.ID, id)
	default:
		return nil, argErr("type must be card or item")
	}
	if err != nil {
		return nil, err
	}
	return map[string]any{"restored": true, "type": typ, "id": id}, nil
}
