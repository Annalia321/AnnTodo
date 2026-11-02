package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	"anntodo/pkg/core"
	mw "anntodo/pkg/middleware"
	"anntodo/pkg/settings"
	"anntodo/pkg/storage"
)

type client struct {
	t    *testing.T
	base string
	http *http.Client
}

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	cfg, err := settings.Load(filepath.Join(dir, "conf", "anntodo.ini"))
	if err != nil {
		t.Fatal(err)
	}
	db, err := storage.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{App: core.New(db, cfg), Limiter: mw.NewLoginLimiter()}
	ts := httptest.NewServer(s.Router())
	t.Cleanup(func() {
		ts.Close()
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return ts
}

func newClient(t *testing.T, base string) *client {
	jar, _ := cookiejar.New(nil)
	c := &client{t: t, base: base, http: &http.Client{Jar: jar}}
	c.do("GET", "/api/v1/system/status", nil, nil) // obtain CSRF cookie
	return c
}

func (c *client) csrf() string {
	u, _ := url.Parse(c.base)
	for _, ck := range c.http.Jar.Cookies(u) {
		if ck.Name == mw.CSRFCookie {
			return ck.Value
		}
	}
	return ""
}

func (c *client) do(method, path string, body any, out any) int {
	c.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, c.base+path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if tok := c.csrf(); tok != "" {
		req.Header.Set(mw.CSRFHeader, tok)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	if out != nil {
		_ = json.NewDecoder(resp.Body).Decode(out)
	}
	return resp.StatusCode
}

func (c *client) must(method, path string, body any, out any) {
	c.t.Helper()
	if code := c.do(method, path, body, out); code >= 300 {
		c.t.Fatalf("%s %s: status %d", method, path, code)
	}
}

type card struct {
	ID    uint64 `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type item struct {
	ID          uint64  `json:"id"`
	CardID      uint64  `json:"card_id"`
	Title       string  `json:"title"`
	DueDate     *string `json:"due_date"`
	DueTime     *string `json:"due_time"`
	Important   bool    `json:"important"`
	Urgent      bool    `json:"urgent"`
	Completed   bool    `json:"completed"`
	CompletedAt *string `json:"completed_at"`
	SortOrder   int64   `json:"sort_order"`
}

func TestFullFlow(t *testing.T) {
	ts := newTestServer(t)
	c := newClient(t, ts.URL)

	var status map[string]any
	c.must("GET", "/api/v1/system/status", nil, &status)
	if status["initialized"] != false {
		t.Fatal("expected uninitialized system")
	}

	// CSRF is enforced.
	raw := &http.Client{}
	resp, _ := raw.Post(ts.URL+"/api/v1/auth/setup", "application/json", bytes.NewBufferString(`{}`))
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected CSRF rejection, got %d", resp.StatusCode)
	}

	// Initialise the first administrator.
	if code := c.do("POST", "/api/v1/auth/setup", map[string]any{"username": "Admin", "password": "short"}, nil); code != 400 {
		t.Fatalf("weak password accepted: %d", code)
	}
	var me struct {
		User struct {
			ID       uint64 `json:"id"`
			Username string `json:"username"`
			Role     string `json:"role"`
		} `json:"user"`
	}
	c.must("POST", "/api/v1/auth/setup", map[string]any{"username": "Admin", "password": "password123"}, &me)
	if me.User.Username != "admin" || me.User.Role != "admin" {
		t.Fatalf("unexpected user %+v", me.User)
	}
	if code := c.do("POST", "/api/v1/auth/setup", map[string]any{"username": "other", "password": "password123"}, nil); code != 409 {
		t.Fatalf("second setup allowed: %d", code)
	}

	// Cards.
	var a, b card
	c.must("POST", "/api/v1/cards", map[string]any{"name": "Today", "color": "#1e88e5"}, &a)
	c.must("POST", "/api/v1/cards", map[string]any{"name": "Shopping", "color": "#43a047"}, &b)
	if code := c.do("POST", "/api/v1/cards", map[string]any{"name": "Bad", "color": "red"}, nil); code != 400 {
		t.Fatal("invalid color accepted")
	}
	c.must("PUT", "/api/v1/cards/order", map[string]any{"ids": []uint64{b.ID, a.ID}}, nil)
	var cards []card
	c.must("GET", "/api/v1/cards", nil, &cards)
	if len(cards) != 2 || cards[0].ID != b.ID {
		t.Fatalf("card order not applied: %+v", cards)
	}

	// Items.
	var i1, i2, i3 item
	c.must("POST", "/api/v1/items", map[string]any{"card_id": a.ID, "title": "one"}, &i1)
	c.must("POST", "/api/v1/items", map[string]any{"card_id": a.ID, "title": "two", "due_date": "2026-11-02", "due_time": "09:30"}, &i2)
	c.must("POST", "/api/v1/items", map[string]any{"card_id": a.ID, "title": "zero", "position": "top"}, &i3)
	if i3.SortOrder >= i1.SortOrder {
		t.Fatal("top position not applied")
	}
	if code := c.do("POST", "/api/v1/items", map[string]any{"card_id": a.ID, "title": "x", "due_time": "09:30"}, nil); code != 400 {
		t.Fatal("time without date accepted")
	}

	// Partial update: clear the date clears the time too.
	var upd item
	c.must("PUT", "/api/v1/items/"+itoa(i2.ID), map[string]any{"due_date": nil, "important": true}, &upd)
	if upd.DueDate != nil || upd.DueTime != nil || !upd.Important || upd.Title != "two" {
		t.Fatalf("patch failed: %+v", upd)
	}

	// Complete / uncomplete.
	var done item
	c.must("POST", "/api/v1/items/"+itoa(i1.ID)+"/complete", map[string]any{"completed": true}, &done)
	if !done.Completed || done.CompletedAt == nil {
		t.Fatal("complete failed")
	}
	c.must("POST", "/api/v1/items/"+itoa(i1.ID)+"/complete", map[string]any{"completed": false}, &done)
	if done.Completed || done.CompletedAt != nil {
		t.Fatal("uncomplete failed")
	}

	// Cross-card reorder.
	c.must("PUT", "/api/v1/items/order", map[string]any{"lists": []map[string]any{
		{"card_id": a.ID, "ids": []uint64{i3.ID, i1.ID}},
		{"card_id": b.ID, "ids": []uint64{i2.ID}},
	}}, nil)
	var bItems []item
	c.must("GET", "/api/v1/items?card_id="+itoa(b.ID), nil, &bItems)
	if len(bItems) != 1 || bItems[0].ID != i2.ID {
		t.Fatalf("cross-card move failed: %+v", bItems)
	}

	// Quadrants.
	c.must("PUT", "/api/v1/items/quadrant-order", map[string]any{"cells": []map[string]any{
		{"important": true, "urgent": true, "ids": []uint64{i3.ID}},
	}}, nil)
	var all []item
	c.must("GET", "/api/v1/items", nil, &all)
	for _, it := range all {
		if it.ID == i3.ID && (!it.Important || !it.Urgent) {
			t.Fatal("quadrant flags not set")
		}
	}

	// Delete a card -> trash -> restore.
	c.must("DELETE", "/api/v1/cards/"+itoa(a.ID), nil, nil)
	c.must("DELETE", "/api/v1/items/"+itoa(i2.ID), nil, nil)
	var trash struct {
		Cards []struct {
			ID        uint64 `json:"id"`
			ItemCount int64  `json:"item_count"`
		} `json:"cards"`
		Items []struct {
			ID uint64 `json:"id"`
		} `json:"items"`
	}
	c.must("GET", "/api/v1/trash", nil, &trash)
	if len(trash.Cards) != 1 || trash.Cards[0].ItemCount != 2 || len(trash.Items) != 1 {
		t.Fatalf("unexpected trash %+v", trash)
	}
	c.must("POST", "/api/v1/trash/card/"+itoa(a.ID)+"/restore", nil, nil)
	c.must("GET", "/api/v1/items?card_id="+itoa(a.ID), nil, &all)
	if len(all) != 2 {
		t.Fatalf("restore failed: %d items", len(all))
	}
	c.must("DELETE", "/api/v1/trash/item/"+itoa(i2.ID), nil, nil)

	// Calendar range query.
	var ranged []item
	c.must("POST", "/api/v1/items", map[string]any{"card_id": b.ID, "title": "due", "due_date": "2026-12-10"}, nil)
	c.must("GET", "/api/v1/items?from=2026-12-01&to=2026-12-31", nil, &ranged)
	if len(ranged) != 1 || ranged[0].Title != "due" {
		t.Fatalf("range query failed: %+v", ranged)
	}

	// Settings merge.
	var st map[string]any
	c.must("PUT", "/api/v1/settings", map[string]any{"theme": "dark", "current_card_id": b.ID}, &st)
	c.must("PUT", "/api/v1/settings", map[string]any{"language": "en"}, &st)
	if st["theme"] != "dark" || st["language"] != "en" {
		t.Fatalf("settings merge failed: %+v", st)
	}

	// Export then import duplicates the cards.
	var exported map[string]any
	c.must("GET", "/api/v1/data/export", nil, &exported)
	var imported map[string]int
	c.must("POST", "/api/v1/data/import", exported, &imported)
	if imported["cards"] != 2 {
		t.Fatalf("import result %+v", imported)
	}

	// Ownership: a second user cannot see or touch the admin's data.
	c.must("PUT", "/api/v1/admin/settings", map[string]any{"allow_register": true, "site_name": "My Todo"}, nil)
	u2 := newClient(t, ts.URL)
	u2.must("POST", "/api/v1/auth/register", map[string]any{"username": "bob", "password": "password123"}, nil)
	if code := u2.do("PUT", "/api/v1/cards/"+itoa(a.ID), map[string]any{"name": "hack"}, nil); code != 404 {
		t.Fatalf("foreign card editable: %d", code)
	}
	if code := u2.do("GET", "/api/v1/admin/users", nil, nil); code != 403 {
		t.Fatalf("non-admin reached admin api: %d", code)
	}
	var bobCards []card
	u2.must("GET", "/api/v1/cards", nil, &bobCards)
	if len(bobCards) != 0 {
		t.Fatal("bob sees foreign cards")
	}

	// Sessions: list and logout.
	var sessions []map[string]any
	c.must("GET", "/api/v1/account/sessions", nil, &sessions)
	if len(sessions) != 1 || sessions[0]["current"] != true {
		t.Fatalf("sessions %+v", sessions)
	}
	c.must("POST", "/api/v1/auth/logout", nil, nil)
	if code := c.do("GET", "/api/v1/cards", nil, nil); code != 401 {
		t.Fatalf("logged out session still valid: %d", code)
	}

	// Login rate limiting.
	for i := 0; i < 5; i++ {
		c.do("POST", "/api/v1/auth/login", map[string]any{"username": "admin", "password": "wrong-password"}, nil)
	}
	if code := c.do("POST", "/api/v1/auth/login", map[string]any{"username": "admin", "password": "password123"}, nil); code != 429 {
		t.Fatalf("expected rate limit, got %d", code)
	}
}

func itoa(n uint64) string {
	b, _ := json.Marshal(n)
	return string(b)
}
