package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/operaodev/cardex/internal/inventory"
)

// InventoryHandler expone las operaciones del inventario a través de HTTP.
type InventoryHandler struct {
	service inventory.Service
}

// NewInventoryHandler crea una nueva instancia del handler de inventario.
func NewInventoryHandler(s inventory.Service) *InventoryHandler {
	return &InventoryHandler{service: s}
}

// GetInventory devuelve el inventario completo de un usuario.
// GET /inventory/:user_id
func (h *InventoryHandler) GetInventory(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Debe proporcionar el user_id"})
		return
	}

	items, err := h.service.GetInventory(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, items)
}

// GetLogs devuelve el historial de movimientos de un registro de inventario.
// GET /inventory/logs/:inventory_id
func (h *InventoryHandler) GetLogs(c *gin.Context) {
	idStr := c.Param("inventory_id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "inventory_id inválido"})
		return
	}

	logs, err := h.service.GetLogs(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, logs)
}

// Restock añade cartas al inventario (entrada de stock o creación inicial).
// POST /inventory/restock
// Body: { "user_id", "card_id", "count", "price", "note" }
func (h *InventoryHandler) Restock(c *gin.Context) {
	var input inventory.AddInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cuerpo de la petición inválido"})
		return
	}

	inv, err := h.service.Restock(input)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, inv)
}

// Sell registra la venta de cartas, reduciendo el stock.
// POST /inventory/sell
// Body: { "user_id", "card_id", "count", "note" }
func (h *InventoryHandler) Sell(c *gin.Context) {
	var input inventory.SellInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cuerpo de la petición inválido"})
		return
	}

	inv, err := h.service.Sell(input)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, inv)
}

// RegisterLoss registra pérdida o daño de cartas.
// POST /inventory/loss
// Body: { "user_id", "card_id", "count", "log_type": "perdida"|"daño", "note" }
func (h *InventoryHandler) RegisterLoss(c *gin.Context) {
	var input inventory.LossInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cuerpo de la petición inválido"})
		return
	}

	inv, err := h.service.RegisterLoss(input)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, inv)
}

// RegisterReturn registra una devolución de cartas.
// POST /inventory/return
// Body: { "user_id", "card_id", "count", "note" }
func (h *InventoryHandler) RegisterReturn(c *gin.Context) {
	var input inventory.ReturnInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cuerpo de la petición inválido"})
		return
	}

	inv, err := h.service.RegisterReturn(input)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, inv)
}

// ChangePrice actualiza el precio de una carta en el inventario sin mover stock.
// POST /inventory/price
// Body: { "user_id", "card_id", "new_price", "note" }
func (h *InventoryHandler) ChangePrice(c *gin.Context) {
	var input inventory.PriceChangeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cuerpo de la petición inválido"})
		return
	}

	inv, err := h.service.ChangePrice(input)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, inv)
}

// handleServiceError mapea los errores del servicio a códigos HTTP apropiados.
func (h *InventoryHandler) handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, inventory.ErrInventoryNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, inventory.ErrInsufficientStock):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, inventory.ErrInvalidDelta),
		errors.Is(err, inventory.ErrInvalidPrice):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
