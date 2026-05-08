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
		cardsGroup.GET("/search", h.GetByName)
		cardsGroup.GET("/:id", h.GetByID)
	}

	return r
}

func TestGetByIDHandler(t *testing.T) {
	mockSvc := &cards.MockService{
		GetByIDFn: func(id uint64) (*cards.Card, error) {
			return &cards.Card{
				ID:           12345,
				Names:        map[cards.LangCode]string{"en": "Dark Magician"},
				Descriptions: map[cards.LangCode]string{"en": "El mago supremo."},
			}, nil
		},
	}

	h := NewCardsHandler(mockSvc)
	router := setupTestRouter(h)

	// Crear una petición HTTP falsa
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/cards/12345", nil)

	// 2. Act (Ejecutar)
	router.ServeHTTP(w, req)

	// 3. Assert (Verificar resultados)
	if w.Code != http.StatusOK {
		t.Errorf("Se esperaba status 200, se obtuvo %d", w.Code)
	}

	var response cards.Card
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Fallo al parsear el JSON de respuesta: %v", err)
	}

	if response.Names["en"] != "Dark Magician" {
		t.Errorf("Se esperaba nombre 'Dark Magician', se obtuvo '%s'", response.Names["en"])
	}
}

func TestGetByNameHandler(t *testing.T) {
	mockSvc := &cards.MockService{
		GetByNameFn: func(name string) ([]cards.Card, error) {
			return []cards.Card{
				{ID: 1, Names: map[cards.LangCode]string{"en": "Dark Magician"}},
				{ID: 2, Names: map[cards.LangCode]string{"en": "Dark Magician Girl"}},
			}, nil
		},
	}
	h := NewCardsHandler(mockSvc)
	router := setupTestRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/cards/search?name=Dark%20Magician", nil)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Se esperaba status 200, se obtuvo %d", w.Code)
	}

	var response []cards.Card
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Fallo al parsear el JSON de respuesta: %v", err)
	}

	if len(response) != 2 {
		t.Errorf("Se esperaban 2 cartas, se obtuvieron %d", len(response))
	}
}
