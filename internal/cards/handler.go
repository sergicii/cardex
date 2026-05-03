package cards

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler expone las funciones del servicio a través de HTTP (usando Gin).
type Handler struct {
	service Service
}

// NewHandler crea una nueva instancia del Handler inyectando el servicio.
func NewHandler(s Service) *Handler {
	return &Handler{
		service: s,
	}
}

// GetByIDHandler maneja las peticiones para obtener una carta por ID.
// Espera una ruta configurada en Gin como: r.GET("/cards/:id", handler.GetByIDHandler)
func (h *Handler) GetByIDHandler(c *gin.Context) {
	// Gin extrae los parámetros de la ruta de forma nativa
	id := c.Param("id")

	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Debe proporcionar el ID de la carta"})
		return
	}

	// Llamamos a nuestra capa de negocio (Service)
	card, err := h.service.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Gin se encarga automáticamente de serializar a JSON y asignar el Content-Type
	c.JSON(http.StatusOK, card)
}

// GetByNameHandler maneja las peticiones para buscar cartas por nombre.
// Espera una ruta como: r.GET("/cards/search", handler.GetByNameHandler)
// y un query param: ?name=Mago
func (h *Handler) GetByNameHandler(c *gin.Context) {
	// Gin extrae los query parameters fácilmente
	name := c.Query("name")

	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Debe proporcionar el parámetro 'name' en la URL"})
		return
	}

	// Llamamos a nuestra capa de negocio
	cards, err := h.service.GetByName(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, cards)
}
