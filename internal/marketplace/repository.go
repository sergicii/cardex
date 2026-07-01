package marketplace

import (
	"github.com/operaodev/cardex/internal/stock"
	"gorm.io/gorm"
)

type Repository interface {
	GetPrices(id uint64) (MarketAnalysis, error)
	GetOffers(input OffersInput) (OffersPage, error)
	GetCards(input FilterInput) (ProductResumePage, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}

func (r *repository) GetPrices(id uint64) (MarketAnalysis, error) {
	var analysis MarketAnalysis

	result := r.db.Table("stocks").
		Joins("JOIN users ON users.id = stocks.user_id").
		Select(`
			COALESCE(MIN(stocks.price), 0) AS low_price,
			COALESCE(AVG(stocks.price), 0) AS average_price,
			COALESCE(MAX(stocks.price), 0) AS high_price,
			COALESCE(SUM(stocks.quantity), 0) AS market_stocks
		`).
		Where("stocks.product_id = ?", id).
		Where("stocks.is_for_sale = ?", true).
		Where("users.is_guest = ?", false).
		Scan(&analysis)

	if result.Error != nil {
		return MarketAnalysis{}, result.Error
	}

	analysis.ProductId = id

	return analysis, nil
}

func (r *repository) GetOffers(input OffersInput) (OffersPage, error) {
	var page OffersPage
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

	// Count total
	query := r.db.Table("stocks").
		Joins("JOIN users ON users.id = stocks.user_id").
		Where("users.is_guest = ?", false).
		Where("stocks.product_id = ?", input.ProductID).
		Where("stocks.is_for_sale = ? OR stocks.is_for_trade = ?", true, true)
	if input.ForSale != nil {
		query = query.Where("stocks.is_for_sale = ?", *input.ForSale)
	}
	if input.ForTrade != nil {
		query = query.Where("stocks.is_for_trade = ?", *input.ForTrade)
	}
	if input.HasStock != nil && *input.HasStock {
		query = query.Where("stocks.quantity > 0")
	}

	query.Count(&page.Total)

	// Fetch page
	var stocks []stock.Stock
	order := "price ASC"
	if input.SortDesc {
		order = "price DESC"
	}

	findQuery := r.db.Preload("User").
		Table("stocks").
		Joins("JOIN users ON users.id = stocks.user_id").
		Where("users.is_guest = ?", false).
		Where("stocks.product_id = ?", input.ProductID).
		Where("stocks.is_for_sale = ? OR stocks.is_for_trade = ?", true, true)
	if input.ForSale != nil {
		findQuery = findQuery.Where("stocks.is_for_sale = ?", *input.ForSale)
	}
	if input.ForTrade != nil {
		findQuery = findQuery.Where("stocks.is_for_trade = ?", *input.ForTrade)
	}
	if input.HasStock != nil {
		if *input.HasStock {
			findQuery = findQuery.Where("stocks.quantity > 0")
		} else {
			findQuery = findQuery.Where("stocks.quantity <= 0")
		}
	}

	result := findQuery.Order(order).Offset(offset).Limit(input.Limit).Find(&stocks)

	if result.Error != nil {
		return page, result.Error
	}

	offers := make([]Offer, 0, len(stocks))
	for _, s := range stocks {
		var discount float64 = 0
		if s.Price > 0 {
			discount = ((s.Price - s.DiscountPrice) / s.Price) * 100
		}
		offers = append(offers, Offer{
			User:          s.User,
			StockID:       s.ID,
			Condition:     s.Condition,
			IsForTrade:    s.IsForTrade,
			Price:         float64(s.Price),
			DiscountPrice: float64(s.DiscountPrice),
			Discount:      discount,
			Quantity:      uint(s.Quantity),
		})
	}

	page.Items = offers
	page.Page = input.Page
	page.Limit = input.Limit
	page.TotalPages = int((page.Total + int64(input.Limit) - 1) / int64(input.Limit))

	return page, nil
}

func (r *repository) GetCards(input FilterInput) (ProductResumePage, error) {
	var page ProductResumePage

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

	stockSubquery := r.db.Table("stocks").
		Select("product_id, COALESCE(SUM(quantity), 0) AS global_stock, COALESCE(AVG(price), 0) AS average_price").
		Where("is_for_sale = ?", true).
		Group("product_id")

	namePattern := "%" + input.Input + "%"
	otherPattern := input.Input + "%"

	baseQuery := func() *gorm.DB {
		q := r.db.Table("products AS p").
			Joins("LEFT JOIN (?) AS st ON st.product_id = p.id", stockSubquery)

		q = q.Where("(p.set_external_id ILIKE ? OR p.name ILIKE ? OR p.code ILIKE ? OR p.archetype ILIKE ?)",
			namePattern, namePattern, otherPattern, otherPattern)

		if input.ProductType != "" {
			q = q.Where("p.type = ?", input.ProductType)
		}

		if len(input.TCGs) > 0 {
			q = q.Where("p.tcg IN ?", input.TCGs)
		}
		if len(input.Langs) > 0 {
			q = q.Where("p.lang IN ?", input.Langs)
		}

		return q
	}

	countQuery := baseQuery()
	countQuery.Count(&page.Total)

	var items []ProductResume
	err := baseQuery().
		Select(`p.id, p.name, COALESCE(p.code, '') AS code, p.set_name,
			p.rarity, p.rarity_code,
			COALESCE(st.global_stock, 0) AS global_stock,
			COALESCE(st.average_price, 0) AS average_price,
			COALESCE(
				NULLIF(p.print_url_large, ''),
				p.images->0->>'image_url',
				p.set_image_large,
				''
			) AS image`).
		Order("p.wanted DESC").
		Offset(offset).
		Limit(input.Limit).
		Scan(&items).Error

	if err != nil {
		return page, err
	}

	page.Items = items
	page.Page = input.Page
	page.Limit = input.Limit
	page.TotalPages = int((page.Total + int64(input.Limit) - 1) / int64(input.Limit))

	return page, nil
}
