package providers

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/operaodev/cardex/internal/products"
)

type SetInfo struct {
	ExternalID   string
	SetType      string // "Structure Deck", "Booster Pack", "Collector's Set", etc.
	Names        map[products.LangCode]string
	Prefixes     map[products.LangCode]string
	SetImage     string
	CardEntries  map[products.LangCode][]SetCardEntry // lang -> parsed card entries from inline tabs
	AjaxTabs     map[products.LangCode]string         // lang -> page name for AJAX tab
}

type SetCardEntry struct {
	CardCode string
	CardName string
	Rarity   string
	Rarities []string
	Quantity uint
	Category string
	Print    string // "New", "Reprint", etc.
	IsBonus  bool
	Lang     products.LangCode
}

func parseSetPage(e *colly.HTMLElement) *SetInfo {
	set := &SetInfo{
		Names:       make(map[products.LangCode]string),
		Prefixes:    make(map[products.LangCode]string),
		CardEntries: make(map[products.LangCode][]SetCardEntry),
		AjaxTabs:    make(map[products.LangCode]string),
	}

	set.ExternalID = strings.TrimSpace(e.DOM.Find("th.infobox-above").First().Text())

	currentSection := ""
	e.DOM.Find("table.infobox tr").Each(func(_ int, row *goquery.Selection) {
		// Detect section headers
		header := row.Find("th.infobox-header")
		if header.Length() > 0 {
			currentSection = strings.TrimSpace(header.Text())
			return
		}

		label := strings.TrimSpace(row.Find("th.infobox-label").Text())
		data := row.Find("td.infobox-data")

		if label == "Type" {
			set.SetType = strings.TrimSpace(data.Find("li").First().Text())
			if set.SetType == "" {
				set.SetType = strings.TrimSpace(data.Text())
			}
		}
			
		// Names only in the "Names" section
		if currentSection == "Names" {
			lang := langFromInfoboxLabel(label)
			if lang != "" {
				nameText := strings.TrimSpace(data.Text())
				// Remove <span lang> wrapper text
				spanText := strings.TrimSpace(data.Find("span").First().Text())
				if spanText != "" {
					nameText = spanText
				}
				set.Names[lang] = nameText
			}
		}

		// Prefixes only in "Set information" section
		if currentSection == "Set information" && label == "Prefix" {
			data.Find("li").Each(func(_ int, li *goquery.Selection) {
				text := li.Text()
				langCode, prefix := parsePrefix(text)
				if langCode != "" {
					set.Prefixes[langCode] = prefix
				}
			})
		}
	})

	// Parse breakdown for quantities
	set.parseBreakdown(e)

	// Parse galleries
	set.parseGalleryLinks(e)

	// Parse card list tabs only if set type contains "deck" or "set" (case-insensitive)
	sType := strings.ToLower(set.SetType)
	if strings.Contains(sType, "deck") || strings.Contains(sType, "set") {
		set.parseCardListTabs(e)
	}

	return set
}

func (s *SetInfo) parseBreakdown(e *colly.HTMLElement) {
	// QuantityPerSet y QuantityPerBox se calculan carta por carta en handleSetListPage
	// El texto del breakdown es inconsistente entre tipos de sets
}

func (s *SetInfo) parseGalleryLinks(e *colly.HTMLElement) {
	// Get main gallery image (first gallerybox with a valid image)
	firstImg := e.DOM.Find("ul.gallery .gallerybox a.image img").First()
	if imgSrc, exists := firstImg.Attr("src"); exists {
		s.SetImage = imgSrc
	}
}

func (s *SetInfo) parseCardListTabs(e *colly.HTMLElement) {
	langFromTitle := map[string]products.LangCode{
		"English":              products.EN,
		"French":               products.FR,
		"German":               products.DE,
		"Italian":              products.IT,
		"Portuguese":           products.PT,
		"Spanish":              products.SP,
		"Japanese":             products.JP,
		"Asian-English":        products.AE,
		"Korean":               products.KR,
		"Simplified Chinese":   products.SC,
		"North American English": products.EN,
	}

	e.DOM.Find("div.tabbertab").Each(func(_ int, tab *goquery.Selection) {
		title := strings.TrimSpace(tab.AttrOr("title", ""))
		lang, ok := langFromTitle[title]
		if !ok {
			return
		}

		// Check if it is an AJAX tab that doesn't have the table loaded
		ajaxDiv := tab.Find("div.set-list-ajax-tab")
		if ajaxDiv.Length() > 0 && tab.Find("table.wikitable.sortable.card-list.set-list__main").Length() == 0 {
			page := ajaxDiv.AttrOr("data-page", "")
			if page != "" {
				s.AjaxTabs[lang] = page
			}
			return
		}

		// Parse card entries directly from the inline tab content
		entries := parseSetCardListFromSelection(tab, lang)
		if len(entries) > 0 {
			s.CardEntries[lang] = entries
		}
	})
}

func parseSetCardList(e *colly.HTMLElement, lang products.LangCode) []SetCardEntry {
	var entries []SetCardEntry

	table := e.DOM.Find("table.wikitable.sortable.card-list.set-list__main").First()
	if table.Length() == 0 {
		return entries
	}

	// Detect column indices
	colIndex := map[string]int{}
	table.Find("th").Each(func(i int, th *goquery.Selection) {
		class := th.AttrOr("class", "")
		switch {
		case strings.Contains(class, "set-list__main__header--card-number"):
			colIndex["card_number"] = i
		case strings.Contains(class, "set-list__main__header--name"):
			colIndex["name"] = i
		case strings.Contains(class, "set-list__main__header--rarity"):
			colIndex["rarity"] = i
		case strings.Contains(class, "set-list__main__header--category"):
			colIndex["category"] = i
		case strings.Contains(class, "set-list__main__header--quantity"):
			colIndex["quantity"] = i
		}
	})

	table.Find("tbody tr").Each(func(_ int, row *goquery.Selection) {
		tds := row.Find("td")
		if tds.Length() == 0 {
			return
		}

		getCell := func(key string) string {
			idx, ok := colIndex[key]
			if !ok || idx >= tds.Length() {
				return ""
			}
			return strings.TrimSpace(tds.Eq(idx).Text())
		}

		code := getCell("card_number")
		if code == "" {
			return
		}

		nameText := getCell("name")
		nameText = strings.Trim(nameText, "\"")

		rarity := getCell("rarity")
		category := getCell("category")
		printType := getCell("print")

		// Parse rarities slice
		var rarities []string
		if idx, ok := colIndex["rarity"]; ok && idx < tds.Length() {
			rarities = extractRaritiesFromCell(tds.Eq(idx))
		}

		quantity := uint(1)
		if qtyStr := getCell("quantity"); qtyStr != "" {
			if n, err := strconv.ParseUint(qtyStr, 10, 32); err == nil {
				quantity = uint(n)
			}
		}

		entries = append(entries, SetCardEntry{
			CardCode: code,
			CardName: nameText,
			Rarity:   rarity,
			Rarities: rarities,
			Quantity: quantity,
			Category: category,
			Print:    printType,
			Lang:     lang,
		})
	})

	return entries
}

// parseSetCardListFromSelection extracts card entries from a goquery.Selection
// (e.g. a tab div or a full page body). It detects bonus sections by looking
// for preceding <h3> headings containing "bonus".
func parseSetCardListFromSelection(root *goquery.Selection, lang products.LangCode) []SetCardEntry {
	var allEntries []SetCardEntry

	root.Find("table.wikitable.sortable.card-list.set-list__main").Each(func(_ int, table *goquery.Selection) {
		isBonus := false

		// Find the closest preceding h3 of either the table itself or its parent container (like div.set-list)
		parent := table.Parent()

		// Helper to check if a selection is preceded by an h2, h3 or h4 containing "bonus"
		checkPrecedingH3 := func(sel *goquery.Selection) bool {
			for prev := sel.Prev(); prev.Length() > 0; prev = prev.Prev() {
				if prev.Is("h2") || prev.Is("h3") || prev.Is("h4") {
					headingText := strings.ToLower(prev.Text())
					return strings.Contains(headingText, "bonus")
				}
			}
			return false
		}

		if checkPrecedingH3(table) {
			isBonus = true
		} else if parent.Length() > 0 && checkPrecedingH3(parent) {
			isBonus = true
		}

		entries := parseSetCardListTable(table, lang, isBonus)
		allEntries = append(allEntries, entries...)
	})

	return allEntries
}

// parseSetCardListWithBonuses wraps parseSetCardListFromSelection for colly.HTMLElement.
func parseSetCardListWithBonuses(e *colly.HTMLElement, lang products.LangCode) []SetCardEntry {
	return parseSetCardListFromSelection(e.DOM, lang)
}

func parseSetCardListTable(table *goquery.Selection, lang products.LangCode, isBonus bool) []SetCardEntry {
	var entries []SetCardEntry

	colIndex := map[string]int{}
	table.Find("th").Each(func(i int, th *goquery.Selection) {
		class := th.AttrOr("class", "")
		switch {
		case strings.Contains(class, "set-list__main__header--card-number"):
			colIndex["card_number"] = i
		case strings.Contains(class, "set-list__main__header--name"):
			colIndex["name"] = i
		case strings.Contains(class, "set-list__main__header--rarity"):
			colIndex["rarity"] = i
		case strings.Contains(class, "set-list__main__header--category"):
			colIndex["category"] = i
		case strings.Contains(class, "set-list__main__header--print"):
			colIndex["print"] = i
		case strings.Contains(class, "set-list__main__header--quantity"):
			colIndex["quantity"] = i
		}
	})

	table.Find("tbody tr").Each(func(_ int, row *goquery.Selection) {
		tds := row.Find("td")
		if tds.Length() == 0 {
			return
		}

		getCell := func(key string) string {
			idx, ok := colIndex[key]
			if !ok || idx >= tds.Length() {
				return ""
			}
			return strings.TrimSpace(tds.Eq(idx).Text())
		}

		code := getCell("card_number")
		if code == "" {
			return
		}

		nameText := getCell("name")
		nameText = strings.Trim(nameText, "\"")

		rarity := getCell("rarity")
		category := getCell("category")
		printType := getCell("print")

		// Parse rarities slice
		var rarities []string
		if idx, ok := colIndex["rarity"]; ok && idx < tds.Length() {
			rarities = extractRaritiesFromCell(tds.Eq(idx))
		}

		quantity := uint(1)
		if isBonus {
			quantity = 0
		} else if qtyStr := getCell("quantity"); qtyStr != "" {
			if n, err := strconv.ParseUint(qtyStr, 10, 32); err == nil {
				quantity = uint(n)
			}
		}

		entries = append(entries, SetCardEntry{
			CardCode: code,
			CardName: nameText,
			Rarity:   rarity,
			Rarities: rarities,
			Quantity: quantity,
			Category: category,
			Print:    printType,
			IsBonus:  isBonus,
			Lang:     lang,
		})
	})

	return entries
}

func langFromInfoboxLabel(label string) products.LangCode {
	switch label {
	case "English":
		return products.EN
	case "French":
		return products.FR
	case "German":
		return products.DE
	case "Italian":
		return products.IT
	case "Portuguese":
		return products.PT
	case "Spanish":
		return products.SP
	case "Japanese":
		return products.JP
	case "Korean":
		return products.KR
	case "Simplified Chinese":
		return products.SC
	}
	return ""
}

func parsePrefix(text string) (products.LangCode, string) {
	// Format: "SR14-EN (en)" or "CH01-FR (fr)"
	text = strings.TrimSpace(text)

	// Find language code in parentheses
	langMatch := regexp.MustCompile(`\(([a-z]{2})\)$`)
	m := langMatch.FindStringSubmatch(text)
	if m == nil {
		return "", ""
	}

	langCode := langCodeFromISO(m[1])
	prefix := strings.TrimSpace(strings.TrimSuffix(text, m[0]))

	return langCode, prefix
}

func langCodeFromISO(iso string) products.LangCode {
	switch iso {
	case "en":
		return products.EN
	case "fr":
		return products.FR
	case "de":
		return products.DE
	case "it":
		return products.IT
	case "pt":
		return products.PT
	case "sp":
		return products.SP
	case "jp":
		return products.JP
	case "kr":
		return products.KR
	case "sc":
		return products.SC
	}
	return ""
}

func extractNumber(text string) uint {
	re := regexp.MustCompile(`\b(\d+)\b`)
	m := re.FindStringSubmatch(text)
	if m == nil {
		return 0
	}
	if n, err := strconv.ParseUint(m[1], 10, 32); err == nil {
		return uint(n)
	}
	return 0
}

func setCardsURL(baseURL, pageName string) string {
	formatted := strings.ReplaceAll(pageName, " ", "_")
	return fmt.Sprintf("%s/wiki/%s", baseURL, url.PathEscape(formatted))
}

func setInfoToProducts(sets map[string]*SetInfo) []products.Product {
	var result []products.Product

	for _, set := range sets {
		for lang, name := range set.Names {
			item := products.Product{
				Type:          products.ProductTypeSet,
				ExternalID:    set.ExternalID,
				SetExternalID: set.ExternalID,
				TCG:           products.YGO,
				Lang:          lang,
				Name:          name,
				SetName:       name,
				SetRegionCode: set.Prefixes[lang],
				SetCode:       extractSetCode(set.Prefixes[lang]),
				SetType:       set.SetType,
				SetImageLarge: deriveSetImageLarge(set.SetImage),
				SetImageSmall: deriveSetImageSmall(set.SetImage),
			}
			result = append(result, item)
		}
	}

	return result
}

func convertSetToProducts(set *SetInfo, cardEntries map[products.LangCode][]SetCardEntry) []products.Product {
	var result []products.Product

	for lang, cards := range cardEntries {
		for _, card := range cards {
			name := set.Names[lang]
			if name == "" {
				name = set.Names[products.EN]
			}
			if name == "" {
				name = set.ExternalID
			}

			item := products.Product{
				Type:           products.ProductTypeSet,
				ExternalID:     set.ExternalID,
				SetExternalID:  set.ExternalID,
				TCG:            products.YGO,
				Lang:           lang,
				Name:           name,
				SetName:        name,
				SetCode:        card.CardCode,
				CardTypes:      card.Category,
				SetType:        set.SetType,
				QuantityPerSet: card.Quantity,
			}

			if set.SetImage != "" {
				item.SetImageLarge = deriveSetImageLarge(set.SetImage)
				item.SetImageSmall = deriveSetImageSmall(set.SetImage)
			}

			result = append(result, item)
		}
	}

	return result
}

func extractRaritiesFromCell(cell *goquery.Selection) []string {
	var rarities []string

	// Try finding <a> links first
	cell.Find("a").Each(func(_ int, a *goquery.Selection) {
		if r := strings.TrimSpace(a.Text()); r != "" {
			rarities = append(rarities, r)
		}
	})

	if len(rarities) > 0 {
		return rarities
	}

	// Fallback: if no <a> tags, get the raw HTML/text and split by <br> tags or newlines
	h, err := cell.Html()
	if err == nil {
		// Replace various <br> forms with a newline
		brRe := regexp.MustCompile(`(?i)<br\s*/?>`)
		h = brRe.ReplaceAllString(h, "\n")
		// Parse back into a temporary document fragment to get clean text
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(h))
		if err == nil {
			text := doc.Text()
			for _, line := range strings.Split(text, "\n") {
				if r := strings.TrimSpace(line); r != "" {
					rarities = append(rarities, r)
				}
			}
		}
	}

	if len(rarities) == 0 {
		if r := strings.TrimSpace(cell.Text()); r != "" {
			rarities = append(rarities, r)
		}
	}

	return rarities
}

func extractSetCode(regionCode string) string {
	if regionCode == "" {
		return ""
	}
	return regexp.MustCompile(`-[A-Z]{2,4}$`).ReplaceAllString(regionCode, "")
}

func deriveSetImageLarge(setImage string) string {
	if setImage == "" {
		return ""
	}
	pngIdx := strings.Index(setImage, ".png")
	if pngIdx == -1 {
		return setImage
	}
	url := setImage[:pngIdx+4]
	if strings.Contains(url, "/thumb/") {
		return strings.ReplaceAll(url, "/thumb/", "/")
	}
	return url
}

func deriveSetImageSmall(setImage string) string {
	if setImage == "" {
		return ""
	}
	re := regexp.MustCompile(`\d+px-`)
	if re.MatchString(setImage) {
		return re.ReplaceAllString(setImage, "257px-")
	}
	return setImage
}
