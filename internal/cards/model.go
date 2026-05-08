package cards

import (
	"time"
)

type Print string
type Rarity string
type TCG string

const (
	TCGMagic   TCG = "mtg"
	TCGYugioh  TCG = "ygo"
	TCGPokemon TCG = "pkm"
)

type Set struct {
	ID   uint64 `json:"id" gorm:"primaryKey;autoIncrement"`
	Code string `json:"code,omitempty" gorm:"index"`
	Name string `json:"name" gorm:"not null;index"`
	TCG  TCG    `json:"tcg" gorm:"size:20;not null;index"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// Code: "RAO5-EN001"
// Rarity: "common"
// SetName: "Raging Tempest"
// SetCode: "RAO5"
// SetSharedCode: "RAO5-001"
// Lang: "en"
// TCG: "ygo"
// Print: "reprint" or "new_artwork"
type PrintedCard struct {
	ID     uint64 `json:"id" gorm:"primaryKey;autoIncrement"`
	Wanted uint   `json:"wanted" gorm:"default:0;not null;index"`
	Code   string `json:"code,omitempty" gorm:"index"`
	Rarity Rarity `json:"rarity,omitempty" gorm:"size:20;index"`

	SetName       string `json:"set" gorm:"not null;index"`
	SetCode       string `json:"set_code,omitempty" gorm:"index"`
	SetSharedCode string `json:"set_shared_code,omitempty" gorm:"index"`

	Lang LangCode `json:"lang" gorm:"size:2;not null;index"`
	TCG  TCG      `json:"tcg" gorm:"size:20;not null;index"`

	ReferenceImage string `json:"reference_image,omitempty" gorm:"-"`
	Print          Print  `json:"print" gorm:"-"`

	CardID uint64 `json:"card_id" gorm:"index;not null"`
	Card   Card   `gorm:"foreignKey:CardID"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// Names: { "en": "Blue-Eyes White Dragon", "pt": "Dragão Branco de Olhos Azuis" }
// Descriptions: { "en": "", "pt": "" }
// Sources: ["yugipedia:url", "ygoprodeck:url"]
type Card struct {
	ID           uint64              `json:"id" gorm:"primaryKey;autoIncrement"`
	ExternalID   string              `json:"external_id" gorm:"not null;index"`
	TCG          TCG                 `json:"tcg" gorm:"size:20;not null;index"`
	Names        map[LangCode]string `json:"names" gorm:"type:jsonb;serializer:json;not null;index:,type:gin"`
	Descriptions map[LangCode]string `json:"descriptions,omitempty" gorm:"type:jsonb;serializer:json;default:'{}'"`
	Type         string              `json:"type" gorm:"not null;index"`
	Subtypes     []string            `json:"subtypes,omitempty" gorm:"type:jsonb;serializer:json;default:'[]';index:,type:gin"`
	Archetype    string              `json:"archetype,omitempty" gorm:"index"`
	Sources      []string            `json:"sources,omitempty" gorm:"type:jsonb;serializer:json;default:'[]'"`
	Images       []CardImage         `json:"images,omitempty" gorm:"type:jsonb;serializer:json;default:'[]'"`

	PrintedCards []PrintedCard `json:"printed_cards,omitempty" gorm:"foreignKey:CardID"`
	MatchedLang  LangCode      `json:"matched_lang,omitempty" gorm:"-"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type CardImage struct {
	URL        string `json:"image_url"`
	URLSmall   string `json:"image_url_small"`
	URLCropped string `json:"image_url_cropped"`
}
