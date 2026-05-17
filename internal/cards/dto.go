package cards

// RecommendationCardDTO representa una carta en el autocompletado/buscador.
// Contiene los datos mínimos para mostrar sugerencias al usuario.
type RecommendationCardDTO struct {
	ID          uint64 `json:"id"`
	TCG         TCG    `json:"tcg"`
	Name        string `json:"name"`
	EnglishName string `json:"english_name"`
	Code        string `json:"code"`
	Rarity      Rarity `json:"rarity"`
	SetName     string `json:"set_name"`
	Image       string `json:"image"`
	Print       Print  `json:"print"`
	Lang        LangCode `json:"lang"`
}

// SummaryCardDTO representa una carta en el catálogo/listado.
// Se usa en el resultado de búsquedas por filtros.
type SummaryCardDTO struct {
	ID          uint64   `json:"id"`
	TCG         TCG      `json:"tcg"`
	EnglishName string   `json:"english_name"`
	Name        string   `json:"name"`
	Lang        LangCode `json:"lang"`
	Code        string   `json:"code"`
	Type        string   `json:"type"`
	Subtypes    []string `json:"subtypes"`
	Archetype   string   `json:"archetype,omitempty"`
	SetName     string   `json:"set_name"`
	SetCode     string   `json:"set_code,omitempty"`
	Rarity      Rarity   `json:"rarity"`
	Print       Print    `json:"print,omitempty"`
	Wanted      uint     `json:"wanted"`
	Image       string   `json:"image"`
}

type SuggestionResult struct {
	Results     []RecommendationCardDTO `json:"results"`
	Suggestions []CatalogFilters        `json:"suggestions"`
}

// PaginatedResult envuelve una lista de resultados con metadata de paginación.
type PaginatedResult[T any] struct {
	Data       []T `json:"data"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalPages int `json:"total_pages"`
}

// CatalogFilters define los criterios de búsqueda para el catálogo.
// Page y Limit son obligatorios para todas las queries de listado.
type CatalogFilters struct {
	Name      string   `json:"name"`
	TCG       TCG      `json:"tcg"`
	Lang      LangCode `json:"lang"`
	Archetype string   `json:"archetype"`
	Type      string   `json:"type"`
	Subtype   string   `json:"subtype"`
	SetCode   string   `json:"set_code"`
	Rarity    Rarity   `json:"rarity"`
	Page      int      `json:"page"`
	Limit     int      `json:"limit"`
}
