package stock

import (
	"time"

	"github.com/operaodev/cardex/internal/products"
	"github.com/operaodev/cardex/internal/users"
	"gorm.io/gorm"
)

type Condition string

const (
	ConditionMint        Condition = "mint"
	ConditionNearMint    Condition = "near_mint"
	ConditionLightPlayed Condition = "light_played"
	ConditionModPlayed   Condition = "mod_played"
	ConditionHeavyPlayed Condition = "heavy_played"
	ConditionDamaged     Condition = "damaged"
)

type Stock struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"                            json:"id"`
	UserID    string    `gorm:"not null;type:uuid;uniqueIndex:idx_stock_owner,priority:1" json:"user_id"`
	ProductID uint64    `gorm:"not null;uniqueIndex:idx_stock_owner,priority:2" json:"product_id"`
	Condition Condition `gorm:"not null;uniqueIndex:idx_stock_owner,priority:3" json:"condition"`

	IsForSale     bool    `gorm:"default:true;not null"                json:"is_for_sale"`
	IsForTrade    bool    `gorm:"default:false;not null"               json:"is_for_trade"`
	Quantity      int     `gorm:"default:1;not null"                   json:"quantity"`
	Price         float64 `gorm:"type:numeric(10,2);default:0;not null" json:"price"`
	DiscountPrice float64 `gorm:"type:numeric(10,2);default:0;not null" json:"discount_price"`

	User    users.User       `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"  json:"-"`
	Product products.Product `gorm:"foreignKey:ProductID;references:ID;constraint:OnDelete:RESTRICT" json:"product"`

	Logs []Log `gorm:"foreignKey:StockID;references:ID" json:"logs,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// HasDiscount retorna true si hay un precio de descuento activo
func (s *Stock) HasDiscount() bool {
	return s.DiscountPrice > 0 && s.DiscountPrice < s.Price
}

// EffectivePrice retorna el precio que el comprador paga
func (s *Stock) EffectivePrice() float64 {
	if s.HasDiscount() {
		return s.DiscountPrice
	}
	return s.Price
}

func (s *Stock) BeforeUpdate(tx *gorm.DB) error {
	// Partial update (Model().Where().Update) — modelo sin ID, nada que comparar.
	if s.ID == 0 {
		return nil
	}

	var old Stock
	if err := tx.First(&old, s.ID).Error; err != nil {
		return err
	}

	var logs []Log

	if old.Price != s.Price {
		logs = append(logs, Log{
			StockID:       s.ID,
			LogType:       LogPriceChange,
			PreviousPrice: old.Price,
			NewPrice:      s.Price,
		})
	}

	if old.DiscountPrice != s.DiscountPrice {
		logs = append(logs, Log{
			StockID:          s.ID,
			LogType:          LogDiscountChange,
			PreviousDiscount: old.DiscountPrice,
			NewDiscount:      s.DiscountPrice,
		})
	}

	if len(logs) == 0 {
		return nil
	}

	return tx.Create(&logs).Error
}

type LogType string

const (
	// Físicos
	LogUnboxing   LogType = "unboxing"   // apertura de caja
	LogAdd        LogType = "add"        // creación inicial
	LogRestock    LogType = "restock"    // entrada de stock
	LogSale       LogType = "sale"       // venta
	LogTrade      LogType = "trade"      // intercambio
	LogReturn     LogType = "return"     // devolución
	LogGift       LogType = "gift"       // donación
	LogLost       LogType = "lost"       // pérdida
	LogDamage     LogType = "damage"     // carta dañada
	LogAdjustment LogType = "adjustment" // ajuste manual
	LogRollback   LogType = "rollback"   // rollback a estado anterior

	// Precio
	LogPriceChange    LogType = "price_change"    // cambio en Price
	LogDiscountChange LogType = "discount_change" // cambio en DiscountPrice
)

type Log struct {
	ID          uint64  `gorm:"primaryKey;autoIncrement"                        json:"id"`
	StockID     uint64  `gorm:"not null;index:idx_stock_type,priority:1"          json:"stock_id"`
	ParentLogID *uint64 `gorm:"index"                                             json:"parent_log_id,omitempty"`
	LogType     LogType `gorm:"not null;size:30;index:idx_stock_type,priority:2"  json:"log_type"`

	// Cantidad
	Delta         int `gorm:"default:0" json:"delta"`
	PreviousStock int `gorm:"default:0" json:"previous_stock"`
	NewStock      int `gorm:"default:0" json:"new_stock"`

	// Precio
	PreviousPrice    float64 `gorm:"type:numeric(10,2);default:0" json:"previous_price"`
	NewPrice         float64 `gorm:"type:numeric(10,2);default:0" json:"new_price"`
	PreviousDiscount float64 `gorm:"type:numeric(10,2);default:0" json:"previous_discount"`
	NewDiscount      float64 `gorm:"type:numeric(10,2);default:0" json:"new_discount"`

	Note string `gorm:"type:text" json:"note,omitempty"`

	Stock     Stock `gorm:"foreignKey:StockID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
	ParentLog *Log  `gorm:"foreignKey:ParentLogID;references:ID"                            json:"parent_log,omitempty"`
	Children  []Log `gorm:"foreignKey:ParentLogID;references:ID"                            json:"children,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}
