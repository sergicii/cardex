package stock

import (
	"errors"

	"gorm.io/gorm"
)

type Repository interface {
	Create(stock *Stock) error
	FindByID(id uint64) (*Stock, error)
	FindByUserAndProductAndCondition(userID string, productID uint64, condition Condition) (*Stock, error)
	GetByUserID(userID string) ([]Stock, error)
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

func (r *repository) GetByUserID(userID string) ([]Stock, error) {
	var stocks []Stock
	result := r.db.Preload("Product").Where("user_id = ?", userID).Find(&stocks)
	if result.Error != nil {
		return nil, result.Error
	}
	return stocks, nil
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
