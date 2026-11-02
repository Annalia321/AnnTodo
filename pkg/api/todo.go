package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"anntodo/pkg/core"
)

func (s *Server) listCards(c *gin.Context) {
	cards, err := s.App.ListCards(uid(c))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, cards)
}

func (s *Server) createCard(c *gin.Context) {
	var in core.CardInput
	if !bind(c, &in) {
		return
	}
	card, err := s.App.CreateCard(uid(c), in)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, card)
}

func (s *Server) updateCard(c *gin.Context) {
	id, good := paramID(c, "id")
	if !good {
		return
	}
	var in core.CardPatch
	if !bind(c, &in) {
		return
	}
	card, err := s.App.UpdateCard(uid(c), id, in)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, card)
}

func (s *Server) deleteCard(c *gin.Context) {
	id, good := paramID(c, "id")
	if !good {
		return
	}
	if err := s.App.DeleteCard(uid(c), id); err != nil {
		fail(c, err)
		return
	}
	noContent(c)
}

func (s *Server) reorderCards(c *gin.Context) {
	var in struct {
		IDs []uint64 `json:"ids"`
	}
	if !bind(c, &in) {
		return
	}
	if err := s.App.ReorderCards(uid(c), in.IDs); err != nil {
		fail(c, err)
		return
	}
	noContent(c)
}

func (s *Server) listItems(c *gin.Context) {
	var f core.ItemFilter
	if v := c.Query("card_id"); v != "" {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			fail(c, core.BadRequest("invalid card_id"))
			return
		}
		f.CardID = id
	}
	f.From, f.To = c.Query("from"), c.Query("to")
	items, err := s.App.ListItems(uid(c), f)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, items)
}

func (s *Server) createItem(c *gin.Context) {
	var in core.ItemInput
	if !bind(c, &in) {
		return
	}
	it, err := s.App.CreateItem(uid(c), in)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, it)
}

func (s *Server) updateItem(c *gin.Context) {
	id, good := paramID(c, "id")
	if !good {
		return
	}
	var in core.ItemPatch
	if !bind(c, &in) {
		return
	}
	it, err := s.App.UpdateItem(uid(c), id, in)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, it)
}

func (s *Server) completeItem(c *gin.Context) {
	id, good := paramID(c, "id")
	if !good {
		return
	}
	var in struct {
		Completed bool `json:"completed"`
	}
	if !bind(c, &in) {
		return
	}
	it, err := s.App.SetCompleted(uid(c), id, in.Completed)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, it)
}

func (s *Server) moveItem(c *gin.Context) {
	id, good := paramID(c, "id")
	if !good {
		return
	}
	var in struct {
		CardID uint64 `json:"card_id"`
	}
	if !bind(c, &in) {
		return
	}
	it, err := s.App.MoveItem(uid(c), id, in.CardID)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, it)
}

func (s *Server) deleteItem(c *gin.Context) {
	id, good := paramID(c, "id")
	if !good {
		return
	}
	if err := s.App.DeleteItem(uid(c), id); err != nil {
		fail(c, err)
		return
	}
	noContent(c)
}

func (s *Server) reorderItems(c *gin.Context) {
	var in struct {
		Lists []core.OrderList `json:"lists"`
	}
	if !bind(c, &in) {
		return
	}
	if err := s.App.ReorderItems(uid(c), in.Lists); err != nil {
		fail(c, err)
		return
	}
	noContent(c)
}

func (s *Server) reorderQuadrants(c *gin.Context) {
	var in struct {
		Cells []core.QuadrantCell `json:"cells"`
	}
	if !bind(c, &in) {
		return
	}
	if err := s.App.ReorderQuadrants(uid(c), in.Cells); err != nil {
		fail(c, err)
		return
	}
	noContent(c)
}

func (s *Server) getSettings(c *gin.Context) {
	st, err := s.App.GetUserSettings(uid(c))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, st)
}

func (s *Server) patchSettings(c *gin.Context) {
	var in map[string]any
	if !bind(c, &in) {
		return
	}
	st, err := s.App.PatchUserSettings(uid(c), in)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, st)
}
