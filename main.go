package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/operaodev/cardex/internal/cards"
)

func main() {
	// 1. Conectar a PostgreSQL (COMENTADO POR AHORA PORQUE NO HAY DB)
	// database.Connect()

	// 2. Inyección de dependencias (USAMOS EL MOCK EN LUGAR DEL REAL)
	// Creamos un par de cartas falsas para probar
	mockCards := []cards.Card{
		{
			ID:          "YGO-123-SP",
			Lang:        "SP",
			Name:        "Mago Oscuro",
			Description: "El mago supremo.",
		},
		{
			ID:          "YGO-125-EN",
			Lang:        "EN",
			Name:        "Ojama Black",
			Description: "It is very weak, but it can be used as a shield.",
		},
		{
			ID:          "YGO-125-SP",
			Lang:        "SP",
			Name:        "Ojama Negro",
			Description: "Es muy débil, pero se puede usar como escudo.",
		},
		{
			ID:          "YGO-123-EN",
			Lang:        "EN",
			Name:        "Dark Magician",
			Description: "El mago supremo.",
		},
		{
			ID:          "YGO-001-EN",
			Lang:        "EN",
			Name:        "Blue-Eyes White Dragon",
			Description: "This legendary dragon is a powerful engine of destruction.",
		},
	}

	repo := cards.NewMockRepository(mockCards) // <-- Inyectamos el Mock!
	service := cards.NewService(repo)
	handler := cards.NewHandler(service)

	// 3. Iniciar el motor de Gin
	r := gin.Default()

	// 4. Configurar Rutas
	cardsGroup := r.Group("/cards")
	{
		// /cards/search?name=Kuriboh
		cardsGroup.GET("/search", handler.GetByNameHandler)
		// /cards/scg-1234
		cardsGroup.GET("/:id", handler.GetByIDHandler)
	}

	// 5. Iniciar el servidor
	log.Println("Servidor iniciado en http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}
