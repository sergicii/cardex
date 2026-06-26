package products

import (
	"math/rand/v2"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository interface {
	Upsert(products []Product) (int, error)
	GetByID(id uint64) (*Product, error)
	GetRelatedCards(input RelatedCardsInput) (*RelatedCardsResponse, error)
	GetCardsBySet(setExternalID string, lang LangCode) ([]Product, error)

	GetRandomNames(count int) ([]string, error)
	GetSuggestions(input SuggestionInput) ([]SuggestionDTO, error)
	GetSuggestionsByUser(userID string, input SuggestionInput) ([]SuggestionDTO, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}

func (r *repository) Upsert(products []Product) (int, error) {
	const batchSize = 100
	total := 0

	for i := 0; i < len(products); i += batchSize {
		end := min(i+batchSize, len(products))
		batch := products[i:end]

		result := r.db.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "external_id"},
				{Name: "set_external_id"},
				{Name: "tcg"},
				{Name: "code"},
				{Name: "lang"},
				{Name: "rarity"},
				{Name: "edition"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"edition",
				"print_url_small",
				"print_url_large",
				"quantity_per_set",
				"set_region_code",
				"set_image_small",
				"set_image_large",
				"updated_at",
			}),
		}).Create(&batch)

		if result.Error != nil {
			return 0, result.Error
		}
		total += int(result.RowsAffected)
	}

	return total, nil
}

func (r *repository) GetByID(id uint64) (*Product, error) {
	var product Product
	result := r.db.Where("id = ?", id).First(&product)
	if result.Error != nil {
		return nil, result.Error
	}

	go r.db.Model(&product).UpdateColumn("wanted", gorm.Expr("wanted + 1"))

	return &product, nil
}

func (r *repository) GetSuggestions(input SuggestionInput) ([]SuggestionDTO, error) {
	var suggestions []SuggestionDTO

	// El SELECT incluye todos los campos del DTO más wanted para el ORDER BY externo
	baseSelect := `
		id,
		external_id,
		set_external_id,
		type,
		tcg,
		name,
		code,
		rarity,
		rarity_code,
		set_name,
		set_code,
		set_region_code,
		lang,
		edition,
		wanted,
		COALESCE(
			NULLIF(print_url_small, ''),
			images->0->>'image_url_small',
			set_image_small,
			''
		) AS image
	`

	// Construimos filtros opcionales de TCG e Idioma comunes
	filterQuery := r.db.Table("products")
	if input.TCG != "" {
		filterQuery = filterQuery.Where("tcg = ?", input.TCG)
	}
	if input.Lang != "" {
		filterQuery = filterQuery.Where("lang = ?", input.Lang)
	}

	namePattern := "%" + input.Input + "%"
	otherPattern := input.Input + "%"

	// Consulta única con CASE para prioridad y OR en WHERE.
	// CASE asigna la prioridad según qué columna hizo match (la primera que coincida).
	err := filterQuery.
		Select(baseSelect+`,
			CASE
				WHEN name ILIKE ? THEN 1
				WHEN code ILIKE ? THEN 2
				WHEN archetype ILIKE ? THEN 3
				WHEN external_id ILIKE ? THEN 4
			END AS priority
		`, namePattern, otherPattern, otherPattern, otherPattern).
		Where(`(
			name ILIKE ?
			OR code ILIKE ?
			OR archetype ILIKE ?
			OR external_id ILIKE ?
		)`, namePattern, otherPattern, otherPattern, otherPattern).
		Order("wanted DESC, priority ASC").
		Limit(20).
		Find(&suggestions).Error

	if err != nil {
		return nil, err
	}

	return suggestions, nil
}

func (r *repository) GetRelatedCards(input RelatedCardsInput) (*RelatedCardsResponse, error) {
	var all []RelatedCardDTO
	r.db.Table("products").
		Select(`
			id, external_id, set_external_id, type, tcg,
			name, code, rarity, rarity_code,
			set_name, set_code, lang, edition, wanted,
			COALESCE(
				NULLIF(print_url_small, ''),
				images->0->>'image_url_small',
				set_image_small, ''
			) AS image
		`).
		Where("external_id = ?", input.ExternalID).
		Where("set_external_id = ?", input.SetExternalID).
		Where("tcg = ?", input.TCG).
		Where("id != ?", input.ID).
		Order("wanted DESC").
		Find(&all)

	same := make([]RelatedCardDTO, 0, len(all))
	different := make([]RelatedCardDTO, 0, len(all))
	for _, row := range all {
		if row.Lang == input.Lang {
			same = append(same, row)
		} else {
			different = append(different, row)
		}
	}

	return &RelatedCardsResponse{
		SameLangDifferentRarity: same,
		DifferentLang:           different,
	}, nil
}

func (r *repository) GetSuggestionsByUser(userID string, input SuggestionInput) ([]SuggestionDTO, error) {
	var suggestions []SuggestionDTO

	baseSelect := `
		p.id,
		p.external_id,
		p.set_external_id,
		p.type,
		p.tcg,
		p.name,
		p.code,
		p.rarity,
		p.rarity_code,
		p.set_name,
		p.set_code,
		p.lang,
		p.edition,
		p.wanted,
		COALESCE(
			NULLIF(p.print_url_small, ''),
			p.images->0->>'image_url_small',
			p.set_image_small,
			''
		) AS image
	`

	filterQuery := r.db.Table("products AS p")
	if input.TCG != "" {
		filterQuery = filterQuery.Where("p.tcg = ?", input.TCG)
	}
	if input.Lang != "" {
		filterQuery = filterQuery.Where("p.lang = ?", input.Lang)
	}

	namePattern := "%" + input.Input + "%"
	otherPattern := input.Input + "%"

	err := filterQuery.
		Select(baseSelect+`,
			COALESCE(SUM(s.quantity), 0) AS stock,
			COALESCE(SUM(w.quantity), 0) AS copies_in_wishlist,
			CASE
				WHEN p.name ILIKE ? THEN 1
				WHEN p.code ILIKE ? THEN 2
				WHEN p.archetype ILIKE ? THEN 3
				WHEN p.external_id ILIKE ? THEN 4
			END AS priority
		`, namePattern, otherPattern, otherPattern, otherPattern).
		Joins("LEFT JOIN stocks AS s ON s.product_id = p.id AND s.user_id = ?", userID).
		Joins("LEFT JOIN wishlists AS w ON w.product_id = p.id AND w.user_id = ?", userID).
		Where(`(
			p.name ILIKE ?
			OR p.code ILIKE ?
			OR p.archetype ILIKE ?
			OR p.external_id ILIKE ?
		)`, namePattern, otherPattern, otherPattern, otherPattern).
		Group("p.id").
		Order("p.wanted DESC, priority ASC").
		Limit(10).
		Find(&suggestions).Error

	if err != nil {
		return nil, err
	}

	return suggestions, nil
}

func (r *repository) GetRandomNames(count int) ([]string, error) {
	var rows []struct {
		Name      string
		Code      string
		Archetype string
	}

	r.db.Table("products").
		Select("name, COALESCE(code, '') AS code, COALESCE(archetype, '') AS archetype").
		Where("lang = ?", "EN").
		Find(&rows)

	seen := make(map[string]bool)
	var values []string
	for _, row := range rows {
		for _, v := range []string{row.Name, row.Code, row.Archetype} {
			if v != "" && !seen[v] {
				seen[v] = true
				values = append(values, v)
			}
		}
	}

	rand.Shuffle(len(values), func(i, j int) {
		values[i], values[j] = values[j], values[i]
	})
	if len(values) > count {
		values = values[:count]
	}

	return values, nil
}

func (r *repository) GetCardsBySet(setExternalID string, lang LangCode) ([]Product, error) {
	var cards []Product
	result := r.db.
		Where("set_external_id = ? AND lang = ? AND type = ?", setExternalID, lang, ProductTypeCard).
		Order("serie_code ASC").
		Find(&cards)
	if result.Error != nil {
		return nil, result.Error
	}
	return cards, nil
}
