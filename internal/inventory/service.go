package inventory

import "fmt"

// AddInput contiene los datos para agregar cartas al inventario (restock o primera entrada).
type AddInput struct {
	UserID string `json:"user_id"`
	CardID uint64 `json:"card_id"`
	Count  int    `json:"count"`
	Price  int    `json:"price"`
	Note   string `json:"note"`
}

// SellInput contiene los datos para registrar una venta.
type SellInput struct {
	UserID string `json:"user_id"`
	CardID uint64 `json:"card_id"`
	Count  int    `json:"count"`
	Note   string `json:"note"`
}

// LossInput contiene los datos para registrar pérdidas o daños.
type LossInput struct {
	UserID  string  `json:"user_id"`
	CardID  uint64  `json:"card_id"`
	Count   int     `json:"count"`
	LogType LogType `json:"log_type"` // LogLoss o LogDamage
	Note    string  `json:"note"`
}

// ReturnInput contiene los datos para registrar una devolución.
type ReturnInput struct {
	UserID string `json:"user_id"`
	CardID uint64 `json:"card_id"`
	Count  int    `json:"count"`
	Note   string `json:"note"`
}

// PriceChangeInput contiene los datos para cambiar el precio sin alterar el stock.
type PriceChangeInput struct {
	UserID   string `json:"user_id"`
	CardID   uint64 `json:"card_id"`
	NewPrice int    `json:"new_price"`
	Note     string `json:"note"`
}

// Service define el contrato de operaciones sobre el inventario.
type Service interface {
	// Restock agrega cartas al inventario (entrada de stock o primera vez).
	Restock(input AddInput) (*Inventory, error)

	// Sell registra la venta de cartas, reduciendo el stock.
	Sell(input SellInput) (*Inventory, error)

	// RegisterLoss registra pérdida o daño de cartas, reduciendo el stock.
	RegisterLoss(input LossInput) (*Inventory, error)

	// RegisterReturn registra una devolución, restituyendo el stock.
	RegisterReturn(input ReturnInput) (*Inventory, error)

	// ChangePrice actualiza el precio de una carta en el inventario sin mover stock.
	ChangePrice(input PriceChangeInput) (*Inventory, error)

	// GetInventory devuelve el inventario completo de un usuario.
	GetInventory(userID string) ([]Inventory, error)

	// GetLogs devuelve el historial de movimientos de un registro de inventario.
	GetLogs(inventoryID uint64) ([]InventoryLog, error)
}

type service struct {
	repo Repository
}

// NewService crea una nueva instancia del servicio de inventario.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// Restock añade unidades al inventario existente (o lo crea si es la primera vez).
// Genera un log de tipo LogRestock automáticamente.
func (s *service) Restock(input AddInput) (*Inventory, error) {
	if input.Count <= 0 {
		return nil, ErrInvalidDelta
	}
	if input.Price < 0 {
		return nil, ErrInvalidPrice
	}

	inv, err := s.repo.FindByUserAndCard(input.UserID, input.CardID)
	if err != nil && err != ErrInventoryNotFound {
		return nil, err
	}

	if inv == nil {
		// Primera entrada: crear el registro
		inv = &Inventory{
			UserID: input.UserID,
			CardID: input.CardID,
			Count:  input.Count,
			Price:  input.Price,
		}
	} else {
		inv.Count += input.Count
		if input.Price > 0 {
			inv.Price = input.Price
		}
	}

	log := &InventoryLog{
		LogType: LogRestock,
		Delta:   +input.Count,
		Note:    input.Note,
	}

	if err := s.repo.Upsert(inv, log); err != nil {
		return nil, fmt.Errorf("error al registrar restock: %w", err)
	}
	return inv, nil
}

// Sell descuenta unidades del inventario y genera un log de tipo LogSale.
// Devuelve ErrInsufficientStock si el stock actual es menor que la cantidad a vender.
func (s *service) Sell(input SellInput) (*Inventory, error) {
	if input.Count <= 0 {
		return nil, ErrInvalidDelta
	}

	inv, err := s.repo.FindByUserAndCard(input.UserID, input.CardID)
	if err != nil {
		return nil, err
	}
	if inv.Count < input.Count {
		return nil, ErrInsufficientStock
	}

	inv.Count -= input.Count

	log := &InventoryLog{
		LogType: LogSale,
		Delta:   -input.Count,
		Note:    input.Note,
	}

	if err := s.repo.Upsert(inv, log); err != nil {
		return nil, fmt.Errorf("error al registrar venta: %w", err)
	}
	return inv, nil
}

// RegisterLoss descuenta unidades por pérdida o daño y genera el log correspondiente.
// LogType debe ser LogLoss o LogDamage; se valida en el servicio.
func (s *service) RegisterLoss(input LossInput) (*Inventory, error) {
	if input.Count <= 0 {
		return nil, ErrInvalidDelta
	}
	if input.LogType != LogLoss && input.LogType != LogDamage {
		return nil, fmt.Errorf("log_type inválido para pérdida: use '%s' o '%s'", LogLoss, LogDamage)
	}

	inv, err := s.repo.FindByUserAndCard(input.UserID, input.CardID)
	if err != nil {
		return nil, err
	}
	if inv.Count < input.Count {
		return nil, ErrInsufficientStock
	}

	inv.Count -= input.Count

	log := &InventoryLog{
		LogType: input.LogType,
		Delta:   -input.Count,
		Note:    input.Note,
	}

	if err := s.repo.Upsert(inv, log); err != nil {
		return nil, fmt.Errorf("error al registrar %s: %w", input.LogType, err)
	}
	return inv, nil
}

// RegisterReturn restituye unidades al inventario por devolución de comprador.
// Genera un log de tipo LogReturn.
func (s *service) RegisterReturn(input ReturnInput) (*Inventory, error) {
	if input.Count <= 0 {
		return nil, ErrInvalidDelta
	}

	inv, err := s.repo.FindByUserAndCard(input.UserID, input.CardID)
	if err != nil {
		return nil, err
	}

	inv.Count += input.Count

	log := &InventoryLog{
		LogType: LogReturn,
		Delta:   +input.Count,
		Note:    input.Note,
	}

	if err := s.repo.Upsert(inv, log); err != nil {
		return nil, fmt.Errorf("error al registrar devolución: %w", err)
	}
	return inv, nil
}

// ChangePrice actualiza el precio de una carta sin modificar el stock.
// Genera un log de tipo LogPriceChange con PreviousPrice y NewPrice.
func (s *service) ChangePrice(input PriceChangeInput) (*Inventory, error) {
	if input.NewPrice < 0 {
		return nil, ErrInvalidPrice
	}

	inv, err := s.repo.FindByUserAndCard(input.UserID, input.CardID)
	if err != nil {
		return nil, err
	}

	log := &InventoryLog{
		LogType:       LogPriceChange,
		Delta:         0,
		PreviousPrice: inv.Price,
		NewPrice:      input.NewPrice,
		Note:          input.Note,
	}

	inv.Price = input.NewPrice

	if err := s.repo.Upsert(inv, log); err != nil {
		return nil, fmt.Errorf("error al cambiar precio: %w", err)
	}
	return inv, nil
}

// GetInventory devuelve todos los registros de inventario de un usuario.
func (s *service) GetInventory(userID string) ([]Inventory, error) {
	if userID == "" {
		return nil, fmt.Errorf("el user_id es obligatorio")
	}
	return s.repo.FindByUser(userID)
}

// GetLogs devuelve el historial de movimientos de un registro de inventario.
func (s *service) GetLogs(inventoryID uint64) ([]InventoryLog, error) {
	if inventoryID == 0 {
		return nil, fmt.Errorf("el inventory_id es obligatorio")
	}
	return s.repo.GetLogs(inventoryID)
}
