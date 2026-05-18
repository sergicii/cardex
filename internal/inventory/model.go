package inventory

import (
	"time"

	"github.com/operaodev/cardex/internal/cards"
	"github.com/operaodev/cardex/internal/users"
)

// LogType representa el tipo de acción que generó un movimiento en el inventario.
type LogType string

const (
	// LogSale registra la venta de una o más cartas del inventario.
	LogSale LogType = "venta"
	// LogRestock registra la entrada de nuevas cartas al inventario.
	LogRestock LogType = "restock"
	// LogLoss registra la pérdida (extravío) de cartas del inventario.
	LogLoss LogType = "perdida"
	// LogDamage registra cartas dañadas que ya no son comercializables.
	LogDamage LogType = "daño"
	// LogReturn registra la devolución de cartas (por parte del comprador).
	LogReturn LogType = "devolucion"
	// LogPriceChange registra un cambio de precio sin alterar el stock.
	LogPriceChange LogType = "cambio_de_precio"
)

// Inventory representa la posición actual de una carta en el inventario de un usuario.
// La combinación (user_id, card_id) es única: un usuario solo tiene una entrada por carta.
type Inventory struct {
	ID     uint64 `gorm:"primaryKey;autoIncrement"                           json:"id"`
	UserID string `gorm:"not null;uniqueIndex:idx_inventory_owner,priority:1" json:"user_id"`
	CardID uint64 `gorm:"not null;uniqueIndex:idx_inventory_owner,priority:2" json:"card_id"`
	Count  int    `gorm:"default:1;not null"                                  json:"count"`
	Price  int    `gorm:"default:0;not null"                                  json:"price"` // precio en centavos / unidad mínima

	// Claves foráneas
	User users.User `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"  json:"-"`
	Card cards.Card `gorm:"foreignKey:CardID;references:ID;constraint:OnDelete:RESTRICT" json:"card"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// InventoryLog es el registro inmutable de cada cambio ocurrido en un Inventory.
// Se genera automáticamente en cada operación de creación o modificación.
type InventoryLog struct {
	ID          uint64  `gorm:"primaryKey;autoIncrement" json:"id"`
	InventoryID uint64  `gorm:"not null;index"           json:"inventory_id"`
	LogType     LogType `gorm:"not null;size:20;index"   json:"log_type"`

	// Delta: variación de cantidad (+restock/+devolucion, -venta/-perdida/-daño, 0=cambio_de_precio).
	Delta int `gorm:"not null;default:0" json:"delta"`

	// PreviousPrice y NewPrice se usan en LogPriceChange; en otras acciones pueden ser 0.
	PreviousPrice int `gorm:"default:0" json:"previous_price"`
	NewPrice      int `gorm:"default:0" json:"new_price"`

	// Note es un campo libre para dejar observaciones sobre el movimiento.
	Note string `gorm:"type:text" json:"note,omitempty"`

	// Clave foránea hacia Inventory (CASCADE: si se borra el inventario, se borran sus logs).
	Inventory Inventory `gorm:"foreignKey:InventoryID;references:ID;constraint:OnDelete:CASCADE" json:"-"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}
