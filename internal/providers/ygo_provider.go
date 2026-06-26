package providers

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gocolly/colly"
	"github.com/operaodev/cardex/internal/products"
)

// UniqueKey devuelve la clave de mapa para un YGOCard: "ExternalID-Lang"
func (y YGOCard) UniqueKey() string {
	return fmt.Sprintf("%s-%s", y.ExternalID, y.Lang)
}

type YGOProvider struct {
	httpClient       *http.Client
	ygoproBaseUrl    string
	yugipediaBaseUrl string
}

func NewYGOProvider() *YGOProvider {
	return &YGOProvider{
		httpClient:       &http.Client{Timeout: httpClientTimeout},
		ygoproBaseUrl:    "https://db.ygoprodeck.com/api/v7",
		yugipediaBaseUrl: "https://yugipedia.com",
	}
}

func (y *YGOProvider) FetchItems() ([]products.Product, error) {
	englishCards, err := y.fetchCardsYGOPro("")
	if err != nil {
		return nil, err
	}
	return y.fetchAll(englishCards)
}

func (y *YGOProvider) FetchItemsByName(name string) ([]products.Product, error) {
	englishCards, err := y.fetchCardsYGOPro(name)
	if err != nil {
		return nil, err
	}
	return y.fetchAll(englishCards)
}

// wikiPagePath convierte un nombre de carta en un segmento de ruta URL compatible con MediaWiki.
func wikiPagePath(name string) string {
	p := strings.ReplaceAll(name, " ", "_")
	p = strings.ReplaceAll(p, "?", "%3F")
	p = strings.ReplaceAll(p, "#", "%23")
	return p
}

func createColly() *colly.Collector {
	c := colly.NewCollector(
		colly.Async(true),
		colly.UserAgent("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"),
	)
	c.SetRequestTimeout(scrapeRequestTimeout)
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: scrapeParallelism,
		Delay:       scrapeDelay,
	})

	setBrowserHeaders := func(r *colly.Request) {
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.9")
		r.Headers.Set("Cache-Control", "max-age=0")
		r.Headers.Set("Sec-Ch-Ua", `"Not/A)Brand";v="8", "Chromium";v="125", "Google Chrome";v="125"`)
		r.Headers.Set("Sec-Ch-Ua-Mobile", "?0")
		r.Headers.Set("Sec-Ch-Ua-Platform", `"Linux"`)
		r.Headers.Set("Sec-Fetch-Dest", "document")
		r.Headers.Set("Sec-Fetch-Mode", "navigate")
		r.Headers.Set("Sec-Fetch-Site", "none")
		r.Headers.Set("Sec-Fetch-User", "?1")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")
	}
	c.OnRequest(setBrowserHeaders)
	return c
}

// fetchAll ejecuta el pipeline unificado de scraping.
func (y *YGOProvider) fetchAll(englishCards []YGOCard) ([]products.Product, error) {
	log.Printf("[fetchAll] iniciando pipeline para %d carta(s)", len(englishCards))

	enCardByName := make(map[string]YGOCard, len(englishCards))
	for _, card := range englishCards {
		enCardByName[card.Name] = card
	}

	c := createColly()

	var (
		mu               sync.Mutex
		allItems         []products.Product
		galleryURLs      = make(map[string]string) // ExternalID → relative gallery URL
		setURLs          = make(map[string]string) // wikiPageName → full wiki URL
		setPageToExtID   = make(map[string]string) // wikiPageName → SetExternalID
		failedCards      []YGOCard
		failedCardsRetry []YGOCard
		collectedSets    = make(map[string]*SetInfo)
		processedCount   atomic.Int64
		cfBlockCount     atomic.Int64
	)

	// OnError handler
	c.OnError(func(r *colly.Response, err error) {
		cardName := r.Request.Ctx.Get("name")
		phase := r.Request.Ctx.Get("phase")
		if r != nil && (r.StatusCode == http.StatusForbidden || r.StatusCode == http.StatusServiceUnavailable) {
			blocks := cfBlockCount.Add(1)
			log.Printf("[fetchAll] BLOQUEO CLOUDFLARE #%d fase %s para %q (%s): estado %d",
				blocks, phase, cardName, r.Request.URL, r.StatusCode)
		} else {
			log.Printf("[fetchAll] error HTTP fase %s para %q (%s): %v",
				phase, cardName, r.Request.URL, err)

			if cardName != "" && (phase == "card" || phase == "card-retry") {
				mu.Lock()
				if card, ok := enCardByName[cardName]; ok {
					if phase == "card" {
						failedCards = append(failedCards, card)
					} else {
						failedCardsRetry = append(failedCardsRetry, card)
					}
				}
				mu.Unlock()
			}
		}
	})

	// OnHTML dispatcher
	c.OnHTML("body", func(e *colly.HTMLElement) {
		phase := e.Request.Ctx.Get("phase")
		switch phase {
		case "card", "card-retry", "card-retry-suffix":
			y.handleCardPage(e, &mu, enCardByName, &allItems, galleryURLs, setURLs, setPageToExtID, &failedCards, &failedCardsRetry, &processedCount)
		case "gallery":
			y.handleGalleryPage(e, &mu, &allItems, &processedCount)
		case "set":
			y.handleSetPage(e, &mu, &allItems, collectedSets, setPageToExtID, &processedCount, c)
		case "set-list":
			y.handleSetListPage(e, &mu, &allItems)
		}
	})

	// Phase 1: Scrape cards by ID
	log.Printf("[fetchAll] fase 1: scrapeo por ID para %d carta(s)", len(englishCards))
	for i := 0; i < len(englishCards); i += scrapeBatchSize {
		end := min(i+scrapeBatchSize, len(englishCards))

		for _, card := range englishCards[i:end] {
			ctx := colly.NewContext()
			ctx.Put("phase", "card")
			ctx.Put("name", card.Name)
			ctx.Put("id", fmt.Sprintf("%d", card.ID))
			wikiURL := fmt.Sprintf("%s/wiki/%s", y.yugipediaBaseUrl, fmt.Sprintf("%08d", card.ID))
			_ = c.Request("GET", wikiURL, nil, ctx, nil)
		}

		c.Wait()

		if end < len(englishCards) {
			log.Printf("[fetchAll] fase 1 lote: %d/%d cartas, pausando %v", end, len(englishCards), scrapeBatchPause)
			time.Sleep(scrapeBatchPause)
		}
	}

	// Phase 2a: Retry failed cards by simple name
	if len(failedCards) > 0 {
		log.Printf("[fetchAll] fase 2a: reintento por nombre para %d carta(s)", len(failedCards))
		for i := 0; i < len(failedCards); i += scrapeBatchSize {
			end := min(i+scrapeBatchSize, len(failedCards))

			for _, card := range failedCards[i:end] {
				ctx := colly.NewContext()
				ctx.Put("phase", "card-retry")
				ctx.Put("name", card.Name)
				ctx.Put("id", fmt.Sprintf("%d", card.ID))
				wikiURL := fmt.Sprintf("%s/wiki/%s", y.yugipediaBaseUrl, url.PathEscape(card.Name))
				_ = c.Request("GET", wikiURL, nil, ctx, nil)
				log.Println(wikiURL)
			}

			c.Wait()

			if end < len(failedCards) {
				log.Printf("[fetchAll] fase 2a lote: %d/%d cartas, pausando %v", end, len(failedCards), scrapeBatchPause)
				time.Sleep(scrapeBatchPause)
			}
		}
	}

	// Phase 2b: Retry with _(card) suffix
	if len(failedCardsRetry) > 0 {
		log.Printf("[fetchAll] fase 2b: reintento con sufijo _(card) para %d carta(s)", len(failedCardsRetry))
		for i := 0; i < len(failedCardsRetry); i += scrapeBatchSize {
			end := min(i+scrapeBatchSize, len(failedCardsRetry))

			for _, card := range failedCardsRetry[i:end] {
				ctx := colly.NewContext()
				ctx.Put("phase", "card-retry-suffix")
				ctx.Put("name", card.Name)
				ctx.Put("id", fmt.Sprintf("%d", card.ID))
				wikiURL := fmt.Sprintf("%s/wiki/%s_(card)", y.yugipediaBaseUrl, url.PathEscape(card.Name))
				_ = c.Request("GET", wikiURL, nil, ctx, nil)
			}

			c.Wait()

			if end < len(failedCardsRetry) {
				log.Printf("[fetchAll] fase 2b lote: %d/%d cartas, pausando %v", end, len(failedCardsRetry), scrapeBatchPause)
				time.Sleep(scrapeBatchPause)
			}
		}
	}

	// Pause before galleries
	time.Sleep(scrapeBatchPause)

	// Phase 3: Scrape galleries
	if len(galleryURLs) > 0 {
		log.Printf("[fetchAll] fase 3: scrapeo de galerías para %d carta(s)", len(galleryURLs))
		galleryList := make([]struct{ externalID, galleryURL string }, 0, len(galleryURLs))
		for extID, gURL := range galleryURLs {
			galleryList = append(galleryList, struct{ externalID, galleryURL string }{extID, gURL})
		}

		for i := 0; i < len(galleryList); i += scrapeBatchSize {
			end := min(i+scrapeBatchSize, len(galleryList))

			for _, g := range galleryList[i:end] {
				ctx := colly.NewContext()
				ctx.Put("phase", "gallery")
				ctx.Put("externalID", g.externalID)
				galleryURL := fmt.Sprintf("%s%s", y.yugipediaBaseUrl, g.galleryURL)
				_ = c.Request("GET", galleryURL, nil, ctx, nil)
			}

			c.Wait()

			if end < len(galleryList) {
				log.Printf("[fetchAll] fase 3 lote: %d/%d galerías, pausando %v", end, len(galleryList), scrapeBatchPause)
				time.Sleep(scrapeBatchPause)
			}
		}
	}

	// Pause before sets
	time.Sleep(scrapeBatchPause)

	// Phase 4: Scrape sets
	if len(setURLs) > 0 {
		log.Printf("[fetchAll] fase 4: scrapeo de sets para %d set(s)", len(setURLs))
		setList := make([]struct{ pageName, wikiURL string }, 0, len(setURLs))
		for pageName, wikiURL := range setURLs {
			setList = append(setList, struct{ pageName, wikiURL string }{pageName, wikiURL})
		}

		for i := 0; i < len(setList); i += scrapeBatchSize {
			end := min(i+scrapeBatchSize, len(setList))

			for _, s := range setList[i:end] {
				ctx := colly.NewContext()
				ctx.Put("phase", "set")
				ctx.Put("pageName", s.pageName)
				_ = c.Request("GET", s.wikiURL, nil, ctx, nil)
			}

			c.Wait()

			if end < len(setList) {
				log.Printf("[fetchAll] fase 4 lote: %d/%d sets, pausando %v", end, len(setList), scrapeBatchPause)
				time.Sleep(scrapeBatchPause)
			}
		}
	}

	// Create set products
	setProducts := setInfoToProducts(collectedSets)
	allItems = append(allItems, setProducts...)

	// Deduplicate
	allItems = deduplicateProducts(allItems)

	// Normalize strings
	normalizeProducts(allItems)

	// Forzar quantity=0 en set products
	for i := range allItems {
		if allItems[i].Type == products.ProductTypeSet {
			allItems[i].QuantityPerSet = 0
		}
	}

	log.Printf("[fetchAll] terminado — %d items totales (%d cartas + %d sets)", len(allItems), len(allItems)-len(setProducts), len(setProducts))
	return allItems, nil
}

func (y *YGOProvider) handleCardPage(e *colly.HTMLElement, mu *sync.Mutex, enCardByName map[string]YGOCard, allItems *[]products.Product, galleryURLs map[string]string, setURLs map[string]string, setPageToExtID map[string]string, failedCards *[]YGOCard, failedCardsRetry *[]YGOCard, processedCount *atomic.Int64) {
	cardName := e.Request.Ctx.Get("name")
	phase := e.Request.Ctx.Get("phase")

	count := processedCount.Add(1)
	if count%progressLogInterval == 0 {
		log.Printf("[fetchAll] %s progreso: %d cartas procesadas, %d items hasta ahora", phase, count, len(*allItems))
	}

	mu.Lock()
	enCard, ok := enCardByName[cardName]
	mu.Unlock()
	if !ok {
		log.Printf("[fetchAll] carta %q no encontrada en el mapa (posible redirección)", cardName)
		return
	}

	translatedCards := parseCards(e, enCard)
	translatedCards[enCard.UniqueKey()] = enCard

	// Extract set URLs from CTS tables
	printEntries := parseCTSTables(e)
	for _, entry := range printEntries {
		if entry.SetURL != "" {
			mu.Lock()
			setURLs[entry.SetURL] = fmt.Sprintf("%s/wiki/%s", y.yugipediaBaseUrl, url.PathEscape(entry.SetURL))
			setPageToExtID[entry.SetURL] = entry.SetExternalID
			mu.Unlock()
		}
	}

	localItems := mapPrints(e, enCard, translatedCards)

	// Extract gallery URL
	if gURL := parseGalleryLink(e); gURL != "" {
		mu.Lock()
		galleryURLs[enCard.ExternalID] = gURL
		mu.Unlock()
	}

	mu.Lock()
	if len(localItems) > 0 {
		*allItems = append(*allItems, localItems...)
	} else {
		switch phase {
		case "card":
			*failedCards = append(*failedCards, enCard)
		case "card-retry":
			*failedCardsRetry = append(*failedCardsRetry, enCard)
		case "card-retry-suffix":
			log.Printf("[fetchAll] carta %q no encontrada después de todos los intentos", cardName)
		}
	}
	mu.Unlock()
}

func (y *YGOProvider) handleGalleryPage(e *colly.HTMLElement, mu *sync.Mutex, allItems *[]products.Product, processedCount *atomic.Int64) {
	count := processedCount.Add(1)
	if count%progressLogInterval == 0 {
		log.Printf("[fetchAll] gallery progreso: %d galerías procesadas", count)
	}

	entries := parseGallery(e)

	mu.Lock()
	defer mu.Unlock()

	// Build item index
	itemIndex := make(map[string]*products.Product, len(*allItems))
	for i := range *allItems {
		if (*allItems)[i].Type == products.ProductTypeCard {
			key := fmt.Sprintf("%s|%s|%s|%s|%s", (*allItems)[i].SetExternalID, (*allItems)[i].Code, (*allItems)[i].Lang, (*allItems)[i].Rarity, (*allItems)[i].Edition)
			itemIndex[key] = &(*allItems)[i]
		}
	}

	for _, entry := range entries {
		key := fmt.Sprintf("%s|%s|%s|%s|%s", entry.Set, entry.Code, entry.Lang, entry.Rarity, entry.Edition)
		item, ok := itemIndex[key]
		if !ok {
			fallbackKey := fmt.Sprintf("%s|%s|%s|%s|", entry.Set, entry.Code, entry.Lang, entry.Rarity)
			item, ok = itemIndex[fallbackKey]
		}
		if ok {
			item.PrintURLSmall = entry.PrintURLSmall
			item.PrintURLLarge = entry.PrintURLLarge
			item.Edition = entry.Edition
			item.RarityCode = entry.RarityCode
		}
	}
}

func hasRarityMatch(itemRarity string, entryRarities []string) bool {
	if len(entryRarities) == 0 {
		return true
	}
	normItem := strings.ToLower(strings.TrimSpace(itemRarity))
	for _, er := range entryRarities {
		normEntry := strings.ToLower(strings.TrimSpace(er))
		if normItem == normEntry {
			return true
		}
		if normItem != "" && normEntry != "" && (strings.Contains(normItem, normEntry) || strings.Contains(normEntry, normItem)) {
			return true
		}
	}
	return false
}

func (y *YGOProvider) handleSetPage(e *colly.HTMLElement, mu *sync.Mutex, allItems *[]products.Product, collectedSets map[string]*SetInfo, setPageToExtID map[string]string, processedCount *atomic.Int64, c *colly.Collector) {
	pageName := e.Request.Ctx.Get("pageName")
	count := processedCount.Add(1)
	if count%progressLogInterval == 0 {
		log.Printf("[fetchAll] set progreso: %d sets procesados", count)
	}

	setInfo := parseSetPage(e)
	// Usar el SetExternalID del CTS (más completo, incluye sufijos como "(All-Foil Edition)")
	if setExternalID := setPageToExtID[pageName]; setExternalID != "" {
		setInfo.ExternalID = setExternalID
	} else if setInfo.ExternalID == "" {
		setInfo.ExternalID = pageName
	}

	mu.Lock()
	collectedSets[pageName] = setInfo

	// Build set index for enrichment
	setIndex := make(map[string][]*products.Product)
	for i := range *allItems {
		if (*allItems)[i].Type == products.ProductTypeCard && (*allItems)[i].SetExternalID != "" {
			setIndex[(*allItems)[i].SetExternalID] = append(setIndex[(*allItems)[i].SetExternalID], &(*allItems)[i])
		}
	}

	// Get the SetExternalID for this wiki page
	setExternalID := setPageToExtID[pageName]
	if setExternalID == "" {
		setExternalID = pageName
	}

	// Enrich existing items with set metadata
	for _, item := range setIndex[setExternalID] {
		item.SetType = setInfo.SetType
	}

	// Enrich QuantityPerSet from the inline card list entries (already parsed from tabs)
	for lang, entries := range setInfo.CardEntries {
		for _, item := range setIndex[setExternalID] {
			if item.Lang != lang {
				continue
			}
			for _, entry := range entries {
				if item.Code == entry.CardCode && hasRarityMatch(item.Rarity, entry.Rarities) {
					if entry.IsBonus {
						item.QuantityPerSet = 0
					} else if entry.Quantity > 0 {
						item.QuantityPerSet = entry.Quantity
					} else {
						item.QuantityPerSet = 1
					}
				}
			}
		}
	}

	// Queue AJAX tabs if any
	for lang, pageName := range setInfo.AjaxTabs {
		hasLang := false
		for _, item := range setIndex[setExternalID] {
			if item.Lang == lang {
				hasLang = true
				break
			}
		}
		if !hasLang {
			continue
		}

		ctx := colly.NewContext()
		ctx.Put("phase", "set-list")
		ctx.Put("setExternalID", setExternalID)
		ctx.Put("lang", string(lang))

		ajaxURL := setCardsURL(y.yugipediaBaseUrl, pageName)
		_ = c.Request("GET", ajaxURL, nil, ctx, nil)
	}

	mu.Unlock()
}

func (y *YGOProvider) handleSetListPage(e *colly.HTMLElement, mu *sync.Mutex, allItems *[]products.Product) {
	setExternalID := e.Request.Ctx.Get("setExternalID")
	langStr := e.Request.Ctx.Get("lang")
	lang := products.LangCode(langStr)

	entries := parseSetCardListFromSelection(e.DOM, lang)
	if len(entries) == 0 {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	// Build set index for enrichment
	setIndex := make(map[string][]*products.Product)
	for i := range *allItems {
		if (*allItems)[i].Type == products.ProductTypeCard && (*allItems)[i].SetExternalID == setExternalID && (*allItems)[i].Lang == lang {
			setIndex[setExternalID] = append(setIndex[setExternalID], &(*allItems)[i])
		}
	}

	// Enrich QuantityPerSet from the parsed card list entries
	for _, item := range setIndex[setExternalID] {
		for _, entry := range entries {
			if item.Code == entry.CardCode && hasRarityMatch(item.Rarity, entry.Rarities) {
				if entry.IsBonus {
					item.QuantityPerSet = 0
				} else if entry.Quantity > 0 {
					item.QuantityPerSet = entry.Quantity
				} else {
					item.QuantityPerSet = 1
				}
			}
		}
	}
}

// deduplicateProducts removes duplicate products by UniqueKey, keeping the last occurrence.
func deduplicateProducts(items []products.Product) []products.Product {
	seen := make(map[products.ProductUniqueKey]int, len(items))
	for i, item := range items {
		seen[item.UniqueKey()] = i
	}

	result := make([]products.Product, 0, len(seen))
	for i, item := range items {
		if idx, ok := seen[item.UniqueKey()]; ok && idx == i {
			result = append(result, item)
		}
	}
	return result
}

// fetchCardsYGOPro llama a la API de YGOPRODeck y devuelve las cartas en inglés.
func (y *YGOProvider) fetchCardsYGOPro(name string) ([]YGOCard, error) {
	var reqURL string
	if name != "" {
		reqURL = fmt.Sprintf("%s/cardinfo.php?fname=%s", y.ygoproBaseUrl, url.QueryEscape(name))
	} else {
		reqURL = fmt.Sprintf("%s/cardinfo.php", y.ygoproBaseUrl)
	}

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := y.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected code YGOPRODeck: %d", resp.StatusCode)
	}

	var response struct {
		Data []YGOCard `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	if len(response.Data) == 0 {
		return nil, fmt.Errorf("no cards found")
	}

	cards := make([]YGOCard, 0, len(response.Data))
	for _, card := range response.Data {
		card.Name = html.UnescapeString(card.Name)
		card.ExternalID = card.Name
		card.Lang = products.EN
		card.Types = strings.ReplaceAll(card.Types, " ", "/")
		cards = append(cards, card)
	}
	return cards, nil
}
