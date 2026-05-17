package cards

type MockService struct {
	GetByIDFn        func(id uint64) (*Card, error)
	GetSuggestionsFn func(tcg TCG, lang LangCode, name string) (*SuggestionResult, error)
	GetCatalogFn     func(filters CatalogFilters) (*PaginatedResult[SummaryCardDTO], error)
}

func (s *MockService) GetByID(id uint64) (*Card, error) {
	return s.GetByIDFn(id)
}

func (s *MockService) GetSuggestions(tcg TCG, lang LangCode, name string) (*SuggestionResult, error) {
	return s.GetSuggestionsFn(tcg, lang, name)
}

func (s *MockService) GetCatalog(filters CatalogFilters) (*PaginatedResult[SummaryCardDTO], error) {
	return s.GetCatalogFn(filters)
}
