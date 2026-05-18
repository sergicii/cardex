package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/operaodev/cardex/internal/users"
)

// UsersHandler expone las funciones del servicio de usuarios a través de HTTP.
type UsersHandler struct {
	service users.Service
}

// NewUsersHandler crea una nueva instancia del Handler inyectando el servicio.
func NewUsersHandler(s users.Service) *UsersHandler {
	return &UsersHandler{service: s}
}

// Register maneja las peticiones de registro de nuevos usuarios.
// Body JSON: { "name": "...", "email": "...", "password": "..." }
func (h *UsersHandler) Register(c *gin.Context) {
	var input users.RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cuerpo de la petición inválido"})
		return
	}

	user, err := h.service.Register(input)
	if err != nil {
		if errors.Is(err, users.ErrEmailAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Devolver el usuario creado (HashedPassword excluida por el tag json:"-")
	c.JSON(http.StatusCreated, user)
}

// Login maneja las peticiones de autenticación de usuarios.
// Body JSON: { "email": "...", "password": "..." }
func (h *UsersHandler) Login(c *gin.Context) {
	var input users.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cuerpo de la petición inválido"})
		return
	}

	user, err := h.service.Login(input)
	if err != nil {
		if errors.Is(err, users.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error interno del servidor"})
		return
	}

	// Devolver el usuario autenticado (HashedPassword excluida por el tag json:"-")
	c.JSON(http.StatusOK, user)
}
