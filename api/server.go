package api

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/operaodev/cardex/api/handler"
)

type Server struct {
	router           *gin.Engine
	cardsHandler     *handler.CardsHandler
	searchHandler    *handler.SearchHandler
	syncHandler      *handler.SyncHandler
	usersHandler     *handler.UsersHandler
	inventoryHandler *handler.InventoryHandler
}

func NewServer(
	cardsH *handler.CardsHandler,
	searchH *handler.SearchHandler,
	syncH *handler.SyncHandler,
	usersH *handler.UsersHandler,
	inventoryH *handler.InventoryHandler,
) *Server {
	router := gin.Default()

	// Middleware de CORS
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
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
		router:           router,
		cardsHandler:     cardsH,
		searchHandler:    searchH,
		syncHandler:      syncH,
		usersHandler:     usersH,
		inventoryHandler: inventoryH,
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	cardsGroup := s.router.Group("/cards")
	{
		// GET /cards?tcg=ygo&lang=sp&page=1&limit=20
		cardsGroup.GET("", s.cardsHandler.GetCatalog)
		// GET /cards/1234
		cardsGroup.GET("/:id", s.cardsHandler.GetByID)
		// GET /cards/suggestions?tcg=ygo&lang=sp&name=Kuriboh
		cardsGroup.GET("/suggestions", s.cardsHandler.GetSuggestions)

		// Proveedores externos
		// GET /cards/search/:provider/:id
		cardsGroup.GET("/search/:provider/:id", s.searchHandler.SearchByIDInProvider)
		// GET /cards/search/:provider?name=Kuriboh
		cardsGroup.GET("/search/:provider", s.searchHandler.SearchByNamesInProvider)
		// GET /cards/search/:provider/all
		cardsGroup.GET("/search/:provider/all", s.searchHandler.SearchAllInProvider)
	}

	// Rutas de sincronización (administración)
	syncGroup := s.router.Group("/sync")
	{
		// GET /sync/status
		syncGroup.GET("/status", s.syncHandler.SyncStatus)
		// POST /sync/ygo
		syncGroup.POST("/:tcg", s.syncHandler.TriggerSync)
	}

	// Rutas de usuarios (auth)
	usersGroup := s.router.Group("/users")
	{
		// POST /users/register
		usersGroup.POST("/register", s.usersHandler.Register)
		// POST /users/login
		usersGroup.POST("/login", s.usersHandler.Login)
	}

	// Rutas de inventario
	invGroup := s.router.Group("/inventory")
	{
		// GET /inventory/:user_id
		invGroup.GET("/:user_id", s.inventoryHandler.GetInventory)
		// GET /inventory/logs/:inventory_id
		invGroup.GET("/logs/:inventory_id", s.inventoryHandler.GetLogs)
		// POST /inventory/restock
		invGroup.POST("/restock", s.inventoryHandler.Restock)
		// POST /inventory/sell
		invGroup.POST("/sell", s.inventoryHandler.Sell)
		// POST /inventory/loss
		invGroup.POST("/loss", s.inventoryHandler.RegisterLoss)
		// POST /inventory/return
		invGroup.POST("/return", s.inventoryHandler.RegisterReturn)
		// POST /inventory/price
		invGroup.POST("/price", s.inventoryHandler.ChangePrice)
	}
}

func (s *Server) Start(addr string) error {
	log.Printf("Iniciando servidor en %s", addr)
	return s.router.Run(addr)
}
