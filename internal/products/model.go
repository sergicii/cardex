package products

import (
	"strings"
	"time"
)

type (
	ProductType string
	TCG         string
	LangCode    string
)

const (
	ProductTypeSet  ProductType = "set"
	ProductTypeCard ProductType = "card"
)

const (
	YGO TCG = "ygo"
)

const (
	EN LangCode = "EN" // English
	FR LangCode = "FR" // French
	DE LangCode = "DE" // German
	IT LangCode = "IT" // Italian
	PT LangCode = "PT" // Portuguese
	SP LangCode = "SP" // Spanish

	JP LangCode = "JP" // Japanese
	AE LangCode = "AE" // Asian-English
	KR LangCode = "KR" // Korean
	TC LangCode = "TC" // Traditional Chinese
	SC LangCode = "SC" // Simplified Chinese
)

// Product representa una carta impresa (print) o un set dentro de un TCG.
//
// Identidad única de un print:  ExternalID + SetExternalID + TCG + Code + Lang + Rarity + Edition
// Identidad única de un set:    ExternalID + SetExternalID + TCG + Code + Lang + Rarity + Edition
// (mismo constraint unificado, campos vacíos actúan como discriminador)
type Product struct {
	ID   uint64      `json:"id"    gorm:"primaryKey;autoIncrement"`
	Type ProductType `json:"type"  gorm:"not null"`

	ExternalID    string   `json:"external_id"    gorm:"uniqueIndex:idx_product_identity,priority:1"`
	SetExternalID string   `json:"set_external_id" gorm:"uniqueIndex:idx_product_identity,priority:2"`
	TCG           TCG      `json:"tcg"             gorm:"size:20;not null;uniqueIndex:idx_product_identity,priority:3"`
	Code          string   `json:"code,omitempty"  gorm:"uniqueIndex:idx_product_identity,priority:4"`
	Lang          LangCode `json:"lang"            gorm:"size:2;not null;uniqueIndex:idx_product_identity,priority:5"`
	Rarity        string   `json:"rarity,omitempty" gorm:"size:60;uniqueIndex:idx_product_identity,priority:6"`

	Name       string `json:"name"               gorm:"not null;index"`            // búsqueda por nombre
	SetName    string `json:"set_name"            gorm:"not null;index"`           // búsqueda por set
	SetCode    string `json:"set_code,omitempty"  gorm:"index"`                    // búsqueda por code
	RarityCode string `json:"rarity_code,omitempty" gorm:"index"`                  // búsqueda por rareza
	Archetype  string `json:"archetype,omitempty" gorm:"index"`                    // filtro de arquetipo
	Wanted     uint   `json:"wanted"              gorm:"default:0;not null;index"` // feed principal

	Description string      `json:"description,omitempty"`
	CardTypes   string      `json:"tags,omitempty"`
	Images      []CardImage `json:"images,omitempty"       gorm:"type:jsonb;serializer:json;default:'[]'"`
	SerieCode   string      `json:"serie_code,omitempty"`

	Edition        string `json:"edition,omitempty" gorm:"uniqueIndex:idx_product_identity,priority:7"`
	PrintURLSmall  string `json:"print_url_small,omitempty"`
	PrintURLLarge  string `json:"print_url_large,omitempty"`
	QuantityPerSet uint   `json:"quantity_per_set" gorm:"default:0"`

	SetRegionCode string `json:"set_region_code,omitempty"`
	SetType       string `json:"set_type,omitempty"`
	SetImageSmall string `json:"set_image_small,omitempty"`
	SetImageLarge string `json:"set_image_large,omitempty"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type CardImage struct {
	URL        string `json:"image_url"`
	URLSmall   string `json:"image_url_small"`
	URLCropped string `json:"image_url_cropped"`
}

type ProductUniqueKey string

// UniqueKey devuelve la clave compuesta que identifica de forma única un Product,
// espejando el índice idx_product_identity de la base de datos.
// Ejemplo: Ojama Yellow|Chazz Princeton|YGO|CHPR-EN01|EN|Common
func (p Product) UniqueKey() ProductUniqueKey {
	return ProductUniqueKey(strings.Join([]string{
		p.ExternalID,
		p.SetExternalID,
		string(p.TCG),
		p.Code,
		string(p.Lang),
		p.Rarity,
		p.Edition,
	}, "|"))
}
