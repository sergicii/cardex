package custompacks

import (
	"time"

	"github.com/operaodev/cardex/internal/products"
	"github.com/operaodev/cardex/internal/stock"
	"github.com/operaodev/cardex/internal/users"
)

type Wishlist struct {
	ID        uint64 `json:"id"         gorm:"primaryKey;autoIncrement"`
	UserID    string `json:"user_id"    gorm:"not null;type:uuid;uniqueIndex:idx_wishlist_user_product,priority:1"`
	ProductID uint64 `json:"product_id" gorm:"not null;uniqueIndex:idx_wishlist_user_product,priority:2"`
	Quantity  int    `json:"quantity"   gorm:"default:1;not null"`

	User    users.User       `json:"-"       gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	Product products.Product `json:"product"  gorm:"foreignKey:ProductID;references:ID;constraint:OnDelete:RESTRICT"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type Bundle struct {
	ID     uint64 `json:"id"         gorm:"primaryKey;autoIncrement"`
	UserID string `json:"user_id"    gorm:"not null;type:uuid"`

	User  users.User   `json:"-"       gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	Items []BundleItem `json:"items"     gorm:"foreignKey:BundleID;references:ID;constraint:OnDelete:CASCADE"`
}

type BundleItemType string

const (
	BundleItemTypeGift BundleItemType = "gift"
	BundleItemTypeSale BundleItemType = "sale"
)

type BundleItem struct {
	ID       uint64         `json:"id"         gorm:"primaryKey;autoIncrement"`
	BundleID uint64         `json:"bundle_id"  gorm:"not null;"`
	StockID  uint64         `json:"stock_id"   gorm:"not null;"`
	Quantity int            `json:"quantity"   gorm:"default:1;not null"`
	Type     BundleItemType `json:"type"     gorm:"not null;"`

	Bundle Bundle      `json:"-"       gorm:"foreignKey:BundleID;references:ID;constraint:OnDelete:CASCADE"`
	Stock  stock.Stock `json:"-"       gorm:"foreignKey:StockID;references:ID;constraint:OnDelete:CASCADE"`
}
