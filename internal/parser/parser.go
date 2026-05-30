package parser

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Citation represents a patent citation entry.
type Citation struct {
	PublicationNumber string `json:"publication_number"`
	PriorityDate      string `json:"priority_date,omitempty"`
	PublicationDate   string `json:"publication_date,omitempty"`
	Assignee          string `json:"assignee,omitempty"`
	Title             string `json:"title,omitempty"`
	CitedByExaminer   bool   `json:"cited_by_examiner"`
	CitedByThirdParty bool   `json:"cited_by_third_party"`
}

// NonPatentCitation represents a non-patent literature citation.
type NonPatentCitation struct {
	Title             string `json:"title"`
	CitedByExaminer   bool   `json:"cited_by_examiner"`
	CitedByThirdParty bool   `json:"cited_by_third_party"`
}

// SimilarDocument represents a similar document entry.
type SimilarDocument struct {
	PublicationNumber string `json:"publication_number,omitempty"`
	PublicationDate   string `json:"publication_date,omitempty"`
	Title             string `json:"title,omitempty"`
	ScholarID         string `json:"scholar_id,omitempty"`
	Authors           string `json:"authors,omitempty"`
}

// FamilyApplication represents a family application entry.
type FamilyApplication struct {
	FilingDate        string `json:"filing_date,omitempty"`
	CountryCode       string `json:"country_code,omitempty"`
	ApplicationNumber string `json:"application_number,omitempty"`
	PatentNumber      string `json:"patent_number,omitempty"`
	LegalStatusCat    string `json:"legal_status_cat,omitempty"`
	LegalStatus       string `json:"legal_status,omitempty"`
}

// StructuredClaim holds a single parsed claim with its number and text.
// Type is "independent" or "dependent" when detectable from HTML claim-ref tags (US/EP);
// omitted for translated pages where markup is unavailable.
type StructuredClaim struct {
	Number    string   `json:"number"`
	Type      string   `json:"type,omitempty"`
	DependsOn []string `json:"depends_on,omitempty"`
	Text      string   `json:"text"`
}

// StructuredDescription holds a single description paragraph with its number, id, and text.
type StructuredDescription struct {
	Number string `json:"number"`
	ID     string `json:"id,omitempty"`
	Text   string `json:"text"`
}

// PatentData holds all parsed fields for a patent.
type PatentData struct {
	PublicationNumber       string              `json:"publication_number"`
	NumberWithoutKind       string              `json:"number_without_kind"`
	ApplicationNumber       string              `json:"application_number"`
	KindCode                string              `json:"kind_code"`
	Country                 string              `json:"country"`
	Title                   string              `json:"title"`
	Abstract                string              `json:"abstract"`
	Inventors               []string            `json:"inventors"`
	Assignee                string              `json:"assignee"`
	FilingDate              string              `json:"filing_date"`
	PublicationDate         string              `json:"publication_date"`
	CPCCodes                []string            `json:"cpc_codes"`
	Claims                  []string            `json:"claims"`
	Description             string              `json:"description"`
	PDFURL                  string              `json:"pdf_url"`
	PatentURL               string              `json:"patent_url"`
	BackwardCitations       []Citation          `json:"backward_citations"`
	BackwardCitationsFamily []Citation          `json:"backward_citations_family"`
	NonPatentCitations      []NonPatentCitation `json:"non_patent_citations"`
	ForwardCitations        []Citation          `json:"forward_citations"`
	ForwardCitationsFamily  []Citation          `json:"forward_citations_family"`
	SimilarDocuments        []SimilarDocument   `json:"similar_documents"`
	FamilyApplications      []FamilyApplication     `json:"family_applications"`
	ClaimsStructured        []StructuredClaim       `json:"claims_structured,omitempty"`
	DescriptionStructured   []StructuredDescription `json:"description_structured,omitempty"`
}

var (
	canonicalRE = regexp.MustCompile(
		`(?i)<link\s[^>]*rel="canonical"\s[^>]*href="https://patents\.google\.com/patent/([^/]+)/`)
	metaAppRE = regexp.MustCompile(
		`(?i)<meta\s[^>]*name="citation_patent_application_number"\s[^>]*content="([^"]+)"`)
	pubNumRE     = regexp.MustCompile(`^([A-Z]{2})(\d+)([A-Z]\d?)?$`)
	claimSpaceRE = regexp.MustCompile(`(\d+)\s+\.\s+`)
	multiSpaceRE = regexp.MustCompile(`\s{2,}`)
	cpcFullRE    = regexp.MustCompile(`^[A-H]\d{2}[A-Z]\d+/\d+`)

	figRE      = regexp.MustCompile(`\b(FIGS?|TABLES?)\.\s*(\d+)\s+([A-Z])\b`)
	refNumRE   = regexp.MustCompile(`(\d+)\s+([A-Z]{1,2})(?:[^a-z])`)
	spaceComRE = regexp.MustCompile(`(\w)\s+([,;])`)
)

var smartQuoteReplacer = strings.NewReplacer(
	"‘", "'", "’", "'",
	"‚", "'", "‛", "'",
	"“", "'", "”", "'",
	"„", "'", "‟", "'",
	"′", "'", "″", "'",
	"–", "-", "—", "-",
	"…", "...",
)

// ParseAll parses all fields from Google Patents HTML.
func ParseAll(html string) PatentData {
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))

	pubNumber, numberWithoutKind, kindCode := parsePubNumber(html)
	appNumber := parseAppNumber(html)

	country := ""
	if len(pubNumber) >= 2 && isAlpha(pubNumber[:2]) {
		country = pubNumber[:2]
	}

	inventors := parseInventors(doc)
	contributors := parseDCContributors(doc)

	pdfURL := ""
	if tag := doc.Find("a[itemprop='pdfLink']").First(); tag.Length() > 0 {
		pdfURL, _ = tag.Attr("href")
		pdfURL = strings.TrimSpace(pdfURL)
	}

	patentURL := ""
	if pubNumber != "" {
		patentURL = "https://patents.google.com/patent/" + pubNumber
	}

	return PatentData{
		PublicationNumber:       pubNumber,
		NumberWithoutKind:       numberWithoutKind,
		ApplicationNumber:       appNumber,
		KindCode:                kindCode,
		Country:                 country,
		Title:                   parseTitle(doc),
		Abstract:                parseAbstract(doc),
		Inventors:               inventors,
		Assignee:                parseAssignee(inventors, contributors),
		FilingDate:              dcDate(doc, "dateSubmitted"),
		PublicationDate:         parsePublicationDate(doc),
		CPCCodes:                parseCPCCodes(doc),
		Claims:                  parseClaims(doc),
		Description:             parseDescription(doc),
		PDFURL:                  pdfURL,
		PatentURL:               patentURL,
		BackwardCitations:       parseCitationRows(doc, "backwardReferencesOrig"),
		BackwardCitationsFamily: parseCitationRows(doc, "backwardReferencesFamily"),
		NonPatentCitations:      parseNonPatentCitations(doc),
		ForwardCitations:        parseCitationRows(doc, "forwardReferencesOrig"),
		ForwardCitationsFamily:  parseCitationRows(doc, "forwardReferencesFamily"),
		SimilarDocuments:        parseSimilarDocuments(doc),
		FamilyApplications:      parseFamilyApplications(doc),
		ClaimsStructured:        parseClaimsStructured(doc),
		DescriptionStructured:   parseDescriptionStructured(doc),
	}
}

func isAlpha(s string) bool {
	for _, c := range s {
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return true
}

// ── Number parsing ────────────────────────────────────────────────────────────

func parsePubNumber(html string) (pubNumber, numberWithoutKind, kindCode string) {
	m := canonicalRE.FindStringSubmatch(html)
	if m == nil {
		return
	}
	raw := strings.ToUpper(m[1])
	km := pubNumRE.FindStringSubmatch(raw)
	if km == nil {
		return
	}
	country, digits, kind := km[1], km[2], km[3]
	pubNumber = country + digits + kind
	numberWithoutKind = country + digits
	kindCode = kind
	return
}

func parseAppNumber(html string) string {
	m := metaAppRE.FindStringSubmatch(html)
	if m == nil {
		return ""
	}
	parts := strings.SplitN(m[1], ":", 2)
	if len(parts) != 2 {
		return ""
	}
	country := strings.ToUpper(strings.TrimSpace(parts[0]))
	rest := strings.TrimSpace(parts[1])
	if country == "PC" && strings.ToUpper(rest[:min(2, len(rest))]) == "T/" {
		return "PCT/" + rest[2:]
	}
	nonDigitRE := regexp.MustCompile(`[^0-9]`)
	digits := nonDigitRE.ReplaceAllString(rest, "")
	if country == "" || digits == "" {
		return ""
	}
	return country + digits
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ── Bibliographic fields ──────────────────────────────────────────────────────

func metaContent(doc *goquery.Document, name string) string {
	val, _ := doc.Find("meta[name='" + name + "']").First().Attr("content")
	return strings.TrimSpace(val)
}

func parseTitle(doc *goquery.Document) string {
	for _, name := range []string{"DC.title", "citation_title"} {
		if val := metaContent(doc, name); val != "" {
			return val
		}
	}
	if t := doc.Find("title").First(); t.Length() > 0 {
		text := strings.TrimSpace(t.Text())
		if idx := strings.Index(text, " - Google Patents"); idx >= 0 {
			return strings.TrimSpace(text[:idx])
		}
		return text
	}
	return ""
}

func parseAbstract(doc *goquery.Document) string {
	if section := doc.Find("[itemprop='abstract']").First(); section.Length() > 0 {
		target := section
		if inner := section.Find(".abstract").First(); inner.Length() > 0 {
			target = inner
		}
		// Translated (/en) pages: strip original-language spans then use meta as
		// the DOM abstract section retains only the "Translated from …" label.
		if target.Find("span.notranslate").Length() > 0 {
			if meta := strings.TrimSpace(metaContent(doc, "description")); meta != "" {
				return meta
			}
		}
		text := strings.TrimSpace(target.Text())
		if text != "" {
			return text
		}
	}
	if tag := doc.Find(".abstract").First(); tag.Length() > 0 {
		text := strings.TrimSpace(tag.Text())
		if text != "" {
			return text
		}
	}
	return strings.TrimSpace(metaContent(doc, "description"))
}

func parseInventors(doc *goquery.Document) []string {
	var result []string
	doc.Find("meta[name='citation_author']").Each(func(_ int, s *goquery.Selection) {
		if val, ok := s.Attr("content"); ok && strings.TrimSpace(val) != "" {
			result = append(result, strings.TrimSpace(val))
		}
	})
	if len(result) > 0 {
		return result
	}
	doc.Find("[itemprop='inventor']").Each(func(_ int, s *goquery.Selection) {
		if text := strings.TrimSpace(s.Text()); text != "" {
			result = append(result, text)
		}
	})
	return result
}

func parseDCContributors(doc *goquery.Document) []string {
	var result []string
	doc.Find("meta[name='DC.contributor']").Each(func(_ int, s *goquery.Selection) {
		if val, ok := s.Attr("content"); ok && strings.TrimSpace(val) != "" {
			result = append(result, strings.TrimSpace(val))
		}
	})
	return result
}

func parseAssignee(inventors, contributors []string) string {
	inventorSet := make(map[string]bool, len(inventors))
	for _, inv := range inventors {
		inventorSet[inv] = true
	}
	for _, name := range contributors {
		if !inventorSet[name] {
			return name
		}
	}
	return ""
}

// ── Classification ────────────────────────────────────────────────────────────

func parseCPCCodes(doc *goquery.Document) []string {
	seen := make(map[string]bool)
	var result []string
	doc.Find("li[itemprop='classifications']").Each(func(_ int, li *goquery.Selection) {
		if li.Find("meta[itemprop='IsCPC'][content='true']").Length() == 0 {
			return
		}
		code := strings.TrimSpace(li.Find("[itemprop='Code']").First().Text())
		if cpcFullRE.MatchString(code) && !seen[code] {
			seen[code] = true
			result = append(result, code)
		}
	})
	return result
}

// ── Claims / Description ──────────────────────────────────────────────────────

func parseClaims(doc *goquery.Document) []string {
	var container *goquery.Selection

	if div := doc.Find("div.claims").First(); div.Length() > 0 {
		container = div
	} else if section := doc.Find("[itemprop='claims']").First(); section.Length() > 0 {
		if ol := section.Find("ol.claims").First(); ol.Length() > 0 {
			container = ol
		}
	}
	if container == nil {
		// Translated (/en) pages use <claim num="N"> custom elements.
		return parseClaimsTranslated(doc)
	}

	var result []string
	container.Children().Each(func(_ int, child *goquery.Selection) {
		classes, _ := child.Attr("class")
		if !strings.Contains(classes, "claim") {
			return
		}
		text := strings.TrimSpace(child.Text())
		text = claimSpaceRE.ReplaceAllString(text, "$1. ")
		text = multiSpaceRE.ReplaceAllString(text, " ")
		text = strings.TrimSpace(text)
		if text == "" {
			return
		}
		if !regexp.MustCompile(`^\d+\.`).MatchString(text) {
			if inner := child.Find(".claim[num]").First(); inner.Length() > 0 {
				num, _ := inner.Attr("num")
				text = strings.TrimLeft(num, "0")
				if text == "" {
					text = "0"
				}
				text = text + ". " + strings.TrimSpace(child.Text())
			}
		}
		result = append(result, text)
	})
	return result
}

// parseClaimsTranslated handles <claim num="N"> custom elements in /en translated pages.
// Structure: <span class="notranslate"><span class="google-src-text">원문</span>English</span>
// Removing google-src-text leaves only the English translation inside notranslate.
func parseClaimsTranslated(doc *goquery.Document) []string {
	var result []string
	doc.Find("claim").Each(func(_ int, s *goquery.Selection) {
		cloned := s.Clone()
		cloned.Find("span.google-src-text").Remove()
		text := strings.TrimSpace(cloned.Text())
		text = multiSpaceRE.ReplaceAllString(text, " ")
		if text != "" {
			result = append(result, text)
		}
	})
	return result
}

func parseClaimsStructured(doc *goquery.Document) []StructuredClaim {
	var container *goquery.Selection
	if div := doc.Find("div.claims").First(); div.Length() > 0 {
		container = div
	} else if section := doc.Find("[itemprop='claims']").First(); section.Length() > 0 {
		if ol := section.Find("ol.claims").First(); ol.Length() > 0 {
			container = ol
		}
	}
	if container == nil {
		return parseClaimsStructuredTranslated(doc)
	}

	var result []StructuredClaim
	container.Children().Each(func(_ int, child *goquery.Selection) {
		classes, _ := child.Attr("class")
		if !strings.Contains(classes, "claim") {
			return
		}
		// US patent HTML wraps each claim: outer <div class="claim"> (no num)
		// contains inner <div id="CLM-XXXXX" num="XXXXX" class="claim">.
		// Prefer the num attribute from the inner element when the outer lacks it.
		num, _ := child.Attr("num")
		target := child
		if num == "" {
			if inner := child.Find("[num]").First(); inner.Length() > 0 {
				num, _ = inner.Attr("num")
				target = inner
			}
		}
		num = strings.TrimLeft(num, "0")
		if num == "" {
			num = "0"
		}
		text := strings.TrimSpace(child.Text())
		text = claimSpaceRE.ReplaceAllString(text, "$1. ")
		text = multiSpaceRE.ReplaceAllString(text, " ")
		text = strings.TrimSpace(text)
		if text == "" {
			return
		}

		// Detect independent/dependent via <claim-ref idref="CLM-XXXXX"> tags.
		// Only present in patent-office HTML (US/EP); absent in translated pages.
		claimType := "independent"
		var dependsOn []string
		seen := make(map[string]bool)
		target.Find("claim-ref[idref]").Each(func(_ int, ref *goquery.Selection) {
			idref, _ := ref.Attr("idref")
			n := strings.TrimLeft(strings.TrimPrefix(strings.ToUpper(idref), "CLM-"), "0")
			if n == "" {
				n = "0"
			}
			if !seen[n] {
				seen[n] = true
				dependsOn = append(dependsOn, n)
			}
		})
		if len(dependsOn) > 0 {
			claimType = "dependent"
		}

		sc := StructuredClaim{Number: num, Type: claimType, Text: text}
		if len(dependsOn) > 0 {
			sc.DependsOn = dependsOn
		}
		result = append(result, sc)
	})
	return result
}

func parseClaimsStructuredTranslated(doc *goquery.Document) []StructuredClaim {
	var result []StructuredClaim
	doc.Find("claim").Each(func(_ int, s *goquery.Selection) {
		num, _ := s.Attr("num")
		num = strings.TrimLeft(num, "0")
		if num == "" {
			num = "0"
		}
		cloned := s.Clone()
		cloned.Find("span.google-src-text").Remove()
		text := strings.TrimSpace(cloned.Text())
		text = multiSpaceRE.ReplaceAllString(text, " ")
		if text != "" {
			result = append(result, StructuredClaim{Number: num, Text: text})
		}
	})
	return result
}

func parseDescriptionStructured(doc *goquery.Document) []StructuredDescription {
	var result []StructuredDescription
	doc.Find("div.description-paragraph[num]").Each(func(_ int, s *goquery.Selection) {
		num, _ := s.Attr("num")
		id, _ := s.Attr("id")
		text := strings.TrimSpace(s.Text())
		text = multiSpaceRE.ReplaceAllString(text, " ")
		if text == "" {
			return
		}
		result = append(result, StructuredDescription{
			Number: strings.TrimLeft(num, "0"),
			ID:     id,
			Text:   text,
		})
	})
	if len(result) == 0 {
		return parseDescriptionStructuredTranslated(doc)
	}
	return result
}

// parseDescriptionStructuredTranslated handles <p> tags in /en translated pages.
// Paragraphs are numbered sequentially; <span class="notranslate"> (original language) is stripped.
func parseDescriptionStructuredTranslated(doc *goquery.Document) []StructuredDescription {
	section := doc.Find("[itemprop='description']").First()
	if section.Length() == 0 {
		return nil
	}
	cloned := section.Clone()
	cloned.Find("span.notranslate").Remove()
	var result []StructuredDescription
	i := 0
	cloned.Find("p").Each(func(_ int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		text = multiSpaceRE.ReplaceAllString(text, " ")
		if text == "" {
			return
		}
		i++
		result = append(result, StructuredDescription{
			Number: strconv.Itoa(i),
			Text:   text,
		})
	})
	return result
}

func parseDescription(doc *goquery.Document) string {
	var tag *goquery.Selection
	if t := doc.Find(".description").First(); t.Length() > 0 {
		tag = t
	} else if t := doc.Find("[itemprop='description']").First(); t.Length() > 0 {
		tag = t
	}
	if tag == nil {
		return ""
	}
	// Translated (/en) pages embed original-language text in <span class="notranslate">.
	// Clone and strip those spans so only the English translation remains.
	if tag.Find("span.notranslate").Length() > 0 {
		tag = tag.Clone()
		tag.Find("span.notranslate").Remove()
	}
	text := strings.TrimSpace(tag.Text())
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = figRE.ReplaceAllString(text, "$1. $2$3")
	text = refNumRE.ReplaceAllStringFunc(text, func(s string) string {
		m := refNumRE.FindStringSubmatch(s)
		if m != nil {
			return m[1] + m[2]
		}
		return s
	})
	text = spaceComRE.ReplaceAllString(text, "$1$2")
	text = smartQuoteReplacer.Replace(text)
	return text
}

// ── Dates ─────────────────────────────────────────────────────────────────────

func dcDate(doc *goquery.Document, scheme string) string {
	val := ""
	doc.Find("meta[name='DC.date']").Each(func(_ int, s *goquery.Selection) {
		if sc, _ := s.Attr("scheme"); sc == scheme {
			val, _ = s.Attr("content")
			val = strings.TrimSpace(val)
		}
	})
	return val
}

func parsePublicationDate(doc *goquery.Document) string {
	if val := dcDate(doc, "issue"); val != "" {
		return val
	}
	result := ""
	doc.Find("dd").Each(func(_ int, dd *goquery.Selection) {
		if result != "" {
			return
		}
		timeTag := dd.Find("time[itemprop='publicationDate']").First()
		if timeTag.Length() == 0 {
			return
		}
		dt, _ := timeTag.Attr("datetime")
		dt = strings.TrimSpace(dt)
		if dt != "" {
			result = dt
		} else {
			result = strings.TrimSpace(timeTag.Text())
		}
	})
	return result
}

// ── Citations ─────────────────────────────────────────────────────────────────

func citationMarker(tr *goquery.Selection) (examiner, thirdParty bool) {
	examiner = tr.Find("[itemprop='examinerCited']").Length() > 0
	thirdParty = tr.Find("[itemprop='thirdPartyCited']").Length() > 0
	if !thirdParty {
		tr.Find("span").Each(func(_ int, span *goquery.Selection) {
			if strings.TrimSpace(span.Text()) == "†" {
				thirdParty = true
			}
		})
	}
	return
}

func parseCitationRows(doc *goquery.Document, itempropValue string) []Citation {
	var result []Citation
	doc.Find("tr[itemprop='" + itempropValue + "']").Each(func(_ int, tr *goquery.Selection) {
		pub := strings.TrimSpace(tr.Find("[itemprop='publicationNumber']").First().Text())
		priority := strings.TrimSpace(tr.Find("[itemprop='priorityDate']").First().Text())
		pubDate := strings.TrimSpace(tr.Find("[itemprop='publicationDate']").First().Text())
		assignee := strings.TrimSpace(tr.Find("[itemprop='assigneeOriginal']").First().Text())
		title := strings.TrimSpace(tr.Find("[itemprop='title']").First().Text())
		examiner, thirdParty := citationMarker(tr)

		c := Citation{
			PublicationNumber: pub,
			CitedByExaminer:   examiner,
			CitedByThirdParty: thirdParty,
		}
		if priority != "" {
			c.PriorityDate = priority
		}
		if pubDate != "" {
			c.PublicationDate = pubDate
		}
		if assignee != "" {
			c.Assignee = assignee
		}
		if title != "" {
			c.Title = title
		}
		result = append(result, c)
	})
	return result
}

func parseNonPatentCitations(doc *goquery.Document) []NonPatentCitation {
	var result []NonPatentCitation
	doc.Find("tr[itemprop='detailedNonPatentLiterature']").Each(func(_ int, tr *goquery.Selection) {
		title := strings.TrimSpace(tr.Find("[itemprop='title']").First().Text())
		title = strings.ReplaceAll(title, `"`, "")
		if title == "" {
			return
		}
		examiner, thirdParty := citationMarker(tr)
		result = append(result, NonPatentCitation{
			Title:             title,
			CitedByExaminer:   examiner,
			CitedByThirdParty: thirdParty,
		})
	})
	return result
}

func parseSimilarDocuments(doc *goquery.Document) []SimilarDocument {
	var result []SimilarDocument
	doc.Find("tr[itemprop='similarDocuments']").Each(func(_ int, tr *goquery.Selection) {
		pubDate := strings.TrimSpace(tr.Find("[itemprop='publicationDate']").First().Text())
		title := strings.TrimSpace(tr.Find("[itemprop='title']").First().Text())

		if pubTag := tr.Find("[itemprop='publicationNumber']").First(); pubTag.Length() > 0 {
			pub := strings.TrimSpace(pubTag.Text())
			d := SimilarDocument{PublicationNumber: pub}
			if pubDate != "" {
				d.PublicationDate = pubDate
			}
			if title != "" {
				d.Title = title
			}
			result = append(result, d)
		} else if scholarTag := tr.Find("[itemprop='scholarID']").First(); scholarTag.Length() > 0 {
			scholarID, _ := scholarTag.Attr("content")
			authors := strings.TrimSpace(tr.Find("[itemprop='scholarAuthors']").First().Text())
			d := SimilarDocument{ScholarID: scholarID}
			if authors != "" {
				d.Authors = authors
			}
			if pubDate != "" {
				d.PublicationDate = pubDate
			}
			if title != "" {
				d.Title = title
			}
			result = append(result, d)
		}
	})
	return result
}

func parseFamilyApplications(doc *goquery.Document) []FamilyApplication {
	var result []FamilyApplication
	doc.Find("li[itemprop='application']").Each(func(_ int, li *goquery.Selection) {
		a := FamilyApplication{}
		if t := strings.TrimSpace(li.Find("[itemprop='filingDate']").First().Text()); t != "" {
			a.FilingDate = t
		}
		if t := strings.TrimSpace(li.Find("[itemprop='countryCode']").First().Text()); t != "" {
			a.CountryCode = t
		}
		if t := strings.TrimSpace(li.Find("[itemprop='applicationNumber']").First().Text()); t != "" {
			a.ApplicationNumber = t
		}
		if t := strings.TrimSpace(li.Find("[itemprop='legalStatusCat']").First().Text()); t != "" {
			a.LegalStatusCat = t
		}
		if t := strings.TrimSpace(li.Find("[itemprop='legalStatus']").First().Text()); t != "" {
			a.LegalStatus = t
		}
		if link := li.Find("a[href]").First(); link.Length() > 0 {
			href, _ := link.Attr("href")
			parts := strings.Split(strings.Trim(href, "/"), "/")
			if len(parts) >= 2 {
				a.PatentNumber = parts[1]
			}
		}
		result = append(result, a)
	})
	return result
}

var imageURLRE = regexp.MustCompile(
	`https://patentimages\.storage\.googleapis\.com` +
		`/[a-f0-9]{2}/[a-f0-9]{2}/[a-f0-9]{2}/[a-f0-9]+` +
		`/([^/"]+\.png)`)

// ParseImageURLs extracts high-resolution figure image URLs from HTML.
// For filenames appearing more than once, the last URL (highest resolution) is used.
// Filenames appearing only once (thumbnails) are excluded.
func ParseImageURLs(html string) []string {
	type entry struct {
		urls []string
		idx  int
	}
	seen := make(map[string]*entry)
	var order []string

	for _, m := range imageURLRE.FindAllStringSubmatch(html, -1) {
		fullURL, filename := m[0], m[1]
		if e, ok := seen[filename]; ok {
			e.urls = append(e.urls, fullURL)
		} else {
			seen[filename] = &entry{urls: []string{fullURL}, idx: len(order)}
			order = append(order, filename)
		}
	}

	var result []string
	for _, filename := range order {
		e := seen[filename]
		if len(e.urls) < 2 {
			continue
		}
		result = append(result, e.urls[len(e.urls)-1])
	}
	return result
}
