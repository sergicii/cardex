package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/operaodev/cardex/internal/cards"
)

// CardsHandler expone las funciones del servicio a través de HTTP.
type CardsHandler struct {
	service cards.Service
}

// NewCardsHandler crea una nueva instancia del Handler inyectando el servicio.
func NewCardsHandler(s cards.Service) *CardsHandler {
	return &CardsHandler{
		service: s,
	}
}

// GetByID maneja las peticiones para obtener una carta por ID.
func (h *CardsHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Debe proporcionar el ID de la carta"})
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	
	card, err := h.service.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, card)
}

// GetByName maneja las peticiones para buscar cartas por nombre.
func (h *CardsHandler) GetByName(c *gin.Context) {
	name := c.Query("name")

	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Debe proporcionar el parámetro 'name' en la URL"})
		return
	}

	results, err := h.service.GetByName(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, results)
}
