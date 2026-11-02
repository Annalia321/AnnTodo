package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

type mcpClient struct {
	t     *testing.T
	url   string
	token string
	next  int
}

func (m *mcpClient) post(body string) (int, []byte, http.Header) {
	m.t.Helper()
	req, _ := http.NewRequest("POST", m.url, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if m.token != "" {
		req.Header.Set("Authorization", "Bearer "+m.token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		m.t.Fatal(err)
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)
	return resp.StatusCode, buf.Bytes(), resp.Header
}

// call invokes a tool and decodes its JSON text result into out.
func (m *mcpClient) call(name string, args map[string]any, out any) bool {
	m.t.Helper()
	m.next++
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": m.next, "method": "tools/call",
		"params": map[string]any{"name": name, "arguments": args}})
	code, raw, _ := m.post(string(body))
	if code != 200 {
		m.t.Fatalf("%s: status %d %s", name, code, raw)
	}
	var r struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
		Error *rpcError `json:"error"`
	}
	if err := json.Unmarshal(raw, &r); err != nil || r.Error != nil || len(r.Result.Content) != 1 {
		m.t.Fatalf("%s: bad response %s", name, raw)
	}
	if r.Result.IsError {
		if out != nil {
			*(out.(*string)) = r.Result.Content[0].Text
		}
		return false
	}
	if out != nil {
		if err := json.Unmarshal([]byte(r.Result.Content[0].Text), out); err != nil {
			m.t.Fatalf("%s: %v in %s", name, err, r.Result.Content[0].Text)
		}
	}
	return true
}

func TestMCP(t *testing.T) {
	ts := newTestServer(t)
	c := newClient(t, ts.URL)
	c.must("POST", "/api/v1/auth/setup", map[string]any{"username": "admin", "password": "password123"}, nil)

	var info struct {
		Exists bool `json:"exists"`
	}
	c.must("GET", "/api/v1/mcp/token", nil, &info)
	if info.Exists {
		t.Fatal("token exists before generation")
	}
	var gen struct {
		Token string `json:"token"`
		Info  struct {
			Exists bool   `json:"exists"`
			Hint   string `json:"hint"`
		} `json:"info"`
	}
	c.must("POST", "/api/v1/mcp/token", map[string]any{"timezone": "Asia/Shanghai"}, &gen)
	if !strings.HasPrefix(gen.Token, "sk-") || !gen.Info.Exists || !strings.HasPrefix(gen.Token, gen.Info.Hint) {
		t.Fatalf("unexpected token response %+v", gen)
	}

	m := &mcpClient{t: t, url: ts.URL + "/mcp"}
	// No or wrong token: 401 with WWW-Authenticate.
	code, _, hdr := m.post(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)
	if code != 401 || !strings.Contains(hdr.Get("WWW-Authenticate"), "Bearer") {
		t.Fatalf("expected 401, got %d", code)
	}
	m.token = "sk-" + strings.Repeat("0", 64)
	if code, _, _ := m.post(`{"jsonrpc":"2.0","id":1,"method":"ping"}`); code != 401 {
		t.Fatalf("wrong token accepted: %d", code)
	}
	m.token = gen.Token

	code, raw, _ := m.post(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`)
	if code != 200 || !strings.Contains(string(raw), `"protocolVersion":"2025-03-26"`) || !strings.Contains(string(raw), `"tools"`) {
		t.Fatalf("initialize: %d %s", code, raw)
	}
	if code, _, _ := m.post(`{"jsonrpc":"2.0","method":"notifications/initialized"}`); code != 202 {
		t.Fatalf("notification: %d", code)
	}
	code, raw, _ = m.post(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	if code != 200 || !strings.Contains(string(raw), `"create_item"`) || !strings.Contains(string(raw), `"inputSchema"`) {
		t.Fatalf("tools/list: %d %s", code, raw)
	}
	_, raw, _ = m.post(`{"jsonrpc":"2.0","id":3,"method":"nope"}`)
	if !strings.Contains(string(raw), "-32601") {
		t.Fatalf("unknown method: %s", raw)
	}
	if code, _, _ := m.post(`{bad`); code != 400 {
		t.Fatalf("parse error: %d", code)
	}
	code, raw, _ = m.post(`[{"jsonrpc":"2.0","id":1,"method":"ping"},{"jsonrpc":"2.0","method":"notifications/x"}]`)
	if code != 200 || !strings.HasPrefix(string(raw), "[") {
		t.Fatalf("batch: %d %s", code, raw)
	}

	var cardOut mcpCard
	m.call("create_card", map[string]any{"name": "Work"}, &cardOut)
	if cardOut.ID == 0 || cardOut.Color != cardPalette[0] {
		t.Fatalf("create_card: %+v", cardOut)
	}
	var it mcpItem
	m.call("create_item", map[string]any{"card_id": float64(cardOut.ID), "title": "Report",
		"due_date": "2026-11-02", "due_time": "09:30", "important": true}, &it)
	if it.ID == 0 || *it.DueDate != "2026-11-02" || *it.DueTime != "09:30" || !it.Important || it.CardName != "Work" {
		t.Fatalf("create_item: %+v", it)
	}
	// Stored in UTC, as the web app expects.
	var items []item
	c.must("GET", "/api/v1/items", nil, &items)
	if len(items) != 1 || *items[0].DueTime != "01:30" {
		t.Fatalf("stored due time: %+v", items)
	}

	m.call("update_item", map[string]any{"id": strconv.FormatUint(it.ID, 10), "due_time": ""}, &it)
	if it.DueTime != nil || *it.DueDate != "2026-11-02" {
		t.Fatalf("update_item all-day: %+v", it)
	}
	m.call("complete_item", map[string]any{"id": it.ID}, &it)
	if !it.Completed || it.CompletedAt == nil {
		t.Fatalf("complete_item: %+v", it)
	}
	var list itemList
	m.call("list_items", map[string]any{}, &list)
	if list.Total != 0 {
		t.Fatalf("open items: %+v", list)
	}
	m.call("search_items", map[string]any{"query": "rep"}, &list)
	if list.Total != 1 {
		t.Fatalf("search: %+v", list)
	}
	var errText string
	if m.call("create_item", map[string]any{"card_id": 999, "title": "x"}, &errText) || !strings.HasPrefix(errText, "error: not found") {
		t.Fatalf("expected tool error, got %q", errText)
	}
	m.call("delete_item", map[string]any{"id": it.ID}, nil)
	var trash struct {
		Items []map[string]any `json:"items"`
	}
	m.call("list_trash", nil, &trash)
	if len(trash.Items) != 1 {
		t.Fatalf("trash: %+v", trash)
	}
	m.call("restore_from_trash", map[string]any{"type": "item", "id": it.ID}, nil)
	m.call("get_item", map[string]any{"id": it.ID}, &it)

	// Revoking the token locks the client out.
	c.must("DELETE", "/api/v1/mcp/token", nil, nil)
	if code, _, _ := m.post(`{"jsonrpc":"2.0","id":9,"method":"ping"}`); code != 401 {
		t.Fatalf("revoked token accepted: %d", code)
	}
}
