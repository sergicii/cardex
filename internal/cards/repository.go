package cards

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository define los métodos que nuestra capa de datos debe tener.
type Repository interface {
	Create(card *Card) error
	Upsert(cards []Card) (int, error)
	GetByID(id uint64) (*Card, error)
	GetSuggestions(tcg TCG, lang LangCode, name string) (*SuggestionResult, error)
	GetCatalog(filters CatalogFilters) (*PaginatedResult[SummaryCardDTO], error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}

// Create ejecuta un simple INSERT INTO cards en la tabla a través de GORM.
func (r *repository) Create(card *Card) error {
	return r.db.Create(card).Error
}

// Upsert inserta o actualiza un lote de cartas en la DB de forma idempotente.
// A nivel SQL ejecuta un INSERT ... ON CONFLICT (...) DO UPDATE SET ... usando
// la identidad única de la carta para evitar duplicados, procesando todo en lotes.
// Devuelve el número de filas afectadas.
func (r *repository) Upsert(cards []Card) (int, error) {
	const batchSize = 100

	total := 0
	for i := 0; i < len(cards); i += batchSize {
		end := min(i+batchSize, len(cards))
		batch := cards[i:end]

		result := r.db.Clauses(clause.OnConflict{
			// idx_card_identity: ExternalID+Code+Lang+Rarity+SetName es la identidad única
			Columns: []clause.Column{
				{Name: "external_id"},
				{Name: "code"},
				{Name: "lang"},
				{Name: "rarity"},
				{Name: "set_english_name"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"english_name", "name", "description",
				"type", "subtypes", "archetype",
				"sources", "card_images",
				"set_name", "set_english_name", "set_code",
				"rarity", "print", "code",
				"updated_at",
			}),
		}).Create(&batch)

		if result.Error != nil {
			return total, result.Error
		}
		total += int(result.RowsAffected)
	}

	return total, nil
}

// GetByID busca una carta por su ID exacto.
// A nivel SQL ejecuta un SELECT * FROM cards WHERE id = ? ORDER BY id LIMIT 1.
func (r *repository) GetByID(id uint64) (*Card, error) {
	var card Card
	result := r.db.First(&card, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &card, nil
}

// GetSuggestions busca cartas por nombre, tcg y lang y devuelve un SuggestionResult con:
//   - results: todos los prints/ediciones individuales que coinciden (máx. 10).
//   - suggestions: un CatalogFilters único por carta (deduplicado por english_name en memoria).
//
// La deduplicación se hace en Go tras la query: por cada english_name distinto se genera
// un CatalogFilters con el nombre de la carta listo para usarse como filtro de catálogo.
func (r *repository) GetSuggestions(tcg TCG, lang LangCode, name string) (*SuggestionResult, error) {
	var results []RecommendationCardDTO
	searchPattern := "%" + name + "%"

	query := r.db.Model(&Card{}).Select(`
		id,
		tcg,
		name,
		english_name,
		code,
		rarity,
		set_name,
		print,
		lang,
		card_images->0->>'image_url_small' AS image
	`).
		Where("name ILIKE ?", searchPattern).
		Limit(10)

	if tcg != "" {
		query = query.Where("tcg = ?", tcg)
	}
	if lang != "" {
		query = query.Where("lang = ?", lang)
	}

	if err := query.Scan(&results).Error; err != nil {
		return nil, err
	}

	// Construir suggestions: un CatalogFilters por carta única (deduplicado por name).
	seen := make(map[string]struct{})
	var suggestions []CatalogFilters
	for _, card := range results {
		if _, ok := seen[card.Name]; ok {
			continue
		}
		seen[card.Name] = struct{}{}
		suggestions = append(suggestions, CatalogFilters{
			Name: card.Name,
			TCG:  card.TCG,
			Lang: card.Lang,
		})
	}

	return &SuggestionResult{
		Suggestions: suggestions,
		Results:     results,
	}, nil
}

// GetCatalog busca cartas aplicando filtros opcionales con paginación.
// A nivel SQL ejecuta un SELECT count(*) seguido de un SELECT de campos específicos con OFFSET y LIMIT,
// construyendo dinámicamente cláusulas WHERE (incluyendo @> para buscar en JSONB).
// Devuelve un SummaryCardDTO con los campos necesarios para el catálogo.
// Todos los filtros son opcionales; Page y Limit son obligatorios.
func (r *repository) GetCatalog(filters CatalogFilters) (*PaginatedResult[SummaryCardDTO], error) {
	var cards []Card
	var total int64

	query := r.db.Model(&Card{})

	// Aplicar filtros opcionales
	if filters.Name != "" {
		query = query.Where("name ILIKE ?", "%"+filters.Name+"%")
	}
	if filters.TCG != "" {
		query = query.Where("tcg = ?", filters.TCG)
	}
	if filters.Lang != "" {
		query = query.Where("lang = ?", filters.Lang)
	}
	if filters.Type != "" {
		query = query.Where("type = ?", filters.Type)
	}
	if filters.Archetype != "" {
		query = query.Where("archetype ILIKE ?", "%"+filters.Archetype+"%")
	}
	if filters.Subtype != "" {
		query = query.Where("subtypes @> ?", `["`+filters.Subtype+`"]`)
	}
	if filters.SetCode != "" {
		query = query.Where("set_code = ?", filters.SetCode)
	}
	if filters.Rarity != "" {
		query = query.Where("rarity = ?", filters.Rarity)
	}

	// Contar total para la paginación
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Aplicar paginación y recuperar solo los campos necesarios (evitando descripción y fuentes enteras)
	offset := (filters.Page - 1) * filters.Limit
	err := query.Select(`
		id,
		tcg,
		english_name,
		name,
		lang,
		code,
		type,
		subtypes,
		archetype,
		set_name,
		set_code,
		rarity,
		print,
		wanted,
		COALESCE(NULLIF(print_image, ''), card_images->0->>'image_url') AS print_image
	`).
		Offset(offset).Limit(filters.Limit).Order("id ASC").Find(&cards).Error
	if err != nil {
		return nil, err
	}

	// Mapear a DTOs
	results := make([]SummaryCardDTO, len(cards))
	for i, c := range cards {
		results[i] = SummaryCardDTO{
			ID:          c.ID,
			TCG:         c.TCG,
			EnglishName: c.EnglishName,
			Name:        c.Name,
			Lang:        c.Lang,
			Code:        c.Code,
			Type:        c.Type,
			Subtypes:    c.Subtypes,
			Archetype:   c.Archetype,
			SetName:     c.SetName,
			SetCode:     c.SetCode,
			Rarity:      c.Rarity,
			Print:       c.Print,
			Wanted:      c.Wanted,
			Image:       c.PrintImage, // PrintImage ya trae la imagen computada desde SQL
		}
	}

	totalPages := int(total) / filters.Limit
	if int(total)%filters.Limit != 0 {
		totalPages++
	}

	return &PaginatedResult[SummaryCardDTO]{
		Data:       results,
		Total:      int(total),
		Page:       filters.Page,
		Limit:      filters.Limit,
		TotalPages: totalPages,
	}, nil
}
