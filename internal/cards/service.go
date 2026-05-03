package cards

import "fmt"

// Service define el contrato de lo que nuestra aplicación puede hacer
// con las cartas.
type Service interface {
	GetByID(id string) (*Card, error)
	GetByName(name string) ([]Card, error)
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
func (s *service) GetByID(id string) (*Card, error) {
	if id == "" {
		return nil, fmt.Errorf("el ID no puede estar vacío")
	}
	return s.repo.GetByID(id)
}

// GetByName obtiene cartas buscando por nombre.
func (s *service) GetByName(name string) ([]Card, error) {
	if name == "" {
		return nil, fmt.Errorf("el nombre no puede estar vacío")
	}
	return s.repo.GetByName(name)
}