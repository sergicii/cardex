package api

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/operaodev/cardex/api/handler"
)

type Server struct {
	router        *gin.Engine
	cardsHandler  *handler.CardsHandler
	searchHandler *handler.SearchHandler
}

func NewServer(cardsH *handler.CardsHandler, searchH *handler.SearchHandler) *Server {
	s := &Server{
		router:        gin.Default(),
		cardsHandler:  cardsH,
		searchHandler: searchH,
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	cardsGroup := s.router.Group("/cards")
	{
		// /cards/search?name=Kuriboh
		cardsGroup.GET("/search", s.cardsHandler.GetByName)
		// /cards/1234
		cardsGroup.GET("/:id", s.cardsHandler.GetByID)
		
		// Proveedores externos
		// /cards/search/:provider/:id
		cardsGroup.GET("/search/:provider/:id", s.searchHandler.SearchByIDInProvider)
		// /cards/search/:provider?name=Kuriboh
		cardsGroup.GET("/search/:provider", s.searchHandler.SearchByNamesInProvider)
	}
}

func (s *Server) Start(addr string) error {
	log.Printf("Iniciando servidor en %s", addr)
	return s.router.Run(addr)
}
