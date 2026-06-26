package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/operaodev/cardex/internal/jwt"
	"github.com/operaodev/cardex/internal/users"
)

type AuthResponse struct {
	User         *users.User `json:"user"`
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
}

type UsersHandler struct {
	service             users.Service
	jwtSecret           string
	accessTokenDuration time.Duration
	refreshTokenDuration time.Duration
}

func NewUsersHandler(s users.Service, jwtSecret string, accessDuration, refreshDuration time.Duration) *UsersHandler {
	return &UsersHandler{
		service:              s,
		jwtSecret:            jwtSecret,
		accessTokenDuration:  accessDuration,
		refreshTokenDuration: refreshDuration,
	}
}

func (h *UsersHandler) RegisterGuest(c *gin.Context) {
	user, err := h.service.RegisterGuest()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	accessToken, refreshToken, err := h.generateAndStoreTokens(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al generar tokens"})
		return
	}

	h.setTokenCookies(c, accessToken, refreshToken)
	c.JSON(http.StatusCreated, AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

func (h *UsersHandler) SendCode(c *gin.Context) {
	var input users.SendVerificationCodeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidJSONBody})
		return
	}

	if err := h.service.SendVerificationCode(input); err != nil {
		if errors.Is(err, users.ErrEmailAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "código de verificación enviado al email"})
}

func (h *UsersHandler) Register(c *gin.Context) {
	var input users.RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidJSONBody})
		return
	}

	user, err := h.service.Register(input)
	if err != nil {
		if errors.Is(err, users.ErrInvalidVerificationCode) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, users.ErrVerificationCodeExpired) {
			c.JSON(http.StatusGone, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, users.ErrEmailAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accessToken, refreshToken, err := h.generateAndStoreTokens(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al generar tokens"})
		return
	}

	h.setTokenCookies(c, accessToken, refreshToken)
	c.JSON(http.StatusCreated, AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

func (h *UsersHandler) Login(c *gin.Context) {
	var input users.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidJSONBody})
		return
	}

	user, err := h.service.Login(input)
	if err != nil {
		if errors.Is(err, users.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
			return
		}
		if errors.Is(err, users.ErrEmailNotVerified) {
			c.JSON(http.StatusForbidden, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error interno del servidor"})
		return
	}

	accessToken, refreshToken, err := h.generateAndStoreTokens(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al generar tokens"})
		return
	}

	h.setTokenCookies(c, accessToken, refreshToken)
	c.JSON(http.StatusOK, AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

func (h *UsersHandler) RefreshToken(c *gin.Context) {
	refreshToken := h.extractRefreshToken(c)
	if refreshToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token requerido"})
		return
	}

	claims, err := jwt.ValidateRefreshToken(refreshToken, h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token inválido o expirado"})
		return
	}

	refreshHash := users.HashToken(refreshToken)
	if _, err := h.service.RefreshSession(claims.UserID, refreshHash); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token inválido o revocado"})
		return
	}

	user, err := h.service.GetByID(claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al obtener usuario"})
		return
	}

	newAccessToken, newRefreshToken, err := h.generateAndStoreTokens(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al generar tokens"})
		return
	}

	h.setTokenCookies(c, newAccessToken, newRefreshToken)
	c.JSON(http.StatusOK, AuthResponse{
		User:         user,
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	})
}

func (h *UsersHandler) UpgradeGuest(c *gin.Context) {
	userID, _ := c.Get("userID")
	var input users.UpgradeGuestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidJSONBody})
		return
	}
	input.UserID = userID.(string)

	user, err := h.service.UpgradeGuest(input)
	if err != nil {
		if errors.Is(err, users.ErrNotAGuest) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, users.ErrInvalidVerificationCode) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, users.ErrVerificationCodeExpired) {
			c.JSON(http.StatusGone, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, users.ErrEmailAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accessToken, refreshToken, err := h.generateAndStoreTokens(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al generar tokens"})
		return
	}

	h.setTokenCookies(c, accessToken, refreshToken)
	c.JSON(http.StatusOK, AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

func (h *UsersHandler) GetMe(c *gin.Context) {
	userID, _ := c.Get("userID")
	user, err := h.service.GetByID(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al obtener el perfil"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *UsersHandler) generateAndStoreTokens(user *users.User) (string, string, error) {
	accessToken, err := jwt.GenerateToken(user.ID, user.Email, user.Name, h.jwtSecret, h.accessTokenDuration)
	if err != nil {
		return "", "", err
	}

	refreshToken, err := jwt.GenerateRefreshToken(user.ID, h.jwtSecret, h.refreshTokenDuration)
	if err != nil {
		return "", "", err
	}

	refreshHash := users.HashToken(refreshToken)
	if err := h.service.StoreRefreshToken(user.ID, refreshHash); err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (h *UsersHandler) setTokenCookies(c *gin.Context, accessToken, refreshToken string) {
	maxAgeAccess := int(h.accessTokenDuration.Seconds())
	maxAgeRefresh := int(h.refreshTokenDuration.Seconds())

	c.SetCookie("access_token", accessToken, maxAgeAccess, "/", "", false, true)
	c.SetCookie("refresh_token", refreshToken, maxAgeRefresh, "/", "", false, true)
}

func (h *UsersHandler) extractRefreshToken(c *gin.Context) string {
	cookie, err := c.Cookie("refresh_token")
	if err == nil && cookie != "" {
		return cookie
	}

	bodyBytes, _ := c.GetRawData()
	if len(bodyBytes) > 0 {
		var body struct {
			RefreshToken string `json:"refresh_token"`
		}
		json.Unmarshal(bodyBytes, &body)
		return body.RefreshToken
	}

	return ""
}


