package cards

import (
	"fmt"
	"time"
)

// CardInfo almacena los datos invariantes de una carta:
// aquellos que no cambian sin importar el idioma (tipo, subtypes, tags, imágenes, etc.).
// Una misma CardInfo puede tener múltiples Cards asociadas, una por cada idioma disponible.
type CardInfo struct {
	ID        string            `json:"id"         gorm:"primaryKey;size:50"`
	Type      string            `json:"type"       gorm:"not null;index"`
	Subtypes  map[string]string `json:"subtypes"   gorm:"type:jsonb;serializer:json;default:'{}'"`
	Tags      map[string]string `json:"tags"       gorm:"type:jsonb;serializer:json;default:'{}'"`
	Archetype string            `json:"archetype"  gorm:"index"`
	Source    string            `json:"source"`
	Images    []CardImage       `json:"images"     gorm:"type:jsonb;serializer:json;default:'[]'"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// HasMany: una CardInfo tiene muchos Cards (uno por idioma)
	Cards []Card `json:"cards,omitempty" gorm:"foreignKey:CardInfoID"`
}

// Card almacena el contenido localizado de una carta para un idioma específico.
// Cada fila es una traducción: misma carta base, distinto idioma.
// Se relaciona con CardInfo a través de CardInfoID (clave foránea).
type Card struct {
	ID          string   `json:"id"           gorm:"primaryKey;size:50"`
	CardInfoID  string   `json:"card_info_id" gorm:"not null;index"`
	Lang        LangCode `json:"lang"         gorm:"size:10;not null;index"`
	Name        string   `json:"name"         gorm:"not null;index"`
	Description string   `json:"description"`

	CreatedAt time.Time `json:"created_at"   gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at"   gorm:"autoUpdateTime"`

	// BelongsTo: el struct CardInfo completo precargado (via Preload o Join)
	Info CardInfo `json:"info,omitempty" gorm:"foreignKey:CardInfoID"`
}

// CardImage almacena las URLs de los distintos tamaños de imagen de una carta.
type CardImage struct {
	URL        string `json:"url"`
	URLSmall   string `json:"url_small"`
	URLCropped string `json:"url_cropped"`
}

//	GenerateCardId("scg", "12345", "en") // "scg-12345-en"
//	GenerateCardId("moxfield", "SKU001", "es") // "moxfield-SKU001-es"
func GenerateCardId(provider, code string, lang LangCode) string {
	return fmt.Sprintf("%s-%s-%s", provider, code, lang)
}

//	GenerateCardInfoId("scg", "12345") // "scg-12345"
//	GenerateCardInfoId("moxfield", "SKU001") // "moxfield-SKU001"
func GenerateCardInfoId(provider, code string) string {
	return fmt.Sprintf("%s-%s", provider, code)
}
