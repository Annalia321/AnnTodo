package api

import "encoding/json"

// MCP tool definitions advertised by tools/list.

type mcpAnnotations struct {
	ReadOnlyHint    bool `json:"readOnlyHint"`
	DestructiveHint bool `json:"destructiveHint"`
	IdempotentHint  bool `json:"idempotentHint"`
	OpenWorldHint   bool `json:"openWorldHint"`
}

type mcpTool struct {
	Name        string         `json:"name"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	Annotations mcpAnnotations `json:"annotations"`
	run         func(*mcpSession, json.RawMessage) (any, error)
}

var (
	annRead    = mcpAnnotations{ReadOnlyHint: true, IdempotentHint: true}
	annCreate  = mcpAnnotations{}
	annUpdate  = mcpAnnotations{IdempotentHint: true}
	annDelete  = mcpAnnotations{DestructiveHint: true, IdempotentHint: true}
	annRestore = mcpAnnotations{IdempotentHint: true}
)

func schema(required []string, props map[string]any) map[string]any {
	if props == nil {
		props = map[string]any{}
	}
	s := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}

func sInt(d string) map[string]any  { return map[string]any{"type": "integer", "description": d} }
func sStr(d string) map[string]any  { return map[string]any{"type": "string", "description": d} }
func sBool(d string) map[string]any { return map[string]any{"type": "boolean", "description": d} }
func sEnum(d string, v ...string) map[string]any {
	return map[string]any{"type": "string", "enum": v, "description": d}
}

func merge(a, b map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

const (
	descDate = "Due date YYYY-MM-DD in the user's timezone."
	descTime = "Due time HH:MM (24-hour) in the user's timezone. Requires due_date; omit for an all-day item."
)

func filterProps(defStatus string) map[string]any {
	return map[string]any{
		"card_id":   sInt("Only items of this card."),
		"status":    sEnum("Completion filter (default "+defStatus+").", "open", "completed", "all"),
		"due_from":  sStr("Only items due on or after this date (YYYY-MM-DD). Items without a due date are excluded when due_from or due_to is set."),
		"due_to":    sStr("Only items due on or before this date (YYYY-MM-DD)."),
		"important": sBool("Only items with this important flag."),
		"urgent":    sBool("Only items with this urgent flag."),
		"limit":     sInt("Maximum items to return (default 100, max 500)."),
	}
}

var mcpTools = []*mcpTool{
	{Name: "get_context", Title: "Get context", Annotations: annRead, run: toolGetContext,
		Description: "Returns the user, the current date, time, weekday and timezone, and all cards with their ids and item counts. Call this first.",
		InputSchema: schema(nil, nil)},
	{Name: "list_cards", Title: "List cards", Annotations: annRead, run: toolListCards,
		Description: "Lists the user's cards (to-do lists) in display order with open and completed item counts.",
		InputSchema: schema(nil, nil)},
	{Name: "create_card", Title: "Create card", Annotations: annCreate, run: toolCreateCard,
		Description: "Creates a card (to-do list) at the end of the card list.",
		InputSchema: schema([]string{"name"}, map[string]any{
			"name":  sStr("Card name, up to 100 characters."),
			"color": sStr("Color as #RRGGBB. Omit to pick the next color of the app palette."),
		})},
	{Name: "update_card", Title: "Update card", Annotations: annUpdate, run: toolUpdateCard,
		Description: "Renames a card or changes its color.",
		InputSchema: schema([]string{"id"}, map[string]any{
			"id":    sInt("Card id."),
			"name":  sStr("New name."),
			"color": sStr("New color as #RRGGBB."),
		})},
	{Name: "delete_card", Title: "Delete card", Annotations: annDelete, run: toolDeleteCard,
		Description: "Moves a card and all of its items to the trash (kept 30 days). Confirm with the user first.",
		InputSchema: schema([]string{"id"}, map[string]any{"id": sInt("Card id.")})},
	{Name: "list_items", Title: "List items", Annotations: annRead, run: toolListItems,
		Description: "Lists to-do items, ordered by card and position. By default only open (not completed) items are returned.",
		InputSchema: schema(nil, filterProps("open"))},
	{Name: "search_items", Title: "Search items", Annotations: annRead, run: toolSearchItems,
		Description: "Finds items whose title or note contains the text (case-insensitive).",
		InputSchema: schema([]string{"query"}, merge(filterProps("all"), map[string]any{"query": sStr("Text to search for.")}))},
	{Name: "get_item", Title: "Get item", Annotations: annRead, run: toolGetItem,
		Description: "Returns one item with its note.",
		InputSchema: schema([]string{"id"}, map[string]any{"id": sInt("Item id.")})},
	{Name: "create_item", Title: "Create item", Annotations: annCreate, run: toolCreateItem,
		Description: "Adds a to-do item to a card.",
		InputSchema: schema([]string{"card_id", "title"}, map[string]any{
			"card_id":   sInt("Card id (see get_context or list_cards)."),
			"title":     sStr("Item title, up to 500 characters."),
			"note":      sStr("Optional note."),
			"due_date":  sStr(descDate),
			"due_time":  sStr(descTime),
			"important": sBool("Important flag (Eisenhower matrix)."),
			"urgent":    sBool("Urgent flag (Eisenhower matrix)."),
			"position":  sEnum("Add at the top or bottom of the card. Default: the user's setting.", "top", "bottom"),
		})},
	{Name: "update_item", Title: "Update item", Annotations: annUpdate, run: toolUpdateItem,
		Description: "Changes fields of an item; only the given fields change. Set due_date to \"\" to remove the due date, or due_time to \"\" to make the item all-day.",
		InputSchema: schema([]string{"id"}, map[string]any{
			"id":        sInt("Item id."),
			"title":     sStr("New title."),
			"note":      sStr("New note (\"\" clears it)."),
			"due_date":  sStr(descDate + " \"\" removes the due date and time."),
			"due_time":  sStr("Due time HH:MM (24-hour) in the user's timezone. \"\" makes the item all-day."),
			"important": sBool("Important flag."),
			"urgent":    sBool("Urgent flag."),
			"card_id":   sInt("Move the item to this card."),
		})},
	{Name: "complete_item", Title: "Complete item", Annotations: annUpdate, run: toolCompleteItem,
		Description: "Marks an item completed, or not completed with completed=false.",
		InputSchema: schema([]string{"id"}, map[string]any{
			"id":        sInt("Item id."),
			"completed": sBool("true (default) to complete, false to reopen."),
		})},
	{Name: "move_item", Title: "Move item", Annotations: annUpdate, run: toolMoveItem,
		Description: "Moves an item to the end of another card.",
		InputSchema: schema([]string{"id", "card_id"}, map[string]any{
			"id":      sInt("Item id."),
			"card_id": sInt("Target card id."),
		})},
	{Name: "delete_item", Title: "Delete item", Annotations: annDelete, run: toolDeleteItem,
		Description: "Moves an item to the trash (kept 30 days).",
		InputSchema: schema([]string{"id"}, map[string]any{"id": sInt("Item id.")})},
	{Name: "list_trash", Title: "List trash", Annotations: annRead, run: toolListTrash,
		Description: "Lists deleted cards and items that can still be restored.",
		InputSchema: schema(nil, nil)},
	{Name: "restore_from_trash", Title: "Restore from trash", Annotations: annRestore, run: toolRestoreFromTrash,
		Description: "Restores a deleted card (with the items deleted together with it) or a deleted item.",
		InputSchema: schema([]string{"type", "id"}, map[string]any{
			"type": sEnum("What to restore.", "card", "item"),
			"id":   sInt("Card or item id from list_trash."),
		})},
}

var mcpToolByName = func() map[string]*mcpTool {
	m := make(map[string]*mcpTool, len(mcpTools))
	for _, t := range mcpTools {
		m[t.Name] = t
	}
	return m
}()
