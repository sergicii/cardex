package cards

import (
	"fmt"
	"strings"
)

// Service define el contrato de lo que nuestra aplicación puede hacer con las cartas.
type Service interface {
	GetByID(id uint64) (*Card, error)
	GetSuggestions(tcg TCG, lang LangCode, name string) (*SuggestionResult, error)
	GetCatalog(filters CatalogFilters) (*PaginatedResult[SummaryCardDTO], error)
}

// service implementa la interfaz Service e inyecta el repositorio.
type service struct {
	repo Repository
}

// NewService crea una nueva instancia del servicio de cartas.
func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

// GetByID obtiene una carta por su ID delegando al repositorio.
// Se utiliza típicamente para mostrar la vista de detalles completos de una carta específica.
func (s *service) GetByID(id uint64) (*Card, error) {
	if id == 0 {
		return nil, fmt.Errorf("el ID no puede estar vacío")
	}
	return s.repo.GetByID(id)
}

// GetSuggestions obtiene recomendaciones de cartas buscando por nombre.
// Se utiliza en barras de búsqueda o componentes de autocompletado en el frontend.
// Exige un mínimo de 3 caracteres para evitar queries demasiado amplias o poco performantes.
func (s *service) GetSuggestions(tcg TCG, lang LangCode, name string) (*SuggestionResult, error) {
	name = strings.TrimSpace(name)
	if len(name) < 3 {
		return nil, fmt.Errorf("el nombre de búsqueda debe tener al menos 3 caracteres")
	}
	return s.repo.GetSuggestions(tcg, lang, name)
}

// GetCatalog obtiene cartas del catálogo aplicando múltiples filtros con paginación.
// Se utiliza para renderizar galerías de cartas, catálogos principales y búsquedas avanzadas.
// Si Page o Limit no están definidos o son inválidos, se aplican valores por defecto seguros.
func (s *service) GetCatalog(filters CatalogFilters) (*PaginatedResult[SummaryCardDTO], error) {
	if filters.Page <= 0 {
		filters.Page = 1
	}
	if filters.Limit <= 0 {
		filters.Limit = 20
	}
	if filters.Limit > 100 {
		filters.Limit = 100 // protección contra queries demasiado grandes
	}
	return s.repo.GetCatalog(filters)
}
