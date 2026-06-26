package providers

import (
	"time"

	"github.com/operaodev/cardex/internal/products"
)

const (
	scrapeBatchSize      = 300
	scrapeBatchPause     = 3 * time.Second
	scrapeParallelism    = 4
	scrapeDelay          = 1200 * time.Millisecond
	scrapeRequestTimeout = 60 * time.Second
	httpClientTimeout    = 60 * time.Second
	progressLogInterval  = 100
)

type YGOCard struct {
	ExternalID     string
	ID             uint                 `json:"id"`
	Name           string               `json:"name"`
	Types          string               `json:"humanReadableCardType"`
	Description    string               `json:"desc"`
	Archetype      string               `json:"archetype"`
	Images         []products.CardImage `json:"card_images"`
	Lang           products.LangCode    `json:"lang"`
	QuantityPerSet int                  `json:"quantity_per_set"`
}

type YGOSet struct {
	SetExternalID string
	Lang          products.LangCode
	SetName       string
	SetCode       string
	SetRegionCode string
	Description   string
	SetType       string
}
