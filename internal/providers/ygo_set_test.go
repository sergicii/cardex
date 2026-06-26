package providers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/operaodev/cardex/internal/products"
)

func TestParseSetPage_StructureDeck(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/page", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintln(w, `<!DOCTYPE html><html><body>
			<table class="infobox">
				<tr><th class="infobox-above">Structure Deck: Fire Kings</th></tr>
				<tr><th colspan="2" class="infobox-header">Names</th></tr>
				<tr><th class="infobox-label">English</th><td class="infobox-data"><i>Structure Deck: Fire Kings</i></td></tr>
				<tr><th class="infobox-label">French</th><td class="infobox-data"><i>Deck de Structure : Les Rois du Feu</i></td></tr>
				<tr><th class="infobox-label">Spanish</th><td class="infobox-data"><i>Baraja de Estructura: Reyes de Fuego</i></td></tr>
				<tr><th colspan="2" class="infobox-header">Set information</th></tr>
				<tr><th class="infobox-label">Medium</th><td class="infobox-data"><a href="/wiki/TCG">TCG</a></td></tr>
				<tr><th class="infobox-label">Type</th><td class="infobox-data"><ul><li>Structure Deck</li></ul></td></tr>
				<tr><th class="infobox-label">Number of cards</th><td class="infobox-data">48</td></tr>
				<tr><th class="infobox-label">Prefix</th><td class="infobox-data">
					<ul><li>SR14-EN (en)</li><li>SR14-FR (fr)</li><li>SR14-SP (sp)</li></ul>
				</td></tr>
				<tr><th colspan="2" class="infobox-header">Release dates</th></tr>
				<tr><th class="infobox-label">English</th><td class="infobox-data">April 14, 2022</td></tr>
				<tr><th class="infobox-label">French</th><td class="infobox-data">April 14, 2022</td></tr>
			</table>
			<h2><span class="mw-headline" id="Breakdown">Breakdown</span></h2>
			<p>Each <i>Structure Deck: Fire Kings</i> contains:</p>
			<ul><li>1 Preconstructed Deck of 48 cards</li></ul>
			<ul class="gallery"><li class="gallerybox"><a class="image"><img src="https://ms.yugipedia.com/SR14-DeckEN.png" /></a></li></ul>
		</body></html>`)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := createColly()
	var captured *SetInfo
	c.OnHTML("body", func(e *colly.HTMLElement) {
		captured = parseSetPage(e)
	})
	c.Visit(server.URL + "/page")
	c.Wait()

	if captured == nil {
		t.Fatal("parseSetPage did not capture data")
	}

	if captured.ExternalID != "Structure Deck: Fire Kings" {
		t.Errorf("ExternalID: got %q, want %q", captured.ExternalID, "Structure Deck: Fire Kings")
	}
	if captured.SetType != "Structure Deck" {
		t.Errorf("SetType: got %q, want %q", captured.SetType, "Structure Deck")
	}

	if len(captured.Names) != 3 {
		t.Errorf("Names: got %d entries, want 3", len(captured.Names))
	}
	if captured.Names[products.EN] != "Structure Deck: Fire Kings" {
		t.Errorf("EN name: got %q", captured.Names[products.EN])
	}
	if captured.Names[products.FR] != "Deck de Structure : Les Rois du Feu" {
		t.Errorf("FR name: got %q", captured.Names[products.FR])
	}
}

func TestParseSetCardList_WithQuantity(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/list", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintln(w, `<!DOCTYPE html><html><body>
			<div class="set-list">
				<table class="wikitable sortable card-list set-list__main">
					<tbody>
						<tr>
							<th scope="col" class="set-list__main__header set-list__main__header--card-number">Card number</th>
							<th scope="col" class="set-list__main__header set-list__main__header--name">Name</th>
							<th scope="col" class="set-list__main__header set-list__main__header--rarity">Rarity</th>
							<th scope="col" class="set-list__main__header set-list__main__header--category">Category</th>
							<th scope="col" class="set-list__main__header set-list__main__header--print">Print</th>
							<th scope="col" class="set-list__main__header set-list__main__header--quantity">Quantity</th>
						</tr>
						<tr>
							<td>SR14-EN001</td><td>"Sacred Fire King Garunix"</td>
							<td>Ultra Rare</td><td>Effect Monster</td><td>New</td><td>1</td>
						</tr>
						<tr>
							<td>SR14-EN004</td><td>"Fire King Avatar Garunix"</td>
							<td>Common</td><td>Effect Monster</td><td>Reprint</td><td>3</td>
						</tr>
					</tbody>
				</table>
			</div>
		</body></html>`)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := createColly()
	var entries []SetCardEntry
	c.OnHTML("body", func(e *colly.HTMLElement) {
		entries = parseSetCardListWithBonuses(e, products.EN)
	})
	c.Visit(server.URL + "/list")
	c.Wait()

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	if entries[1].CardCode != "SR14-EN004" || entries[1].Quantity != 3 {
		t.Errorf("Garunix: code=%q qty=%d, want SR14-EN004 qty=3", entries[1].CardCode, entries[1].Quantity)
	}
	if entries[1].IsBonus {
		t.Error("Garunix should not be a bonus card")
	}
}

func TestParseSetCardList_WithBonusCards(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/bonus", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintln(w, `<!DOCTYPE html><html><body>
			<h3><span class="mw-headline" id="Bonus_cards">Bonus cards</span></h3>
			<div class="set-list">
				<table class="wikitable sortable card-list set-list__main">
					<tbody>
						<tr>
							<th scope="col" class="set-list__main__header set-list__main__header--card-number">Card number</th>
							<th scope="col" class="set-list__main__header set-list__main__header--name">Name</th>
							<th scope="col" class="set-list__main__header set-list__main__header--rarity">Rarity</th>
							<th scope="col" class="set-list__main__header set-list__main__header--category">Category</th>
							<th scope="col" class="set-list__main__header set-list__main__header--print">Print</th>
						</tr>
						<tr>
							<td>L26D-ENS01</td><td>"Sky Striker Ace - Raye"</td>
							<td>Secret Rare</td><td>Effect Monster</td><td>Reprint</td>
						</tr>
					</tbody>
				</table>
			</div>
		</body></html>`)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := createColly()
	var entries []SetCardEntry
	c.OnHTML("body", func(e *colly.HTMLElement) {
		entries = parseSetCardListWithBonuses(e, products.EN)
	})
	c.Visit(server.URL + "/bonus")
	c.Wait()

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if !entries[0].IsBonus {
		t.Error("Expected bonus card to be flagged as bonus")
	}
	if entries[0].Quantity != 0 {
		t.Errorf("Expected bonus card Quantity=0, got %d", entries[0].Quantity)
	}
}

func TestParseSetCardList_DeckWithBonusAndQuantity(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/deck", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintln(w, `<!DOCTYPE html><html><body>
			<h3><span class="mw-headline" id="Bonus_cards">Bonus cards</span></h3>
			<div class="set-list">
				<table class="wikitable sortable card-list set-list__main">
					<tbody>
						<tr>
							<th scope="col" class="set-list__main__header set-list__main__header--card-number">Card number</th>
							<th scope="col" class="set-list__main__header set-list__main__header--name">Name</th>
							<th scope="col" class="set-list__main__header set-list__main__header--rarity">Rarity</th>
							<th scope="col" class="set-list__main__header set-list__main__header--category">Category</th>
							<th scope="col" class="set-list__main__header set-list__main__header--print">Print</th>
						</tr>
						<tr>
							<td>CH01-EN014</td><td>"Dogmatika Ecclesia, the Virtuous"</td>
							<td>Secret Rare</td><td>Effect Monster</td><td>New artwork</td>
						</tr>
					</tbody>
				</table>
			</div>
			<h3><span class="mw-headline" id="Preconstructed_Deck">Preconstructed Deck</span></h3>
			<div class="set-list">
				<table class="wikitable sortable card-list set-list__main">
					<tbody>
						<tr>
							<th scope="col" class="set-list__main__header set-list__main__header--card-number">Card number</th>
							<th scope="col" class="set-list__main__header set-list__main__header--name">Name</th>
							<th scope="col" class="set-list__main__header set-list__main__header--rarity">Rarity</th>
							<th scope="col" class="set-list__main__header set-list__main__header--category">Category</th>
							<th scope="col" class="set-list__main__header set-list__main__header--print">Print</th>
							<th scope="col" class="set-list__main__header set-list__main__header--quantity">Quantity</th>
						</tr>
						<tr>
							<td>L26D-ENS01</td><td>"Sky Striker Ace - Raye"</td>
							<td>Common</td><td>Effect Monster</td><td>Reprint</td><td>3</td>
						</tr>
						<tr>
							<td>L26D-ENS02</td><td>"Nibiru, the Primal Being"</td>
							<td>Common</td><td>Effect Monster</td><td>Reprint</td><td>2</td>
						</tr>
						<tr>
							<td>L26D-ENS04</td><td>"Sky Striker Mecha - Adil Saber"</td>
							<td>Ultra Rare</td><td>Effect Monster</td><td>New</td><td>1</td>
						</tr>
					</tbody>
				</table>
			</div>
		</body></html>`)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := createColly()
	var entries []SetCardEntry
	c.OnHTML("body", func(e *colly.HTMLElement) {
		entries = parseSetCardListWithBonuses(e, products.EN)
	})
	c.Visit(server.URL + "/deck")
	c.Wait()

	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}

	// First entry is bonus
	if !entries[0].IsBonus {
		t.Error("entries[0] should be bonus")
	}
	if entries[0].Quantity != 0 {
		t.Errorf("bonus entry Quantity: got %d, want 0", entries[0].Quantity)
	}

	// Deck entries with quantities
	if entries[0].IsBonus != true || entries[1].IsBonus != false || entries[2].IsBonus != false || entries[3].IsBonus != false {
		t.Error("only the first entry should be bonus")
	}
	if entries[1].Quantity != 3 {
		t.Errorf("entries[1] Quantity: got %d, want 3", entries[1].Quantity)
	}
	if entries[2].Quantity != 2 {
		t.Errorf("entries[2] Quantity: got %d, want 2", entries[2].Quantity)
	}
	if entries[3].Quantity != 1 {
		t.Errorf("entries[3] Quantity: got %d, want 1", entries[3].Quantity)
	}
}

func TestParseSetCardList_DeckNoQuantityColumn(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/deck-noquantity", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintln(w, `<!DOCTYPE html><html><body>
			<h3><span class="mw-headline" id="Preconstructed_Deck">Preconstructed Deck</span></h3>
			<div class="set-list">
				<table class="wikitable sortable card-list set-list__main">
					<tbody>
						<tr>
							<th scope="col" class="set-list__main__header set-list__main__header--card-number">Card number</th>
							<th scope="col" class="set-list__main__header set-list__main__header--name">Name</th>
							<th scope="col" class="set-list__main__header set-list__main__header--rarity">Rarity</th>
							<th scope="col" class="set-list__main__header set-list__main__header--category">Category</th>
							<th scope="col" class="set-list__main__header set-list__main__header--print">Print</th>
						</tr>
						<tr>
							<td>CH01-EN001</td><td>"Fallen of Albaz"</td>
							<td>Ultra Rare</td><td>Effect Monster</td><td>Reprint</td>
						</tr>
						<tr>
							<td>CH01-EN002</td><td>"Incredible Ecclesia"</td>
							<td>Ultra Rare</td><td>Effect Tuner monster</td><td>Reprint</td>
						</tr>
					</tbody>
				</table>
			</div>
		</body></html>`)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := createColly()
	var entries []SetCardEntry
	c.OnHTML("body", func(e *colly.HTMLElement) {
		entries = parseSetCardListWithBonuses(e, products.EN)
	})
	c.Visit(server.URL + "/deck-noquantity")
	c.Wait()

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	for i, entry := range entries {
		if entry.IsBonus {
			t.Errorf("entries[%d] should NOT be bonus", i)
		}
		if entry.Quantity != 1 {
			t.Errorf("entries[%d] Quantity: got %d, want 1 (default for deck cards without quantity column)", i, entry.Quantity)
		}
	}
}

func TestSetInfoToProducts_NoQuantity(t *testing.T) {
	sets := map[string]*SetInfo{
		"Test Set": {
			ExternalID: "Test Set",
			SetType:    "Booster pack",
			Names: map[products.LangCode]string{
				products.EN: "Test Set",
				products.FR: "Set de Test",
			},
			Prefixes: map[products.LangCode]string{
				products.EN: "TEST-EN",
				products.FR: "TEST-FR",
			},
		},
	}

	prods := setInfoToProducts(sets)

	if len(prods) != 2 {
		t.Fatalf("expected 2 products, got %d", len(prods))
	}

	for _, p := range prods {
		if p.Type != products.ProductTypeSet {
			t.Errorf("product type should be 'set', got %q", p.Type)
		}
		if p.QuantityPerSet != 0 {
			t.Errorf("set product QuantityPerSet should be 0, got %d", p.QuantityPerSet)
		}
	}
}

func TestDeduplicateProducts(t *testing.T) {
	items := []products.Product{
		{ExternalID: "Card1", SetExternalID: "Set1", TCG: products.YGO, Code: "S01-001", Lang: products.EN, Rarity: "Common", Edition: ""},
		{ExternalID: "Card1", SetExternalID: "Set1", TCG: products.YGO, Code: "S01-001", Lang: products.EN, Rarity: "Common", Edition: ""},
		{ExternalID: "Card2", SetExternalID: "Set1", TCG: products.YGO, Code: "S01-002", Lang: products.EN, Rarity: "Rare", Edition: ""},
	}

	result := deduplicateProducts(items)

	if len(result) != 2 {
		t.Fatalf("expected 2 items after dedup, got %d", len(result))
	}
}

func TestNormalizeProducts(t *testing.T) {
	items := []products.Product{
		{Name: "The Fallen &amp; The Virtuous  ", SetName: "Duelist Nexus", Rarity: "Ultra Rare", Edition: "1st Edition"},
	}

	normalizeProducts(items)

	if items[0].Name != "The Fallen & The Virtuous" {
		t.Errorf("Name: got %q, want %q", items[0].Name, "The Fallen & The Virtuous")
	}
	if items[0].SetName != "Duelist Nexus" {
		t.Errorf("SetName: got %q, want %q", items[0].SetName, "Duelist Nexus")
	}
}

func TestParseCTSTables_ExtractsSetURL(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/card", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintln(w, `<!DOCTYPE html><html><body>
			<table id="cts--EN" class="wikitable sortable card-list cts">
				<tbody>
					<tr>
						<th scope="col" class="cts__header--number">Number</th>
						<th scope="col" class="cts__header--set">Set</th>
						<th scope="col" class="cts__header--rarity">Rarity</th>
					</tr>
					<tr>
						<td><a href="/wiki/CH01-EN019" class="mw-redirect">CH01-EN019</a></td>
						<td><a href="/wiki/THE_CHRONICLES_DECK:_The_Fallen_%26_The_Virtuous_(All-Foil_Edition)"><i>THE CHRONICLES DECK: The Fallen &amp; The Virtuous</i> (All-Foil Edition)</a></td>
						<td><a href="/wiki/Ultra_Rare">Ultra Rare</a></td>
					</tr>
				</tbody>
			</table>
		</body></html>`)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := createColly()
	var entries []printEntry
	c.OnHTML("body", func(e *colly.HTMLElement) {
		entries = parseCTSTables(e)
	})
	c.Visit(server.URL + "/card")
	c.Wait()

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	if entries[0].SetURL == "" {
		t.Error("SetURL should not be empty")
	}
	if entries[0].SetExternalID != "THE CHRONICLES DECK: The Fallen & The Virtuous (All-Foil Edition)" {
		t.Errorf("SetExternalID: got %q", entries[0].SetExternalID)
	}
}

func TestParseSetPage_CardListTabsCondition(t *testing.T) {
	htmlContent := func(setType string) string {
		return fmt.Sprintf(`<!DOCTYPE html><html><body>
			<table class="infobox">
				<tr><th class="infobox-above">Test Set</th></tr>
				<tr><th class="infobox-label">Type</th><td class="infobox-data">%s</td></tr>
			</table>
			<div class="tabbertab" title="English">
				<table class="wikitable sortable card-list set-list__main">
					<tbody>
						<tr>
							<th scope="col" class="set-list__main__header set-list__main__header--card-number">Card number</th>
							<th scope="col" class="set-list__main__header set-list__main__header--name">Name</th>
						</tr>
						<tr><td>TEST-EN001</td><td>"Test Card"</td></tr>
					</tbody>
				</table>
			</div>
		</body></html>`, setType)
	}

	// 1. Set type containing "deck"
	{
		mux := http.NewServeMux()
		mux.HandleFunc("/deck", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintln(w, htmlContent("Structure Deck"))
		})
		server := httptest.NewServer(mux)
		defer server.Close()

		c := createColly()
		var captured *SetInfo
		c.OnHTML("body", func(e *colly.HTMLElement) {
			captured = parseSetPage(e)
		})
		c.Visit(server.URL + "/deck")
		c.Wait()

		if captured == nil || len(captured.CardEntries[products.EN]) != 1 {
			t.Errorf("expected card entries for type 'Structure Deck', got %v", captured)
		}
	}

	// 2. Set type containing "set"
	{
		mux := http.NewServeMux()
		mux.HandleFunc("/set", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintln(w, htmlContent("Collector's Box Set"))
		})
		server := httptest.NewServer(mux)
		defer server.Close()

		c := createColly()
		var captured *SetInfo
		c.OnHTML("body", func(e *colly.HTMLElement) {
			captured = parseSetPage(e)
		})
		c.Visit(server.URL + "/set")
		c.Wait()

		if captured == nil || len(captured.CardEntries[products.EN]) != 1 {
			t.Errorf("expected card entries for type 'Collector's Box Set', got %v", captured)
		}
	}

	// 3. Set type NOT containing "deck" or "set"
	{
		mux := http.NewServeMux()
		mux.HandleFunc("/pack", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintln(w, htmlContent("Booster pack"))
		})
		server := httptest.NewServer(mux)
		defer server.Close()

		c := createColly()
		var captured *SetInfo
		c.OnHTML("body", func(e *colly.HTMLElement) {
			captured = parseSetPage(e)
		})
		c.Visit(server.URL + "/pack")
		c.Wait()

		if captured == nil || len(captured.CardEntries) != 0 {
			t.Errorf("expected NO card entries for type 'Booster pack', got %v", captured)
		}
	}
}

func TestParseSetPage_MaidenOfWhiteDifferentRarities(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/maiden", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintln(w, `<!DOCTYPE html><html><body>
			<table class="infobox">
				<tr><th class="infobox-above">Structure Deck: Blue-Eyes White Destiny</th></tr>
				<tr><th class="infobox-label">Type</th><td class="infobox-data">Structure Deck</td></tr>
			</table>
			<div class="tabbertab" title="English">
				<h3><span class="mw-headline" id="Bonus_cards">Bonus cards</span></h3>
				<div class="set-list">
					<table class="wikitable sortable card-list set-list__main">
						<tbody>
							<tr>
								<th scope="col" class="set-list__main__header set-list__main__header--card-number">Card number</th>
								<th scope="col" class="set-list__main__header set-list__main__header--name">Name</th>
								<th scope="col" class="set-list__main__header set-list__main__header--rarity">Rarity</th>
							</tr>
							<tr>
								<td>SDWD-EN041</td><td>"Maiden of White"</td>
								<td>Secret Rare<br>Quarter Century Secret Rare</td>
							</tr>
						</tbody>
					</table>
				</div>
				<h3><span class="mw-headline" id="Preconstructed_Deck">Preconstructed Deck</span></h3>
				<div class="set-list">
					<table class="wikitable sortable card-list set-list__main">
						<tbody>
							<tr>
								<th scope="col" class="set-list__main__header set-list__main__header--card-number">Card number</th>
								<th scope="col" class="set-list__main__header set-list__main__header--name">Name</th>
								<th scope="col" class="set-list__main__header set-list__main__header--rarity">Rarity</th>
								<th scope="col" class="set-list__main__header set-list__main__header--quantity">Quantity</th>
							</tr>
							<tr>
								<td>SDWD-EN041</td><td>"Maiden of White"</td>
								<td>Ultra Rare</td><td>1</td>
							</tr>
						</tbody>
					</table>
				</div>
			</div>
		</body></html>`)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := createColly()
	var captured *SetInfo
	c.OnHTML("body", func(e *colly.HTMLElement) {
		captured = parseSetPage(e)
	})
	c.Visit(server.URL + "/maiden")
	c.Wait()

	if captured == nil {
		t.Fatal("expected set to be captured")
	}

	entries := captured.CardEntries[products.EN]
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries parsed, got %d", len(entries))
	}

	// Verify entries parsed from HTML
	if entries[0].CardCode != "SDWD-EN041" || !entries[0].IsBonus || len(entries[0].Rarities) != 2 {
		t.Errorf("invalid bonus entry: %+v", entries[0])
	}
	if entries[1].CardCode != "SDWD-EN041" || entries[1].IsBonus || len(entries[1].Rarities) != 1 || entries[1].Quantity != 1 {
		t.Errorf("invalid deck entry: %+v", entries[1])
	}

	// Verify hasRarityMatch works on them
	if !hasRarityMatch("Secret Rare", entries[0].Rarities) {
		t.Error("Secret Rare should match bonus entry rarities")
	}
	if !hasRarityMatch("Quarter Century Secret Rare", entries[0].Rarities) {
		t.Error("Quarter Century Secret Rare should match bonus entry rarities")
	}
	if hasRarityMatch("Ultra Rare", entries[0].Rarities) {
		t.Error("Ultra Rare should NOT match bonus entry rarities")
	}

	if !hasRarityMatch("Ultra Rare", entries[1].Rarities) {
		t.Error("Ultra Rare should match deck entry rarities")
	}
	if hasRarityMatch("Secret Rare", entries[1].Rarities) {
		t.Error("Secret Rare should NOT match deck entry rarities")
	}
}

func TestParseSetPage_MaidenOfWhiteFR(t *testing.T) {
	htmlContent := `<!DOCTYPE html><html><body>
		<div class="tabbertab" title="French">
			<h2><span class="mw-headline" id="Bonus_cards">Bonus cards</span></h2>
			<div class="set-list">
				<table class="wikitable sortable card-list set-list__main">
					<thead>
						<tr>
							<th scope="col" class="set-list__main__header set-list__main__header--card-number">Card number</th>
							<th scope="col" class="set-list__main__header set-list__main__header--name">English name</th>
							<th scope="col" class="set-list__main__header set-list__main__header--localized-name">French name</th>
							<th scope="col" class="set-list__main__header set-list__main__header--rarity">Rarity</th>
						</tr>
					</thead>
					<tbody>
						<tr>
							<td>SDWD-FR041</td><td>"Maiden of White"</td><td>"Demoiselle Blanche"</td>
							<td><a href="/wiki/Secret_Rare">Secret Rare</a><br><a href="/wiki/Quarter_Century_Secret_Rare">Quarter Century Secret Rare</a></td>
						</tr>
					</tbody>
				</table>
			</div>
			<h2><span class="mw-headline" id="Preconstructed_Deck">Preconstructed Deck</span></h2>
			<div class="set-list">
				<table class="wikitable sortable card-list set-list__main">
					<thead>
						<tr>
							<th scope="col" class="set-list__main__header set-list__main__header--card-number">Card number</th>
							<th scope="col" class="set-list__main__header set-list__main__header--name">English name</th>
							<th scope="col" class="set-list__main__header set-list__main__header--localized-name">French name</th>
							<th scope="col" class="set-list__main__header set-list__main__header--rarity">Rarity</th>
							<th scope="col" class="set-list__main__header set-list__main__header--quantity">Quantity</th>
						</tr>
					</thead>
					<tbody>
						<tr>
							<td>SDWD-FR041</td><td>"Maiden of White"</td><td>"Demoiselle Blanche"</td>
							<td><a href="/wiki/Ultra_Rare">Ultra Rare</a></td><td>1</td>
						</tr>
					</tbody>
				</table>
			</div>
		</div>
	</body></html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("failed to parse test HTML: %v", err)
	}

	tab := doc.Find("div.tabbertab").First()
	entries := parseSetCardListFromSelection(tab, products.FR)

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// entry 0 is bonus
	if entries[0].CardCode != "SDWD-FR041" || !entries[0].IsBonus || len(entries[0].Rarities) != 2 {
		t.Errorf("invalid bonus entry: %+v", entries[0])
	}

	// entry 1 is deck
	if entries[1].CardCode != "SDWD-FR041" || entries[1].IsBonus || len(entries[1].Rarities) != 1 || entries[1].Quantity != 1 {
		t.Errorf("invalid deck entry: %+v", entries[1])
	}

	// Check matching
	productsList := []products.Product{
		{Code: "SDWD-FR041", Lang: products.FR, Rarity: "Secret Rare", SetExternalID: "Structure Deck: Blue-Eyes White Destiny", Type: products.ProductTypeCard},
		{Code: "SDWD-FR041", Lang: products.FR, Rarity: "Quarter Century Secret Rare", SetExternalID: "Structure Deck: Blue-Eyes White Destiny", Type: products.ProductTypeCard},
		{Code: "SDWD-FR041", Lang: products.FR, Rarity: "Ultra Rare", SetExternalID: "Structure Deck: Blue-Eyes White Destiny", Type: products.ProductTypeCard},
	}

	// Simulate enrichment
	for i := range productsList {
		p := &productsList[i]
		for _, entry := range entries {
			if p.Code == entry.CardCode && hasRarityMatch(p.Rarity, entry.Rarities) {
				if entry.IsBonus {
					p.QuantityPerSet = 0
				} else if entry.Quantity > 0 {
					p.QuantityPerSet = entry.Quantity
				} else {
					p.QuantityPerSet = 1
				}
			}
		}
	}

	if productsList[0].QuantityPerSet != 0 {
		t.Errorf("Secret Rare: got %d, want 0", productsList[0].QuantityPerSet)
	}
	if productsList[1].QuantityPerSet != 0 {
		t.Errorf("Quarter Century Secret Rare: got %d, want 0", productsList[1].QuantityPerSet)
	}
	if productsList[2].QuantityPerSet != 1 {
		t.Errorf("Ultra Rare: got %d, want 1", productsList[2].QuantityPerSet)
	}
}

func TestDeriveSetImageLarge(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "yugipedia thumb URL",
			input:    "//ms.yugipedia.com/thumb/0/0a/SR14-DeckEN.png/113px-SR14-DeckEN.png",
			expected: "//ms.yugipedia.com/0/0a/SR14-DeckEN.png",
		},
		{
			name:     "yugipedia URL without thumb",
			input:    "//ms.yugipedia.com/0/0a/SR14-DeckEN.png",
			expected: "//ms.yugipedia.com/0/0a/SR14-DeckEN.png",
		},
		{
			name:     "non-yugipedia URL with thumb",
			input:    "https://example.com/thumb/image.png",
			expected: "https://example.com/image.png",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveSetImageLarge(tt.input)
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDeriveSetImageSmall(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "yugipedia URL with pixel size",
			input:    "//ms.yugipedia.com/thumb/0/0a/SR14-DeckEN.png/113px-SR14-DeckEN.png",
			expected: "//ms.yugipedia.com/thumb/0/0a/SR14-DeckEN.png/257px-SR14-DeckEN.png",
		},
		{
			name:     "yugipedia URL without pixel size",
			input:    "//ms.yugipedia.com/0/0a/SR14-DeckEN.png",
			expected: "//ms.yugipedia.com/0/0a/SR14-DeckEN.png",
		},
		{
			name:     "non-yugipedia URL",
			input:    "https://example.com/113px-image.png",
			expected: "https://example.com/257px-image.png",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveSetImageSmall(tt.input)
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}
