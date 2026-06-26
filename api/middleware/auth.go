package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/operaodev/cardex/internal/jwt"
	"github.com/operaodev/cardex/internal/stock"
)

const (
	UserIDKey   = "userID"
	EmailKey    = "email"
	NameKey     = "name"
	IsGuestKey  = "isGuest"
)

func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token requerido (header Authorization o cookie access_token)"})
			return
		}

		claims, err := jwt.ValidateToken(token, secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token inválido o expirado"})
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Set(EmailKey, claims.Email)
		c.Set(NameKey, claims.Name)
		c.Next()
	}
}

func OptionalAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			c.Next()
			return
		}

		claims, err := jwt.ValidateToken(token, secret)
		if err != nil {
			c.Next()
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Set(EmailKey, claims.Email)
		c.Set(NameKey, claims.Name)
		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return parts[1]
		}
	}

	cookie, err := c.Cookie("access_token")
	if err == nil && cookie != "" {
		return cookie
	}

	return ""
}

func RequireStockOwnership(stockRepo stock.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get(UserIDKey)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "usuario no autenticado"})
			return
		}

		idStr := c.Param("id")
		if idStr == "" {
			idStr = c.Param("stock_id")
		}
		if idStr == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "id de stock requerido"})
			return
		}

		stockID, err := parseUint64(idStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "id de stock inválido"})
			return
		}

		s, err := stockRepo.FindByID(stockID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "stock no encontrado"})
			return
		}

		if s.UserID != userID.(string) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "no tienes permiso sobre este stock"})
			return
		}

		c.Next()
	}
}

func parseUint64(s string) (uint64, error) {
	var n uint64
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, errNotANumber
		}
		n = n*10 + uint64(ch-'0')
	}
	return n, nil
}

var errNotANumber = &parseError{"no es un número"}

type parseError struct {
	msg string
}

func (e *parseError) Error() string { return e.msg }
