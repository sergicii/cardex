package stock

import (
	"errors"
	"strings"

	"github.com/operaodev/cardex/internal/products"
	"gorm.io/gorm"
)

type Repository interface {
	Create(stock *Stock) error
	FindByID(id uint64) (*Stock, error)
	FindByUserAndProductAndCondition(userID string, productID uint64, condition Condition) (*Stock, error)
	GetByUserID(userID string, input FilterInput) (StockPage, error)
	GetFiltersByUserID(userID string, input FilterInput) (FilterOutput, error)
	UpdateQuantity(id uint64, quantity int) error
	UpdatePrice(id uint64, price, discountPrice float64) error
	Update(stock *Stock) error

	CreateLog(log *Log) error
	FindByLogID(id uint64) (*Log, error)
	GetLogsByStockID(stockID uint64) ([]Log, error)

	RunInTransaction(fn func(tx Repository) error) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(stock *Stock) error {
	return r.db.Create(stock).Error
}

func (r *repository) FindByID(id uint64) (*Stock, error) {
	var stock Stock
	result := r.db.Preload("Product").Where("id = ?", id).First(&stock)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ErrStockNotFound
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &stock, nil
}

func (r *repository) FindByUserAndProductAndCondition(userID string, productID uint64, condition Condition) (*Stock, error) {
	var stock Stock
	result := r.db.Preload("Product").
		Where("user_id = ? AND product_id = ? AND condition = ?", userID, productID, condition).
		First(&stock)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &stock, nil
}

func (r *repository) GetByUserID(userID string, input FilterInput) (StockPage, error) {
	var page StockPage

	if input.Limit <= 0 {
		input.Limit = 20
	}
	if input.Limit > 100 {
		input.Limit = 100
	}
	if input.Page <= 0 {
		input.Page = 1
	}
	offset := (input.Page - 1) * input.Limit

	baseQuery := func() *gorm.DB {
		q := r.db.Table("stocks AS s").
			Joins("JOIN products AS p ON p.id = s.product_id").
			Where("s.user_id = ?", userID)

		if term := strings.TrimSpace(input.Input); term != "" {
			pattern := "%" + term + "%"
			q = q.Where(
				"p.name ILIKE ? OR p.code ILIKE ? OR p.set_region_code ILIKE ?",
				pattern, pattern, pattern,
			)
		}
		if input.Type != "" {
			q = q.Where("p.type = ?", input.Type)
		}
		if input.TCG != "" {
			q = q.Where("p.tcg = ?", input.TCG)
		}
		if input.Lang != "" {
			q = q.Where("p.lang = ?", input.Lang)
		}
		if input.SetName != "" {
			q = q.Where("p.set_name = ?", input.SetName)
		}
		if input.Archetype != "" {
			q = q.Where("p.archetype = ?", input.Archetype)
		}
		if input.Rarity != "" {
			q = q.Where("p.rarity = ?", input.Rarity)
		}
		if input.Edition != "" {
			q = q.Where("p.edition = ?", input.Edition)
		}
		return q
	}

	baseQuery().Count(&page.Total)

	var stocks []Stock
	if err := baseQuery().
		Preload("Product").
		Select("s.*").
		Offset(offset).
		Limit(input.Limit).
		Find(&stocks).Error; err != nil {
		return page, err
	}

	page.Items = stocks
	page.Page = input.Page
	page.Limit = input.Limit
	page.TotalPages = int((page.Total + int64(input.Limit) - 1) / int64(input.Limit))

	return page, nil
}

// filterRow es la proyección plana usada internamente por GetFiltersByUserID.
type filterRow struct {
	Type          products.ProductType `gorm:"column:type"`
	TCG           products.TCG         `gorm:"column:tcg"`
	Lang          products.LangCode    `gorm:"column:lang"`
	SetName       string               `gorm:"column:set_name"`
	Archetype     string               `gorm:"column:archetype"`
	Rarity        string               `gorm:"column:rarity"`
	Edition       string               `gorm:"column:edition"`
}

func (r *repository) GetFiltersByUserID(userID string, input FilterInput) (FilterOutput, error) {
	q := r.db.Table("stocks AS s").
		Joins("JOIN products AS p ON p.id = s.product_id").
		Where("s.user_id = ?", userID)

	if term := strings.TrimSpace(input.Input); term != "" {
		pattern := "%" + term + "%"
		q = q.Where(
			"p.name ILIKE ? OR p.code ILIKE ? OR p.set_region_code ILIKE ?",
			pattern, pattern, pattern,
		)
	}

	var rows []filterRow
	if err := q.Select(
		"DISTINCT p.type, p.tcg, p.lang, p.set_name, p.archetype, p.rarity, p.edition",
	).Scan(&rows).Error; err != nil {
		return FilterOutput{}, err
	}

	// Deduplica en sets Go para mayor seguridad
	typeSet := map[products.ProductType]struct{}{}
	tcgSet := map[products.TCG]struct{}{}
	langSet := map[products.LangCode]struct{}{}
	setNameSet := map[string]struct{}{}
	archetypeSet := map[string]struct{}{}
	raritySet := map[string]struct{}{}
	editionSet := map[string]struct{}{}

	for _, row := range rows {
		if row.Type != "" {
			typeSet[row.Type] = struct{}{}
		}
		if row.TCG != "" {
			tcgSet[row.TCG] = struct{}{}
		}
		if row.Lang != "" {
			langSet[row.Lang] = struct{}{}
		}
		if row.SetName != "" {
			setNameSet[row.SetName] = struct{}{}
		}
		if row.Archetype != "" {
			archetypeSet[row.Archetype] = struct{}{}
		}
		if row.Rarity != "" {
			raritySet[row.Rarity] = struct{}{}
		}
		if row.Edition != "" {
			editionSet[row.Edition] = struct{}{}
		}
	}

	out := FilterOutput{
		Type:      make([]products.ProductType, 0, len(typeSet)),
		TCG:       make([]products.TCG, 0, len(tcgSet)),
		Lang:      make([]products.LangCode, 0, len(langSet)),
		SetName:   make([]string, 0, len(setNameSet)),
		Archetype: make([]string, 0, len(archetypeSet)),
		Rarity:    make([]string, 0, len(raritySet)),
		Edition:   make([]string, 0, len(editionSet)),
	}
	for v := range typeSet {
		out.Type = append(out.Type, v)
	}
	for v := range tcgSet {
		out.TCG = append(out.TCG, v)
	}
	for v := range langSet {
		out.Lang = append(out.Lang, v)
	}
	for v := range setNameSet {
		out.SetName = append(out.SetName, v)
	}
	for v := range archetypeSet {
		out.Archetype = append(out.Archetype, v)
	}
	for v := range raritySet {
		out.Rarity = append(out.Rarity, v)
	}
	for v := range editionSet {
		out.Edition = append(out.Edition, v)
	}

	return out, nil
}

func (r *repository) UpdateQuantity(id uint64, quantity int) error {
	result := r.db.Model(&Stock{}).Where("id = ?", id).Update("quantity", quantity)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrStockNotFound
	}
	return nil
}

func (r *repository) UpdatePrice(id uint64, price, discountPrice float64) error {
	result := r.db.Model(&Stock{}).Where("id = ?", id).Updates(map[string]interface{}{
		"price":          price,
		"discount_price": discountPrice,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrStockNotFound
	}
	return nil
}

func (r *repository) Update(stock *Stock) error {
	return r.db.Save(stock).Error
}

func (r *repository) CreateLog(log *Log) error {
	return r.db.Create(log).Error
}

func (r *repository) FindByLogID(id uint64) (*Log, error) {
	var log Log
	result := r.db.Where("id = ?", id).First(&log)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ErrLogNotFound
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &log, nil
}

func (r *repository) GetLogsByStockID(stockID uint64) ([]Log, error) {
	var logs []Log
	result := r.db.Where("stock_id = ?", stockID).Order("created_at DESC").Find(&logs)
	if result.Error != nil {
		return nil, result.Error
	}
	return logs, nil
}

func (r *repository) RunInTransaction(fn func(tx Repository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(&repository{db: tx})
	})
}
