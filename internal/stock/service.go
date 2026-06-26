package stock

import (
	"fmt"

	"github.com/operaodev/cardex/internal/products"
)

type CreateInput struct {
	UserID     string    `json:"user_id"`
	ProductID  uint64    `json:"product_id"`
	Condition  Condition `json:"condition"`
	Quantity   int       `json:"quantity"`
	Price      float64   `json:"price"`
	IsForSale  bool      `json:"is_for_sale"`
	IsForTrade bool      `json:"is_for_trade"`
	Note       string    `json:"note,omitempty"`
}

type QuantityInput struct {
	StockID uint64 `json:"stock_id"`
	Amount  int    `json:"amount"`
	Note    string `json:"note,omitempty"`
}

type DecreaseInput struct {
	StockID uint64 `json:"stock_id"`
	Amount  int    `json:"amount"`
	Note    string `json:"note,omitempty"`
}

type AdjustmentInput struct {
	StockID     uint64 `json:"stock_id"`
	NewQuantity int    `json:"new_quantity"`
	Note        string `json:"note,omitempty"`
}

type RollbackInput struct {
	StockID uint64 `json:"stock_id"`
	LogID   uint64 `json:"log_id"`
	Note    string `json:"note,omitempty"`
}

type PriceInput struct {
	StockID       uint64  `json:"stock_id"`
	Price         float64 `json:"price"`
	DiscountPrice float64 `json:"discount_price,omitempty"`
	Note          string  `json:"note,omitempty"`
}

type OpenBoxItem struct {
	Product  products.Product `json:"product"`
	Quantity int              `json:"quantity"`
}

type OpenBoxInput struct {
	StockID  uint64        `json:"stock_id"`
	Quantity int           `json:"quantity"`
	Items    []OpenBoxItem `json:"items"`
	Note     string        `json:"note,omitempty"`
}

type Service interface {
	Create(input CreateInput) (*Stock, error)
	Restock(input QuantityInput) (*Stock, error)
	Return(input QuantityInput) (*Stock, error)
	Sale(input DecreaseInput) (*Stock, error)
	Trade(input DecreaseInput) (*Stock, error)
	Gift(input DecreaseInput) (*Stock, error)
	Lost(input DecreaseInput) (*Stock, error)
	Damage(input DecreaseInput) (*Stock, error)
	Adjust(input AdjustmentInput) (*Stock, error)
	Rollback(input RollbackInput) (*Stock, error)
	GetStockByUserID(userID string) ([]Stock, error)
	UpdatePrice(input PriceInput) (*Stock, error)
	ToggleForSale(stockID uint64, isForSale bool) (*Stock, error)
	ToggleForTrade(stockID uint64, isForTrade bool) (*Stock, error)
	GetStockByID(id uint64) (*Stock, error)
	GetLogsByStockID(stockID uint64) ([]Log, error)
	OpenBox(input OpenBoxInput) (*Stock, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(input CreateInput) (*Stock, error) {
	if input.Quantity <= 0 {
		return nil, fmt.Errorf("%w: la cantidad debe ser mayor a 0", ErrInvalidQuantity)
	}

	existing, err := s.repo.FindByUserAndProductAndCondition(input.UserID, input.ProductID, input.Condition)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrStockAlreadyExists
	}

	stock := &Stock{
		UserID:     input.UserID,
		ProductID:  input.ProductID,
		Condition:  input.Condition,
		Quantity:   input.Quantity,
		Price:      input.Price,
		IsForSale:  input.IsForSale,
		IsForTrade: input.IsForTrade,
	}

	if err := s.repo.Create(stock); err != nil {
		return nil, err
	}

	log := &Log{
		StockID:       stock.ID,
		LogType:       LogAdd,
		Delta:         input.Quantity,
		PreviousStock: 0,
		NewStock:      input.Quantity,
		Note:          input.Note,
	}

	if err := s.repo.CreateLog(log); err != nil {
		return nil, err
	}

	return s.repo.FindByID(stock.ID)
}

func (s *service) Restock(input QuantityInput) (*Stock, error) {
	return s.increaseQuantity(input.StockID, input.Amount, LogRestock, input.Note)
}

func (s *service) Return(input QuantityInput) (*Stock, error) {
	return s.increaseQuantity(input.StockID, input.Amount, LogReturn, input.Note)
}

func (s *service) Sale(input DecreaseInput) (*Stock, error) {
	return s.decreaseQuantity(input.StockID, input.Amount, LogSale, input.Note)
}

func (s *service) Trade(input DecreaseInput) (*Stock, error) {
	return s.decreaseQuantity(input.StockID, input.Amount, LogTrade, input.Note)
}

func (s *service) Gift(input DecreaseInput) (*Stock, error) {
	return s.decreaseQuantity(input.StockID, input.Amount, LogGift, input.Note)
}

func (s *service) Lost(input DecreaseInput) (*Stock, error) {
	return s.decreaseQuantity(input.StockID, input.Amount, LogLost, input.Note)
}

func (s *service) Damage(input DecreaseInput) (*Stock, error) {
	return s.decreaseQuantity(input.StockID, input.Amount, LogDamage, input.Note)
}

func (s *service) Adjust(input AdjustmentInput) (*Stock, error) {
	if input.NewQuantity < 0 {
		return nil, fmt.Errorf("%w: la cantidad no puede ser negativa", ErrInvalidQuantity)
	}

	stock, err := s.repo.FindByID(input.StockID)
	if err != nil {
		return nil, err
	}

	previousStock := stock.Quantity
	delta := input.NewQuantity - previousStock

	if err := s.repo.UpdateQuantity(stock.ID, input.NewQuantity); err != nil {
		return nil, err
	}

	log := &Log{
		StockID:       stock.ID,
		LogType:       LogAdjustment,
		Delta:         delta,
		PreviousStock: previousStock,
		NewStock:      input.NewQuantity,
		Note:          input.Note,
	}

	if err := s.repo.CreateLog(log); err != nil {
		return nil, err
	}

	return s.repo.FindByID(stock.ID)
}

func (s *service) Rollback(input RollbackInput) (*Stock, error) {
	targetLog, err := s.repo.FindByLogID(input.LogID)
	if err != nil {
		return nil, err
	}

	if targetLog.StockID != input.StockID {
		return nil, fmt.Errorf("el log no pertenece al stock indicado")
	}

	if !isRollbackable(targetLog.LogType) {
		return nil, fmt.Errorf("%w: %s", ErrRollbackNotAllowed, targetLog.LogType)
	}

	stock, err := s.repo.FindByID(input.StockID)
	if err != nil {
		return nil, err
	}

	// Rollback de precio o descuento
	if targetLog.LogType == LogPriceChange {
		stock.Price = targetLog.PreviousPrice
		if err := s.repo.Update(stock); err != nil {
			return nil, err
		}
		// El hook BeforeUpdate genera el log de price_change automáticamente
		return s.repo.FindByID(stock.ID)
	}

	if targetLog.LogType == LogDiscountChange {
		stock.DiscountPrice = targetLog.PreviousDiscount
		if err := s.repo.Update(stock); err != nil {
			return nil, err
		}
		// El hook BeforeUpdate genera el log de discount_change automáticamente
		return s.repo.FindByID(stock.ID)
	}

	// Rollback de cantidad
	previousStock := stock.Quantity
	newStock := targetLog.PreviousStock
	delta := newStock - previousStock

	if err := s.repo.UpdateQuantity(stock.ID, newStock); err != nil {
		return nil, err
	}

	// Crear log de rollback manualmente para cantidad
	rollbackLog := &Log{
		StockID:       stock.ID,
		LogType:       LogRollback,
		Delta:         delta,
		PreviousStock: previousStock,
		NewStock:      newStock,
		Note:          input.Note,
	}

	if err := s.repo.CreateLog(rollbackLog); err != nil {
		return nil, err
	}

	return s.repo.FindByID(stock.ID)
}

func (s *service) GetStockByUserID(userID string) ([]Stock, error) {
	return s.repo.GetByUserID(userID)
}

func (s *service) GetStockByID(id uint64) (*Stock, error) {
	return s.repo.FindByID(id)
}

func (s *service) GetLogsByStockID(stockID uint64) ([]Log, error) {
	return s.repo.GetLogsByStockID(stockID)
}

func (s *service) OpenBox(input OpenBoxInput) (*Stock, error) {
	if input.Quantity <= 0 {
		return nil, fmt.Errorf("%w: la cantidad debe ser mayor a 0", ErrInvalidQuantity)
	}

	setStock, err := s.repo.FindByID(input.StockID)
	if err != nil {
		return nil, err
	}

	if setStock.Product.Type != products.ProductTypeSet {
		return nil, ErrProductNotASet
	}

	if setStock.Quantity < input.Quantity {
		return nil, fmt.Errorf("%w: disponible %d, solicitado %d", ErrInsufficientStock, setStock.Quantity, input.Quantity)
	}

	for _, item := range input.Items {
		if item.Quantity <= 0 {
			return nil, fmt.Errorf("%w: la cantidad del producto debe ser mayor a 0", ErrInvalidQuantity)
		}
		if item.Product.SetExternalID != setStock.Product.SetExternalID || item.Product.Lang != setStock.Product.Lang {
			return nil, ErrProductNotFromSet
		}
	}

	var updatedStock *Stock
	err = s.repo.RunInTransaction(func(tx Repository) error {
		newSetQty := setStock.Quantity - input.Quantity
		if err := tx.UpdateQuantity(setStock.ID, newSetQty); err != nil {
			return err
		}

		unboxLog := &Log{
			StockID:       setStock.ID,
			LogType:       LogUnboxing,
			Delta:         -input.Quantity,
			PreviousStock: setStock.Quantity,
			NewStock:      newSetQty,
			Note:          input.Note,
		}
		if err := tx.CreateLog(unboxLog); err != nil {
			return err
		}

		for _, item := range input.Items {
			existing, err := tx.FindByUserAndProductAndCondition(setStock.UserID, item.Product.ID, ConditionMint)
			if err != nil {
				return err
			}

			if existing != nil {
				prevQty := existing.Quantity
				newQty := prevQty + item.Quantity
				if err := tx.UpdateQuantity(existing.ID, newQty); err != nil {
					return err
				}
				log := &Log{
					StockID:       existing.ID,
					ParentLogID:   &unboxLog.ID,
					LogType:       LogRestock,
					Delta:         item.Quantity,
					PreviousStock: prevQty,
					NewStock:      newQty,
					Note:          input.Note,
				}
				if err := tx.CreateLog(log); err != nil {
					return err
				}
			} else {
				newStock := &Stock{
					UserID:    setStock.UserID,
					ProductID: item.Product.ID,
					Condition: ConditionMint,
					Quantity:  item.Quantity,
				}
				if err := tx.Create(newStock); err != nil {
					return err
				}
				log := &Log{
					StockID:       newStock.ID,
					ParentLogID:   &unboxLog.ID,
					LogType:       LogAdd,
					Delta:         item.Quantity,
					PreviousStock: 0,
					NewStock:      item.Quantity,
					Note:          input.Note,
				}
				if err := tx.CreateLog(log); err != nil {
					return err
				}
			}
		}

		updatedStock, err = tx.FindByID(setStock.ID)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedStock, nil
}

func (s *service) UpdatePrice(input PriceInput) (*Stock, error) {
	stock, err := s.repo.FindByID(input.StockID)
	if err != nil {
		return nil, err
	}

	previousPrice := stock.Price
	previousDiscount := stock.DiscountPrice

	if err := s.repo.UpdatePrice(input.StockID, input.Price, input.DiscountPrice); err != nil {
		return nil, err
	}

	if input.Price > 0 && input.Price != previousPrice {
		log := &Log{
			StockID:       input.StockID,
			LogType:       LogPriceChange,
			PreviousPrice: previousPrice,
			NewPrice:      input.Price,
			Note:          input.Note,
		}
		if err := s.repo.CreateLog(log); err != nil {
			return nil, err
		}
	}

	if input.DiscountPrice != previousDiscount {
		log := &Log{
			StockID:          input.StockID,
			LogType:          LogDiscountChange,
			PreviousDiscount: previousDiscount,
			NewDiscount:      input.DiscountPrice,
			Note:             input.Note,
		}
		if err := s.repo.CreateLog(log); err != nil {
			return nil, err
		}
	}

	return s.repo.FindByID(input.StockID)
}

func (s *service) ToggleForSale(stockID uint64, isForSale bool) (*Stock, error) {
	stock, err := s.repo.FindByID(stockID)
	if err != nil {
		return nil, err
	}

	stock.IsForSale = isForSale
	if err := s.repo.Update(stock); err != nil {
		return nil, err
	}

	return stock, nil
}

func (s *service) ToggleForTrade(stockID uint64, isForTrade bool) (*Stock, error) {
	stock, err := s.repo.FindByID(stockID)
	if err != nil {
		return nil, err
	}

	stock.IsForTrade = isForTrade
	if err := s.repo.Update(stock); err != nil {
		return nil, err
	}

	return stock, nil
}

func (s *service) increaseQuantity(stockID uint64, amount int, logType LogType, note string) (*Stock, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("%w: la cantidad debe ser mayor a 0", ErrInvalidQuantity)
	}

	stock, err := s.repo.FindByID(stockID)
	if err != nil {
		return nil, err
	}

	previousStock := stock.Quantity
	newStock := previousStock + amount

	if err := s.repo.UpdateQuantity(stock.ID, newStock); err != nil {
		return nil, err
	}

	log := &Log{
		StockID:       stock.ID,
		LogType:       logType,
		Delta:         amount,
		PreviousStock: previousStock,
		NewStock:      newStock,
		Note:          note,
	}

	if err := s.repo.CreateLog(log); err != nil {
		return nil, err
	}

	return s.repo.FindByID(stock.ID)
}

func (s *service) decreaseQuantity(stockID uint64, amount int, logType LogType, note string) (*Stock, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("%w: la cantidad debe ser mayor a 0", ErrInvalidQuantity)
	}

	stock, err := s.repo.FindByID(stockID)
	if err != nil {
		return nil, err
	}

	if stock.Quantity < amount {
		return nil, fmt.Errorf("%w: disponible %d, solicitado %d", ErrInsufficientStock, stock.Quantity, amount)
	}

	previousStock := stock.Quantity
	newStock := previousStock - amount

	if err := s.repo.UpdateQuantity(stock.ID, newStock); err != nil {
		return nil, err
	}

	log := &Log{
		StockID:       stock.ID,
		LogType:       logType,
		Delta:         -amount,
		PreviousStock: previousStock,
		NewStock:      newStock,
		Note:          note,
	}

	if err := s.repo.CreateLog(log); err != nil {
		return nil, err
	}

	return s.repo.FindByID(stock.ID)
}

func isRollbackable(logType LogType) bool {
	switch logType {
	case LogAdd, LogRestock, LogReturn, LogSale, LogTrade, LogGift, LogLost, LogDamage, LogAdjustment, LogPriceChange, LogDiscountChange:
		return true
	default:
		return false
	}
}
