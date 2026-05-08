package main

import (
	"log"

	"github.com/operaodev/cardex/api"
	"github.com/operaodev/cardex/api/handler"
	"github.com/operaodev/cardex/internal/cards"
	"github.com/operaodev/cardex/internal/search"
	searchproviders "github.com/operaodev/cardex/internal/search/providers"
)

func main() {
	// 1. Inicializar Repositorios (con datos mock por ahora)
	repo := cards.NewMockRepository(getMockCards())

	// 2. Inicializar Servicios
	cardsSvc := cards.NewService(repo)
	
	ygoProv := searchproviders.NewYGOProvider()
	searchSvc := search.NewService(ygoProv)

	// 3. Inicializar Handlers (Capa de Transporte)
	cardsHandler := handler.NewCardsHandler(cardsSvc)
	searchHandler := handler.NewSearchHandler(searchSvc)

	// 4. Configurar e Iniciar Servidor
	srv := api.NewServer(cardsHandler, searchHandler)

	if err := srv.Start(":8080"); err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}

func getMockCards() []cards.Card {
	return []cards.Card{
		{
			ID:           1,
			Names:        map[cards.LangCode]string{"sp": "Mago Oscuro"},
			Descriptions: map[cards.LangCode]string{"sp": "El mago supremo."},
		},
		{
			ID:           2,
			Names:        map[cards.LangCode]string{"en": "Ojama Black"},
			Descriptions: map[cards.LangCode]string{"en": "It is very weak, but it can be used as a shield."},
		},
		{
			ID:           3,
			Names:        map[cards.LangCode]string{"sp": "Ojama Negro"},
			Descriptions: map[cards.LangCode]string{"sp": "Es muy débil, pero se puede usar como escudo."},
		},
		{
			ID:           4,
			Names:        map[cards.LangCode]string{"en": "Dark Magician"},
			Descriptions: map[cards.LangCode]string{"en": "El mago supremo."},
		},
		{
			ID:           5,
			Names:        map[cards.LangCode]string{"en": "Blue-Eyes White Dragon"},
			Descriptions: map[cards.LangCode]string{"en": "This legendary dragon is a powerful engine of destruction."},
		},
	}
}

