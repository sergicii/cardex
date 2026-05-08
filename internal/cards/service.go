package cards

import (
	"fmt"
	"strings"
)

// Service define el contrato de lo que nuestra aplicación puede hacer
// con las cartas.
type Service interface {
	GetByID(id uint64) (*Card, error)
	GetByName(tcg TCG, name string) ([]Card, error)
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
func (s *service) GetByID(id uint64) (*Card, error) {
	if id == 0 {
		return nil, fmt.Errorf("el ID no puede estar vacío")
	}
	return s.repo.GetByID(id)
}

// GetByName obtiene cartas buscando por nombre.
func (s *service) GetByName(tcg TCG, name string) ([]Card, error) {
	name = strings.TrimSpace(name)

	if len(name) < 3 {
		return nil, fmt.Errorf("el nombre de la carta debe tener al menos 3 caracteres")
	}
	return s.repo.GetByName(tcg, name)
}