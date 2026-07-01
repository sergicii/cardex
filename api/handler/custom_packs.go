package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	custompacks "github.com/operaodev/cardex/internal/custom_packs"
)

type WishlistHandler struct {
	service custompacks.Service
}

func NewWishlistHandler(s custompacks.Service) *WishlistHandler {
	return &WishlistHandler{service: s}
}

// UpsertRequest es el cuerpo de la petición para agregar/quitar items.
type UpsertRequest struct {
	ProductID uint64 `json:"product_id" binding:"required"`
	Delta     int    `json:"delta"     binding:"required"`
}

// GetMyWishlist obtiene la wishlist del usuario autenticado.
// GET /wishlist
func (h *WishlistHandler) GetMyWishlist(c *gin.Context) {
	userID, _ := c.Get("userID")
	items, err := h.service.GetByUserID(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

// Upsert agrega o modifica un item de la wishlist.
// POST /wishlist
func (h *WishlistHandler) Upsert(c *gin.Context) {
	var req UpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": ErrInvalidJSONBody})
		return
	}

	userID, _ := c.Get("userID")
	item, err := h.service.Upsert(userID.(string), req.ProductID, req.Delta)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	if item == nil {
		c.JSON(http.StatusOK, gin.H{"message": "item eliminado por cantidad <= 0"})
		return
	}
	c.JSON(http.StatusOK, item)
}

// Delete elimina un item de la wishlist por su ID.
// DELETE /wishlist/:wishlist_id
func (h *WishlistHandler) Delete(c *gin.Context) {
	wishlistIDStr := c.Param("wishlist_id")
	wishlistID, err := strconv.ParseUint(wishlistIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "wishlist_id inválido"})
		return
	}

	userID, _ := c.Get("userID")
	if err := h.service.Delete(userID.(string), wishlistID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "item eliminado"})
}

// CheckInWishlist verifica si una carta está en la wishlist del usuario autenticado.
// GET /wishlist/check/:product_id
func (h *WishlistHandler) CheckInWishlist(c *gin.Context) {
	productIDStr := c.Param("product_id")
	productID, err := strconv.ParseUint(productIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "product_id inválido"})
		return
	}

	userID, _ := c.Get("userID")
	wishlistID, found, err := h.service.IsInWishlist(userID.(string), productID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"in_wishlist": found,
		"wishlist_id": wishlistID,
	})
}

// GetMyBundles obtiene los bundles del usuario autenticado.
// GET /bundles
func (h *WishlistHandler) GetMyBundles(c *gin.Context) {
	userID, _ := c.Get("userID")
	bundles, err := h.service.GetBundlesByUserID(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bundles)
}

// CreateBundle crea un nuevo bundle para el usuario autenticado.
// POST /bundles
func (h *WishlistHandler) CreateBundle(c *gin.Context) {
	var req struct {
		Items []custompacks.BundleItem `json:"items" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": ErrInvalidJSONBody})
		return
	}

	userID, _ := c.Get("userID")
	bundle, err := h.service.CreateBundle(userID.(string), req.Items)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, bundle)
}

// UpdateBundle edita un bundle existente para el usuario autenticado.
// PUT /bundles/:bundle_id
func (h *WishlistHandler) UpdateBundle(c *gin.Context) {
	bundleIDStr := c.Param("bundle_id")
	bundleID, err := strconv.ParseUint(bundleIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "bundle_id inválido"})
		return
	}

	var req struct {
		Items []custompacks.BundleItem `json:"items" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": ErrInvalidJSONBody})
		return
	}

	userID, _ := c.Get("userID")
	bundle, err := h.service.UpdateBundle(userID.(string), bundleID, req.Items)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bundle)
}

// DeleteBundle elimina un bundle para el usuario autenticado.
// DELETE /bundles/:bundle_id
func (h *WishlistHandler) DeleteBundle(c *gin.Context) {
	bundleIDStr := c.Param("bundle_id")
	bundleID, err := strconv.ParseUint(bundleIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "bundle_id inválido"})
		return
	}

	userID, _ := c.Get("userID")
	if err := h.service.DeleteBundle(userID.(string), bundleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "bundle eliminado"})
}
