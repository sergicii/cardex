package inventory

import (
	"errors"

	"gorm.io/gorm"
)

// Repository define los métodos de acceso a datos del inventario.
// Toda operación que modifica el stock genera un InventoryLog dentro de la misma transacción.
type Repository interface {
	// FindByUserAndCard devuelve el registro de inventario para (userID, cardID).
	FindByUserAndCard(userID string, cardID uint64) (*Inventory, error)

	// FindByUser devuelve todos los registros de inventario de un usuario.
	FindByUser(userID string) ([]Inventory, error)

	// Upsert crea o actualiza el inventario y registra un log en la misma transacción.
	// Si el registro no existe se crea; si existe, se actualiza Count y/o Price.
	Upsert(inv *Inventory, log *InventoryLog) error

	// GetLogs devuelve el historial de movimientos de un Inventory.
	GetLogs(inventoryID uint64) ([]InventoryLog, error)
}

type repository struct {
	db *gorm.DB
}

// NewRepository crea una nueva instancia del repositorio de inventario.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// FindByUserAndCard busca el registro de inventario de un usuario para una carta específica.
// A nivel SQL: SELECT * FROM inventories WHERE user_id = ? AND card_id = ? LIMIT 1.
func (r *repository) FindByUserAndCard(userID string, cardID uint64) (*Inventory, error) {
	var inv Inventory
	result := r.db.
		Where("user_id = ? AND card_id = ?", userID, cardID).
		First(&inv)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ErrInventoryNotFound
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &inv, nil
}

// FindByUser devuelve todos los registros de inventario de un usuario ordenados por ID.
// A nivel SQL: SELECT * FROM inventories WHERE user_id = ? ORDER BY id ASC.
func (r *repository) FindByUser(userID string) ([]Inventory, error) {
	var items []Inventory
	if err := r.db.Preload("Card").Where("user_id = ?", userID).Order("id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// Upsert guarda (crea o actualiza) el inventario y escribe el InventoryLog dentro de una
// única transacción de base de datos, garantizando que nunca haya un cambio de stock sin log.
//
// A nivel SQL:
//   - INSERT INTO inventories ... ON CONFLICT (user_id, card_id) DO UPDATE SET count=?, price=?, updated_at=?
//   - INSERT INTO inventory_logs (inventory_id, log_type, delta, ...) VALUES (...)
func (r *repository) Upsert(inv *Inventory, log *InventoryLog) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Guardar (crear o actualizar) el registro de inventario
		if err := tx.Save(inv).Error; err != nil {
			return err
		}

		// Vincular el log al inventario recién guardado
		log.InventoryID = inv.ID

		// Insertar el log (registro inmutable, nunca se actualiza)
		if err := tx.Create(log).Error; err != nil {
			return err
		}

		return nil
	})
}

// GetLogs devuelve el historial de movimientos de un inventario específico, ordenado cronológicamente.
// A nivel SQL: SELECT * FROM inventory_logs WHERE inventory_id = ? ORDER BY created_at ASC.
func (r *repository) GetLogs(inventoryID uint64) ([]InventoryLog, error) {
	var logs []InventoryLog
	if err := r.db.
		Where("inventory_id = ?", inventoryID).
		Order("created_at ASC").
		Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}
