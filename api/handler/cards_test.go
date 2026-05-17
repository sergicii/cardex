package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/operaodev/cardex/internal/cards"
)

// setupTestRouter configura el router de Gin en modo Test y registra las rutas del handler
func setupTestRouter(h *CardsHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	cardsGroup := r.Group("/cards")
	{
		cardsGroup.GET("/search", h.GetSuggestions)
		cardsGroup.GET("/", h.GetCatalog)
		cardsGroup.GET("/:id", h.GetByID)
	}

	return r
}

func TestGetByIDHandler(t *testing.T) {
	mockSvc := &cards.MockService{
		GetByIDFn: func(id uint64) (*cards.Card, error) {
			return &cards.Card{
				ID:          12345,
				EnglishName: "Dark Magician",
				Name:        "Mago Oscuro",
				Lang:        cards.SP,
			}, nil
		},
	}

	h := NewCardsHandler(mockSvc)
	router := setupTestRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/cards/12345", nil)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Se esperaba status 200, se obtuvo %d", w.Code)
	}

	var response cards.Card
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Fallo al parsear el JSON de respuesta: %v", err)
	}

	if response.EnglishName != "Dark Magician" {
		t.Errorf("Se esperaba EnglishName 'Dark Magician', se obtuvo '%s'", response.EnglishName)
	}
}

func TestGetSuggestionsHandler(t *testing.T) {
	mockSvc := &cards.MockService{
		GetSuggestionsFn: func(tcg cards.TCG, lang cards.LangCode, name string) (*cards.SuggestionResult, error) {
			return &cards.SuggestionResult{
				Suggestions: []cards.CatalogFilters{
					{Name: "Dark Magician", TCG: tcg, Lang: lang},
				},
				Results: []cards.RecommendationCardDTO{
					{ID: 1, Name: "Dark Magician", EnglishName: "Dark Magician"},
					{ID: 2, Name: "Dark Magician Girl", EnglishName: "Dark Magician Girl"},
				},
			}, nil
		},
	}
	h := NewCardsHandler(mockSvc)
	router := setupTestRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/cards/search?name=Dark%20Magician&tcg=ygo&lang=en", nil)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Se esperaba status 200, se obtuvo %d", w.Code)
	}

	var response cards.SuggestionResult
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Fallo al parsear el JSON de respuesta: %v", err)
	}

	if len(response.Suggestions) != 1 {
		t.Errorf("Se esperaba 1 sugerencia, se obtuvieron %d", len(response.Suggestions))
	}
	if len(response.Results) != 2 {
		t.Errorf("Se esperaban 2 resultados, se obtuvieron %d", len(response.Results))
	}
}

func TestGetCatalogHandler(t *testing.T) {
	mockSvc := &cards.MockService{
		GetCatalogFn: func(filters cards.CatalogFilters) (*cards.PaginatedResult[cards.SummaryCardDTO], error) {
			return &cards.PaginatedResult[cards.SummaryCardDTO]{
				Data:       []cards.SummaryCardDTO{{ID: 1, Name: "Dark Magician"}},
				Total:      1,
				Page:       1,
				Limit:      20,
				TotalPages: 1,
			}, nil
		},
	}
	h := NewCardsHandler(mockSvc)
	router := setupTestRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/cards/?tcg=ygo&type=Monster&page=1&limit=20", nil)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Se esperaba status 200, se obtuvo %d", w.Code)
	}

	var response cards.PaginatedResult[cards.SummaryCardDTO]
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Fallo al parsear el JSON de respuesta: %v", err)
	}

	if response.Total != 1 {
		t.Errorf("Se esperaba total 1, se obtuvo %d", response.Total)
	}
}
