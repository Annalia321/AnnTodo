package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"anntodo/pkg/core"
	"anntodo/pkg/models"
	"anntodo/pkg/version"
)

// MCP (Model Context Protocol) server, Streamable HTTP transport.
//
// The endpoint is stateless: every POST /mcp carries one JSON-RPC message (or
// a batch) and gets a plain application/json reply. No SSE stream is opened,
// which the specification allows. Clients authenticate with the personal
// token generated in Settings: "Authorization: Bearer sk-...".

var mcpProtocolVersions = []string{"2025-06-18", "2025-03-26", "2024-11-05"}

const mcpMaxBody = 1 << 20

const mcpInstructions = `AnnTodo is a personal to-do app. Items (to-dos) belong to cards (lists).
Call get_context first: it returns the current date, time and timezone of the user and the list of cards with their ids.
All dates are YYYY-MM-DD and times are HH:MM (24-hour) in the user's timezone. An item with a due_date but no due_time is an all-day item.
The flags important and urgent place an item in the Eisenhower matrix: important+urgent = do first, important only = schedule, urgent only = delegate, neither = eliminate.
Deleting a card or an item moves it to the trash, where it stays for 30 days; restore_from_trash brings it back.
Ask the user before deleting cards, because a card is deleted together with all of its items.`

const (
	rpcParseError     = -32700
	rpcInvalidRequest = -32600
	rpcMethodNotFound = -32601
	rpcInvalidParams  = -32602
	rpcInternalError  = -32603
	rpcUnauthorized   = -32001
)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	Result  json.RawMessage `json:"result"`
	Error   json.RawMessage `json:"error"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

var rpcNullID = json.RawMessage("null")

func rpcFail(id json.RawMessage, code int, msg string) *rpcResponse {
	if len(id) == 0 {
		id = rpcNullID
	}
	return &rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: msg}}
}

// mcpSession is the per-request context of an authenticated MCP call.
type mcpSession struct {
	app  *core.App
	user *models.User
	loc  *time.Location
}

// mcpCORS lets browser-based AI clients call /mcp. Authentication uses a
// bearer token (never cookies), so a wildcard origin is safe here.
func mcpCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "POST, GET, DELETE, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept, Mcp-Session-Id, Mcp-Protocol-Version, Last-Event-ID")
		h.Set("Access-Control-Expose-Headers", "Mcp-Session-Id, WWW-Authenticate")
		h.Set("Access-Control-Max-Age", "86400")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// mcpNotAllowed answers GET (SSE stream) and DELETE (session end): this
// stateless server offers neither.
func (s *Server) mcpNotAllowed(c *gin.Context) {
	c.Header("Allow", "POST, OPTIONS")
	c.JSON(http.StatusMethodNotAllowed, rpcFail(nil, rpcInvalidRequest, "method not allowed: send JSON-RPC messages with POST"))
}

func (s *Server) mcpAuth(c *gin.Context) (*models.User, *models.MCPToken, bool) {
	h := strings.TrimSpace(c.GetHeader("Authorization"))
	token := ""
	if len(h) > 7 && strings.EqualFold(h[:7], "bearer ") {
		token = strings.TrimSpace(h[7:])
	}
	if token != "" {
		u, t, err := s.App.ValidateMCPToken(token)
		if err == nil {
			return u, t, true
		}
		if !errors.Is(err, core.ErrUnauthorized) {
			c.JSON(http.StatusInternalServerError, rpcFail(nil, rpcInternalError, "internal error"))
			return nil, nil, false
		}
	}
	c.Header("WWW-Authenticate", `Bearer realm="AnnTodo MCP"`)
	c.JSON(http.StatusUnauthorized, rpcFail(nil, rpcUnauthorized,
		"unauthorized: missing or invalid token. Generate a token in AnnTodo Settings > MCP and send it as \"Authorization: Bearer sk-...\""))
	return nil, nil, false
}

func (s *Server) mcpPost(c *gin.Context) {
	u, tok, good := s.mcpAuth(c)
	if !good {
		return
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, mcpMaxBody+1))
	if err != nil {
		c.JSON(http.StatusBadRequest, rpcFail(nil, rpcParseError, "cannot read request body"))
		return
	}
	if len(body) > mcpMaxBody {
		c.JSON(http.StatusRequestEntityTooLarge, rpcFail(nil, rpcInvalidRequest, "request body too large"))
		return
	}
	body = bytes.TrimSpace(body)
	if !json.Valid(body) {
		c.JSON(http.StatusBadRequest, rpcFail(nil, rpcParseError, "parse error: body must be JSON"))
		return
	}
	sess := &mcpSession{app: s.App, user: u, loc: s.App.UserLocation(u.ID, tok.Timezone)}

	if body[0] == '[' {
		var msgs []json.RawMessage
		if err := json.Unmarshal(body, &msgs); err != nil || len(msgs) == 0 {
			c.JSON(http.StatusBadRequest, rpcFail(nil, rpcInvalidRequest, "invalid request: empty batch"))
			return
		}
		out := []*rpcResponse{}
		for _, raw := range msgs {
			if r := sess.handle(raw); r != nil {
				out = append(out, r)
			}
		}
		if len(out) == 0 {
			c.Status(http.StatusAccepted)
			return
		}
		c.JSON(http.StatusOK, out)
		return
	}
	r := sess.handle(body)
	if r == nil {
		c.Status(http.StatusAccepted) // notification or client response
		return
	}
	c.JSON(http.StatusOK, r)
}

// handle processes one JSON-RPC message. It returns nil for notifications
// and for responses sent by the client, which need no reply.
func (m *mcpSession) handle(raw json.RawMessage) *rpcResponse {
	var req rpcRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return rpcFail(nil, rpcInvalidRequest, "invalid request: expected a JSON-RPC object")
	}
	if req.Method == "" && (len(req.Result) > 0 || len(req.Error) > 0) {
		return nil
	}
	if req.JSONRPC != "2.0" || req.Method == "" {
		return rpcFail(req.ID, rpcInvalidRequest, "invalid request: jsonrpc must be \"2.0\" and method is required")
	}
	if len(req.ID) == 0 {
		return nil
	}
	result, rerr := m.dispatch(req.Method, req.Params)
	if rerr != nil {
		return rpcFail(req.ID, rerr.Code, rerr.Message)
	}
	return &rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result}
}

func (m *mcpSession) dispatch(method string, params json.RawMessage) (any, *rpcError) {
	switch method {
	case "initialize":
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(params, &p)
		v := mcpProtocolVersions[0]
		for _, x := range mcpProtocolVersions {
			if x == p.ProtocolVersion {
				v = x
			}
		}
		return gin.H{
			"protocolVersion": v,
			"capabilities":    gin.H{"tools": gin.H{"listChanged": false}},
			"serverInfo":      gin.H{"name": "anntodo", "title": "AnnTodo", "version": version.Version},
			"instructions":    mcpInstructions,
		}, nil
	case "ping":
		return struct{}{}, nil
	case "tools/list":
		return gin.H{"tools": mcpTools}, nil
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(params, &p); err != nil || p.Name == "" {
			return nil, &rpcError{Code: rpcInvalidParams, Message: "invalid params: name is required"}
		}
		t := mcpToolByName[p.Name]
		if t == nil {
			return nil, &rpcError{Code: rpcInvalidParams, Message: "unknown tool: " + p.Name}
		}
		args := bytes.TrimSpace(p.Arguments)
		if len(args) == 0 || string(args) == "null" {
			args = []byte("{}")
		}
		if args[0] != '{' {
			return mcpToolError(errors.New("arguments must be a JSON object")), nil
		}
		out, err := t.run(m, args)
		if err != nil {
			return mcpToolError(err), nil
		}
		text, err := json.Marshal(out)
		if err != nil {
			return nil, &rpcError{Code: rpcInternalError, Message: "internal error"}
		}
		return gin.H{"content": []gin.H{{"type": "text", "text": string(text)}}, "isError": false}, nil
	}
	return nil, &rpcError{Code: rpcMethodNotFound, Message: "method not found: " + method}
}

// mcpToolError reports a failed tool call so the model can read the reason
// and correct itself.
func mcpToolError(err error) gin.H {
	msg := err.Error()
	if e, ok := core.AsError(err); ok {
		switch {
		case e == core.ErrNotFound:
			msg = "not found: no card or item with this id (use list_cards / list_items to look up ids)"
		case e.Status >= 500:
			slog.Error("mcp tool failed", "err", err)
			msg = "internal error"
		default:
			msg = e.Message
		}
	} else if _, isArg := err.(*mcpArgError); !isArg {
		slog.Error("mcp tool failed", "err", err)
		msg = "internal error"
	}
	return gin.H{"content": []gin.H{{"type": "text", "text": "error: " + msg}}, "isError": true}
}
