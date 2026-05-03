package cards

import "gorm.io/gorm"

// Repository define los métodos que nuestra capa de datos debe tener.
// Ahora las firmas son mucho más limpias gracias al diseño relacional.
type Repository interface {
	Create(info *CardInfo) error
	GetByID(id string) (*Card, error)
	GetByName(name string) ([]Card, error)
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

// Create inserta la información base de la carta y automáticamente inserta
// TODAS las traducciones (Cards) que estén dentro del slice info.Cards.
func (r *repository) Create(info *CardInfo) error {
	// ¡Magia de GORM! Al guardar CardInfo, automáticamente guarda 
	// las Cards relacionadas por la relación HasMany.
	return r.db.Create(info).Error
}

// GetByID busca una traducción específica por su ID exacto (ej. "scg-1234-es").
func (r *repository) GetByID(id string) (*Card, error) {
	var card Card

	// Preload("Info") trae la información base (Tags, Tipo, etc).
	// Preload("Info.Cards") trae TODAS las traducciones disponibles de esta misma carta.
	// Así matamos dos pájaros de un tiro sin hacer queries manuales.
	result := r.db.Preload("Info").Preload("Info.Cards").Where(&Card{ID: id}).First(&card)

	if result.Error != nil {
		return nil, result.Error
	}

	return &card, nil
}

// GetByName busca todas las cartas que coincidan con un nombre específico.
func (r *repository) GetByName(name string) ([]Card, error) {
	var cards []Card

	// Gracias al nuevo modelo, la búsqueda es un simple WHERE.
	// Añadimos % al principio y al final para buscar coincidencias parciales ("mago" encuentra "Mago Oscuro")
	searchPattern := "%" + name + "%"
	result := r.db.Preload("Info").Where("name ILIKE ?", searchPattern).Find(&cards)

	if result.Error != nil {
		return nil, result.Error
	}

	return cards, nil
}
