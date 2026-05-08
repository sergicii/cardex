package cards

import (
	"gorm.io/gorm"
)

// Repository define los métodos que nuestra capa de datos debe tener.
// Ahora las firmas son mucho más limpias gracias al diseño relacional.
type Repository interface {
	Create(card *Card) error
	GetByID(id uint64) (*Card, error)
	GetByName(tcg TCG, name string) ([]Card, error)
}

// repository es la implementación real que usará PostgreSQL y GORM.
type repository struct {
	db *gorm.DB
}

// NewRepository crea una nueva instancia del repositorio de cartas.
func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}

// Create inserta una carta (fila única con todos sus campos).
func (r *repository) Create(card *Card) error {
	return r.db.Create(card).Error
}

// GetByID busca una carta por su ID exacto (ej. 1).
// Al ser tabla única no se necesita ningún join ni preload.
func (r *repository) GetByID(id uint64) (*Card, error) {
	var card Card

	result := r.db.Where(&Card{ID: id}).First(&card)
	if result.Error != nil {
		return nil, result.Error
	}

	return &card, nil
}

// GetByName busca cartas por nombre y carga sus PrintedCards filtradas por el idioma que coincidió.
func (r *repository) GetByName(tcg TCG, name string) ([]Card, error) {
	var cards []Card
	searchPattern := "%" + name + "%"

	// Usamos un JOIN con jsonb_each_text para encontrar qué idioma (k) coincide con la búsqueda (v).
	// Esto nos permite obtener el 'matched_lang' para cada carta.
	result := r.db.Table("cards").
		Select("cards.*, kv.k as matched_lang").
		Joins("JOIN jsonb_each_text(cards.names) AS kv(k, v) ON kv.v ILIKE ?", searchPattern).
		Preload("PrintedCards", "lang IN (SELECT k FROM jsonb_each_text(cards.names) WHERE v ILIKE ?)", searchPattern).
		Find(&cards)

	if result.Error != nil {
		return nil, result.Error
	}

	return cards, nil
}
