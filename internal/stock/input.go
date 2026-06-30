package stock

import "github.com/operaodev/cardex/internal/products"

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

type FilterInput struct {
	Input     string `json:"input,omitempty"` //name | code | set_region_code
	Type      string `json:"type,omitempty"`
	TCG       string `json:"tcg,omitempty"`
	Lang      string `json:"lang,omitempty"`
	SetName   string `json:"set_name,omitempty"`
	Archetype string `json:"archetype,omitempty"`
	Rarity    string `json:"rarity,omitempty"`
	Edition   string `json:"edition,omitempty"`
	Page      int    `json:"page,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

type FilterOutput struct {
	Type      []products.ProductType `json:"type,omitempty"`
	TCG       []products.TCG         `json:"tcg,omitempty"`
	Lang      []products.LangCode    `json:"lang,omitempty"`
	SetName   []string               `json:"set_name,omitempty"`
	Archetype []string               `json:"archetype,omitempty"`
	Rarity    []string               `json:"rarity,omitempty"`
	Edition   []string               `json:"edition,omitempty"`
}

type StockPage struct {
	Items      []Stock `json:"items"`
	Total      int64   `json:"total"`
	Page       int     `json:"page"`
	Limit      int     `json:"limit"`
	TotalPages int     `json:"total_pages"`
}
