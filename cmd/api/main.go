package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/operaodev/cardex/api"
	"github.com/operaodev/cardex/api/handler"
	"github.com/operaodev/cardex/internal/cards"
	"github.com/operaodev/cardex/internal/database"
	"github.com/operaodev/cardex/internal/inventory"
	"github.com/operaodev/cardex/internal/search"
	searchproviders "github.com/operaodev/cardex/internal/search/providers"
	syncsvc "github.com/operaodev/cardex/internal/sync"
	"github.com/operaodev/cardex/internal/users"
)

func main() {
	// 0. Cargar variables de entorno
	if err := godotenv.Load(); err != nil {
		log.Println("Advertencia: no se cargó .env, usando variables del sistema")
	}

	// 1. Inicializar Base de Datos (Conexión y Automigración)
	database.Connect()

	// 2. Inicializar Repositorios (Capa de Datos)
	repo := cards.NewRepository(database.DB)

	// 3. Inicializar Servicios (Lógica de Negocio)
	cardsSvc := cards.NewService(repo)

	ygoProv := searchproviders.NewYGOProvider()
	searchSvc := search.NewService(ygoProv)

	syncService := syncsvc.NewSyncService(searchSvc, repo)

	// 4. Inicializar Handlers (Capa de Transporte)
	cardsHandler := handler.NewCardsHandler(cardsSvc)
	searchHandler := handler.NewSearchHandler(searchSvc)
	syncHandler := handler.NewSyncHandler(syncService)

	usersRepo := users.NewRepository(database.DB)
	usersSvc := users.NewService(usersRepo)
	usersHandler := handler.NewUsersHandler(usersSvc)

	invRepo := inventory.NewRepository(database.DB)
	invSvc := inventory.NewService(invRepo)
	inventoryHandler := handler.NewInventoryHandler(invSvc)

	// 5. Configurar e Iniciar Servidor
	srv := api.NewServer(cardsHandler, searchHandler, syncHandler, usersHandler, inventoryHandler)

	if err := srv.Start(":8080"); err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}
