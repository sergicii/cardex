package cards

type MockService struct {
	GetByIDFn   func(id uint64) (*Card, error)
	GetByNameFn func(name string) ([]Card, error)
}

func (s *MockService) GetByID(id uint64) (*Card, error) {
	return s.GetByIDFn(id)
}

func (s *MockService) GetByName(name string) ([]Card, error) {
	return s.GetByNameFn(name)
}

