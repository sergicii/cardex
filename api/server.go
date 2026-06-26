package api

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/operaodev/cardex/api/handler"
	"github.com/operaodev/cardex/api/middleware"
	"github.com/operaodev/cardex/internal/stock"
)

type Server struct {
	router             *gin.Engine
	providersHandler   *handler.ProviderHandler
	usersHandler       *handler.UsersHandler
	syncHandler        *handler.SyncHandler
	productsHandler    *handler.ProductsHandler
	stockHandler       *handler.StockHandler
	marketplaceHandler *handler.MarketplaceHandler
	wishlistHandler    *handler.WishlistHandler
	stockRepo          stock.Repository
	jwtSecret          string
}

func NewServer(
	providersH *handler.ProviderHandler,
	usersH *handler.UsersHandler,
	syncH *handler.SyncHandler,
	productsH *handler.ProductsHandler,
	stockH *handler.StockHandler,
	marketplaceH *handler.MarketplaceHandler,
	wishlistH *handler.WishlistHandler,
	stockRepo stock.Repository,
	jwtSecret string,
) *Server {
	router := gin.Default()

	// Middleware de CORS (sin wildcard para soportar cookies/credenciales)
	router.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	s := &Server{
		router:             router,
		providersHandler:   providersH,
		usersHandler:       usersH,
		syncHandler:        syncH,
		productsHandler:    productsH,
		stockHandler:       stockH,
		marketplaceHandler: marketplaceH,
		wishlistHandler:    wishlistH,
		stockRepo:          stockRepo,
		jwtSecret:          jwtSecret,
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	providersGroup := s.router.Group("/providers")
	{
		// GET /providers/:provider/cards
		providersGroup.GET("/:provider/cards", s.providersHandler.FetchCards)
		// GET /providers/:provider/cards/:name
		providersGroup.GET("/:provider/cards/:name", s.providersHandler.FetchCardsByName)
	}

	// Rutas de sincronización (administración)
	syncGroup := s.router.Group("/sync")
	{
		// GET /sync/status
		syncGroup.GET("/status", s.syncHandler.SyncStatus)
		// POST /sync/ygo
		syncGroup.POST("/:tcg", s.syncHandler.TriggerSync)
		// POST /sync/ygo/by-name
		syncGroup.POST("/:tcg/by-name", s.syncHandler.TriggerSyncByName)
	}

	// Rutas de usuarios (auth)
	usersGroup := s.router.Group("/users")
	{
		// POST /users/guest — registro como invitado
		usersGroup.POST("/guest", s.usersHandler.RegisterGuest)
		// POST /users/send-code — enviar código de verificación al email
		usersGroup.POST("/send-code", s.usersHandler.SendCode)
		// POST /users/register — completar registro con código de verificación
		usersGroup.POST("/register", s.usersHandler.Register)
		// POST /users/login
		usersGroup.POST("/login", s.usersHandler.Login)
		// POST /users/refresh — renovar access token usando refresh token
		usersGroup.POST("/refresh", s.usersHandler.RefreshToken)
		// POST /users/upgrade — invitado hace upgrade a cuenta completa (requiere auth)
		usersGroup.POST("/upgrade", middleware.AuthMiddleware(s.jwtSecret), s.usersHandler.UpgradeGuest)
		// GET /users/me (requiere auth)
		usersGroup.GET("/me", middleware.AuthMiddleware(s.jwtSecret), s.usersHandler.GetMe)
	}

	// Middleware de autenticación reutilizado en varias rutas
	auth := middleware.AuthMiddleware(s.jwtSecret)
	optionalAuth := middleware.OptionalAuthMiddleware(s.jwtSecret)

	// Rutas de products
	productsGroup := s.router.Group("/products")
	{
		// Las rutas fijas deben ir antes del comodín :id
		// POST /products/suggestions (auth opcional: incluye stock/wishlist si el usuario está logueado)
		productsGroup.POST("/suggestions", optionalAuth, s.productsHandler.FindSuggestions)
		// POST /products/suggestions/simple (sin auth; rápido sin JOINs de stock/wishlist)
		productsGroup.POST("/suggestions/simple", s.productsHandler.FindSuggestionsSimple)
		// GET /products/random/:count
		productsGroup.GET("/random/:count", s.productsHandler.GetRandomNames)
		// POST /products/related
		productsGroup.POST("/related", s.productsHandler.GetRelatedCards)
		// POST /products/set
		productsGroup.POST("/set", s.productsHandler.GetCardsBySet)
		// GET /products/:id
		productsGroup.GET("/:id", s.productsHandler.GetByID)
	}

	stockGroup := s.router.Group("/stock")
	ownership := middleware.RequireStockOwnership(s.stockRepo)
	{
		// GET /stock/me — stock del usuario autenticado
		stockGroup.GET("/me", auth, s.stockHandler.GetMyStock)
		// GET /stock/:user_id
		stockGroup.GET("/:user_id", auth, s.stockHandler.GetByUserID)
		// GET /stock/id/:id — requiere ownership
		stockGroup.GET("/id/:id", auth, ownership, s.stockHandler.GetByID)
		// GET /stock/logs/:stock_id — requiere ownership
		stockGroup.GET("/logs/:stock_id", auth, ownership, s.stockHandler.GetLogs)
		// POST /stock
		stockGroup.POST("", auth, s.stockHandler.Create)
		// POST /stock/openbox
		stockGroup.POST("/openbox", auth, s.stockHandler.OpenBox)
		// POST /stock/restock
		stockGroup.POST("/restock", auth, s.stockHandler.Restock)
		// POST /stock/return
		stockGroup.POST("/return", auth, s.stockHandler.Return)
		// POST /stock/sale
		stockGroup.POST("/sale", auth, s.stockHandler.Sale)
		// POST /stock/trade
		stockGroup.POST("/trade", auth, s.stockHandler.Trade)
		// POST /stock/gift
		stockGroup.POST("/gift", auth, s.stockHandler.Gift)
		// POST /stock/lost
		stockGroup.POST("/lost", auth, s.stockHandler.Lost)
		// POST /stock/damage
		stockGroup.POST("/damage", auth, s.stockHandler.Damage)
		// POST /stock/adjust
		stockGroup.POST("/adjust", auth, s.stockHandler.Adjust)
		// POST /stock/rollback
		stockGroup.POST("/rollback", auth, s.stockHandler.Rollback)
		// POST /stock/:id/price
		stockGroup.POST("/:id/price", auth, ownership, s.stockHandler.UpdatePrice)
		// POST /stock/:id/for-sale
		stockGroup.POST("/:id/for-sale", auth, ownership, s.stockHandler.SetForSale)
		// POST /stock/:id/for-trade
		stockGroup.POST("/:id/for-trade", auth, ownership, s.stockHandler.SetForTrade)
	}

	// Marketplace
	marketplaceGroup := s.router.Group("/marketplace")
	{
		// GET /marketplace/analysis/:id
		marketplaceGroup.GET("/analysis/:id", s.marketplaceHandler.GetPrices)
		// GET /marketplace/offers/:id
		marketplaceGroup.GET("/offers/:id", s.marketplaceHandler.GetOffers)
		// POST /marketplace/cards
		marketplaceGroup.POST("/cards", s.marketplaceHandler.FindCards)
	}

	// Wishlist (custom packs)
	wishlistGroup := s.router.Group("/wishlist")
	{
		// GET /wishlist
		wishlistGroup.GET("", auth, s.wishlistHandler.GetMyWishlist)
		// POST /wishlist
		wishlistGroup.POST("", auth, s.wishlistHandler.Upsert)
		// DELETE /wishlist/:product_id
		wishlistGroup.DELETE("/:product_id", auth, s.wishlistHandler.Delete)
	}
}

func (s *Server) Start(addr string) error {
	log.Printf("Iniciando servidor en %s", addr)
	return s.router.Run(addr)
}
