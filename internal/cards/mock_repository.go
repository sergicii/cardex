package cards

import (
	"fmt"
	"strings"
)

type mockRepository struct {
	cards []Card
}

func NewMockRepository(cards []Card) Repository {
	return &mockRepository{
		cards: cards,
	}
}

func (m *mockRepository) GetByID(id string) (*Card, error) {
	for _, card := range m.cards {
		if card.ID == id {
			return &card, nil
		}
	}
	return nil, fmt.Errorf("card not found: %s", id)
}

func (m *mockRepository) GetByName(name string) ([]Card, error) {
	var results []Card
	searchName := strings.ToLower(name)
	for _, card := range m.cards {
		// Simulamos el comportamiento de ILIKE %name% usando strings.Contains y ToLower
		if strings.Contains(strings.ToLower(card.Name), searchName) {
			results = append(results, card)
		}
	}
	return results, nil
}

// Create es un mock vacío para cumplir con la interfaz Repository
func (m *mockRepository) Create(info *CardInfo) error {
	return nil
}
