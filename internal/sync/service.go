package sync

import (
	"fmt"
	"log"
	"strings"

	"github.com/operaodev/cardex/internal/cards"
	"github.com/operaodev/cardex/internal/search"
)

// SyncService orquesta la sincronización de cartas externas hacia la DB local.
type SyncService struct {
	searchSvc *search.Service
	cardsRepo cards.Repository
}

func NewSyncService(searchSvc *search.Service, cardsRepo cards.Repository) *SyncService {
	return &SyncService{
		searchSvc: searchSvc,
		cardsRepo: cardsRepo,
	}
}

// SyncAll obtiene todas las cartas del proveedor indicado y las persiste en la DB.
// Solo inserta cartas que tengan impresiones físicas (PrintedCards).
// Devuelve el número de cartas procesadas (nuevas + actualizadas) y un error si falla.
func (s *SyncService) SyncAll(tcg string) (int, error) {
	log.Printf("[sync] Iniciando sincronización completa para TCG=%s", tcg)

	results, err := s.searchSvc.SearchAll(tcg)
	if err != nil {
		return 0, fmt.Errorf("error obteniendo cartas del proveedor: %w", err)
	}

	if len(results) == 0 {
		log.Printf("[sync] No se encontraron cartas para TCG=%s", tcg)
		return 0, nil
	}

	log.Printf("[sync] %d cartas encontradas. Mapeando y persistiendo...", len(results))

	toUpsert := mapResultsToCards(results)

	if len(toUpsert) == 0 {
		log.Printf("[sync] Ninguna carta tiene impresiones físicas para TCG=%s", tcg)
		return 0, nil
	}

	upserted, err := s.cardsRepo.Upsert(toUpsert)
	if err != nil {
		return 0, fmt.Errorf("error persistiendo cartas en la DB: %w", err)
	}

	log.Printf("[sync] Sincronización completada: %d cartas procesadas", upserted)
	return upserted, nil
}

// mapResultsToCards convierte un slice de search.ResultCard al modelo flat de cards.Card.
// Solo genera filas para resultados que tengan impresiones físicas (PrintedCards).
// Genera una fila por cada PrintedCard, usando el idioma del print como clave de nombre/descripción.
// Deduplica los registros en base a su clave única para evitar errores ON CONFLICT en la base de datos.
func mapResultsToCards(results []search.ResultCard) []cards.Card {
	var out []cards.Card
	seen := make(map[string]bool)

	for _, r := range results {
		if r.ExternalID == "" {
			continue
		}

		// Ignorar cartas sin impresiones físicas
		if len(r.PrintedCards) == 0 {
			continue
		}

		englishName := r.Names[cards.EN]

		for _, p := range r.PrintedCards {
			lang := p.Lang
			if lang == "" {
				lang = cards.EN
			}

			localName := r.Names[lang]
			if localName == "" {
				localName = englishName
			}

			description := r.Descriptions[lang]
			if description == "" {
				description = r.Descriptions[cards.EN]
			}

			tcg := p.TCG
			if tcg == "" {
				tcg = r.TCG
			}

			// La identidad única es: ExternalID + Code + Lang + Rarity + SetEnglishName (SetName)
			key := fmt.Sprintf("%s|%s|%s|%s|%s", r.ExternalID, p.Code, lang, p.Rarity, p.SetName)
			if seen[key] {
				continue
			}
			seen[key] = true

			card := cards.Card{
				ExternalID:  r.ExternalID,
				Code:        p.Code,
				TCG:         tcg,
				Name:        localName,
				EnglishName: englishName,
				Description: description,
				Lang:        lang,
				Type:        r.Type,
				Subtypes:    r.Subtypes,
				Archetype:   r.Archetype,
				Sources:     r.Sources,
				CardImages:  r.Images,
				SetName:     p.SetName,
				SetEnglishName: p.SetName,
				SetCode:     strings.Split(p.SetName, "-")[0],
				Rarity:      p.Rarity,
			}

			out = append(out, card)
		}
	}

	return out
}