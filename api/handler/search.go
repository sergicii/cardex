package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/operaodev/cardex/internal/search"
)

type SearchHandler struct {
	svc *search.Service
}

func NewSearchHandler(s *search.Service) *SearchHandler {
	return &SearchHandler{svc: s}
}

// SearchByIDInProvider maneja GET /cards/search/:provider/:id
func (h *SearchHandler) SearchByIDInProvider(c *gin.Context) {
	provider := c.Param("provider")
	id := c.Param("id")

	result, err := h.svc.SearchByID(provider, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// SearchByNamesInProvider maneja GET /cards/search/:provider?name=Kuriboh
func (h *SearchHandler) SearchByNamesInProvider(c *gin.Context) {
	provider := c.Param("provider")
	name := c.Query("name")

	results, err := h.svc.SearchByNames(provider, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, results)
}
