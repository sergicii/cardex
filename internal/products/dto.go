package products

type SuggestionDTO struct {
	ID               uint64      `json:"id"`
	ExternalID       string      `json:"external_id"`
	SetExternalID    string      `json:"set_external_id"`
	Type             ProductType `json:"type"`
	TCG              TCG         `json:"tcg"`
	Wanted           uint        `json:"wanted"`
	Name             string      `json:"name"`
	Code             string      `json:"code"`
	Rarity           string      `json:"rarity"`
	RarityCode       string      `json:"rarity_code"`
	SetName          string      `json:"set_name"`
	SetCode          string      `json:"set_code"`
	SetRegionCode    string      `json:"set_region_code"`
	Lang             string      `json:"lang"`
	Language         LangCode    `json:"language"`
	Image            string      `json:"image"`
	Edition          string      `json:"edition"`
	CopiesInWishlist uint        `json:"copies_in_wishlist"`
	Stock            int         `json:"stock"`
}

type RelatedCardDTO struct {
	ID            uint64      `json:"id"`
	ExternalID    string      `json:"external_id"`
	SetExternalID string      `json:"set_external_id"`
	Type          ProductType `json:"type"`
	TCG           TCG         `json:"tcg"`
	Wanted        uint        `json:"wanted"`
	Name          string      `json:"name"`
	Code          string      `json:"code,omitempty"`
	Rarity        string      `json:"rarity,omitempty"`
	RarityCode    string      `json:"rarity_code,omitempty"`
	SetName       string      `json:"set_name,omitempty"`
	SetCode       string      `json:"set_code,omitempty"`
	Lang          string      `json:"lang,omitempty"`
	Image         string      `json:"image,omitempty"`
	Edition       string      `json:"edition,omitempty"`
}

type RelatedCardsResponse struct {
	SameLangDifferentRarity []RelatedCardDTO `json:"same_lang_different_rarity"`
	DifferentLang           []RelatedCardDTO `json:"different_lang"`
}

type RelatedCardsInput struct {
	ID            uint64 `json:"id"`
	ExternalID    string `json:"external_id"`
	SetExternalID string `json:"set_external_id"`
	TCG           TCG    `json:"tcg"`
	Lang          string `json:"lang"`
}

type GetCardsBySetInput struct {
	SetExternalID string `json:"set_external_id"`
	Lang          string `json:"lang"`
}
