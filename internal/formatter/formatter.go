package formatter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/area99/patent-cli/internal/parser"
)

// FieldOrder defines the canonical output order for all fields.
var FieldOrder = []string{
	"publication_number",
	"number_without_kind",
	"application_number",
	"kind_code",
	"country",
	"title",
	"abstract",
	"inventors",
	"assignee",
	"filing_date",
	"publication_date",
	"cpc_codes",
	"claims",
	"description",
	"patent_url",
	"backward_citations",
	"backward_citations_family",
	"non_patent_citations",
	"forward_citations",
	"forward_citations_family",
	"similar_documents",
	"family_applications",
}

var labels = map[string]string{
	"publication_number":       "Publication Number",
	"number_without_kind":      "Number (no kind)",
	"application_number":       "Application Number",
	"kind_code":                "Kind Code",
	"country":                  "Country",
	"title":                    "Title",
	"abstract":                 "Abstract",
	"inventors":                "Inventors",
	"assignee":                 "Assignee",
	"filing_date":              "Filing Date",
	"publication_date":         "Publication Date",
	"cpc_codes":                "CPC Codes",
	"claims":                   "Claims",
	"claims_structured":        "Claims (Structured)",
	"description":              "Description",
	"description_structured":   "Description (Structured)",
	"patent_url":               "Patent URL",
	"backward_citations":       "Backward Citations",
	"backward_citations_family": "Backward Citations (Family)",
	"non_patent_citations":     "Non-Patent Citations",
	"forward_citations":        "Forward Citations",
	"forward_citations_family": "Forward Citations (Family)",
	"similar_documents":        "Similar Documents",
	"family_applications":      "Family Applications (Worldwide)",
}

// StructuredFieldNames lists the opt-in structured fields not included in default output.
var StructuredFieldNames = []string{"claims_structured", "description_structured"}

// DataMap is an ordered map of field name → value (interface{} for JSON compat).
type DataMap struct {
	keys   []string
	values map[string]interface{}
}

func newDataMap() *DataMap {
	return &DataMap{values: make(map[string]interface{})}
}

func (d *DataMap) set(k string, v interface{}) {
	if _, exists := d.values[k]; !exists {
		d.keys = append(d.keys, k)
	}
	d.values[k] = v
}

func (d *DataMap) get(k string) (interface{}, bool) {
	v, ok := d.values[k]
	return v, ok
}

// Get is the exported version of get, used by cmd/gp-cli.
func (d *DataMap) Get(k string) (interface{}, bool) {
	return d.get(k)
}

// LabelFor returns the human-readable label for a field key.
func LabelFor(k string) string {
	return labelFor(k)
}

func (d *DataMap) MarshalJSON() ([]byte, error) {
	// preserve insertion order
	var sb strings.Builder
	sb.WriteString("{")
	for i, k := range d.keys {
		kb, _ := json.Marshal(k)
		vb, _ := json.Marshal(d.values[k])
		if i > 0 {
			sb.WriteString(",")
		}
		sb.Write(kb)
		sb.WriteString(":")
		sb.Write(vb)
	}
	sb.WriteString("}")
	return []byte(sb.String()), nil
}

// ToDataMap converts PatentData to an ordered DataMap.
func ToDataMap(data parser.PatentData) *DataMap {
	dm := newDataMap()
	dm.set("publication_number", nilIfEmpty(data.PublicationNumber))
	dm.set("number_without_kind", nilIfEmpty(data.NumberWithoutKind))
	dm.set("application_number", nilIfEmpty(data.ApplicationNumber))
	dm.set("kind_code", nilIfEmpty(data.KindCode))
	dm.set("country", nilIfEmpty(data.Country))
	dm.set("title", nilIfEmpty(data.Title))
	dm.set("abstract", nilIfEmpty(data.Abstract))
	dm.set("inventors", strSliceOrNil(data.Inventors))
	dm.set("assignee", nilIfEmpty(data.Assignee))
	dm.set("filing_date", nilIfEmpty(data.FilingDate))
	dm.set("publication_date", nilIfEmpty(data.PublicationDate))
	dm.set("cpc_codes", strSliceOrNil(data.CPCCodes))
	dm.set("claims", strSliceOrNil(data.Claims))
	dm.set("description", nilIfEmpty(data.Description))
	dm.set("patent_url", nilIfEmpty(data.PatentURL))
	dm.set("backward_citations", citationsOrNil(data.BackwardCitations))
	dm.set("backward_citations_family", citationsOrNil(data.BackwardCitationsFamily))
	dm.set("non_patent_citations", nonPatentOrNil(data.NonPatentCitations))
	dm.set("forward_citations", citationsOrNil(data.ForwardCitations))
	dm.set("forward_citations_family", citationsOrNil(data.ForwardCitationsFamily))
	dm.set("similar_documents", similarOrNil(data.SimilarDocuments))
	dm.set("family_applications", familyOrNil(data.FamilyApplications))
	return dm
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func strSliceOrNil(s []string) interface{} {
	if len(s) == 0 {
		return []string{}
	}
	return s
}

func citationsOrNil(s []parser.Citation) interface{} {
	if s == nil {
		return []parser.Citation{}
	}
	return s
}

func nonPatentOrNil(s []parser.NonPatentCitation) interface{} {
	if s == nil {
		return []parser.NonPatentCitation{}
	}
	return s
}

func similarOrNil(s []parser.SimilarDocument) interface{} {
	if s == nil {
		return []parser.SimilarDocument{}
	}
	return s
}

func familyOrNil(s []parser.FamilyApplication) interface{} {
	if s == nil {
		return []parser.FamilyApplication{}
	}
	return s
}

// AddStructuredFields appends claims_structured and description_structured to dm.
// Called only when the user explicitly requests these fields via --field / --fields.
func AddStructuredFields(dm *DataMap, data parser.PatentData) {
	if len(data.ClaimsStructured) > 0 {
		dm.set("claims_structured", data.ClaimsStructured)
	} else {
		dm.set("claims_structured", []parser.StructuredClaim{})
	}
	if len(data.DescriptionStructured) > 0 {
		dm.set("description_structured", data.DescriptionStructured)
	} else {
		dm.set("description_structured", []parser.StructuredDescription{})
	}
}

// SelectFields returns a new DataMap with only the requested fields.
func SelectFields(dm *DataMap, fields []string) *DataMap {
	if len(fields) == 0 {
		return dm
	}
	out := newDataMap()
	for _, f := range fields {
		if v, ok := dm.get(f); ok {
			out.set(f, v)
		}
	}
	return out
}

// Render converts a DataMap to the requested format string.
// For the json format, minify controls whether output is compact or indented.
func Render(dm *DataMap, fmt_ string, minify bool) string {
	switch fmt_ {
	case "json":
		return renderJSON(dm, minify)
	case "text":
		return renderText(dm)
	case "tsv":
		return renderTSV(dm)
	}
	return ""
}

// renderJSON wraps the DataMap in a {ok, results} envelope.
func renderJSON(dm *DataMap, minify bool) string {
	type envelope struct {
		OK      bool            `json:"ok"`
		Results json.RawMessage `json:"results"`
	}
	inner, _ := dm.MarshalJSON()
	env := envelope{OK: true, Results: json.RawMessage(inner)}
	var b []byte
	if minify {
		b, _ = json.Marshal(env)
	} else {
		b, _ = json.MarshalIndent(env, "", "  ")
	}
	return string(b)
}

func renderText(dm *DataMap) string {
	maxLabel := 20
	for _, k := range dm.keys {
		if l := len(labelFor(k)); l > maxLabel {
			maxLabel = l
		}
	}

	var lines []string
	// print in FIELD_ORDER first, then any extra keys
	inOrder := make(map[string]bool)
	for _, k := range FieldOrder {
		if _, ok := dm.get(k); ok {
			inOrder[k] = true
			label := labelFor(k)
			v, _ := dm.get(k)
			lines = append(lines, fmt.Sprintf("%-*s : %s", maxLabel, label, serializeValue(v, k)))
		}
	}
	for _, k := range dm.keys {
		if !inOrder[k] {
			label := labelFor(k)
			v, _ := dm.get(k)
			lines = append(lines, fmt.Sprintf("%-*s : %s", maxLabel, label, serializeValue(v, k)))
		}
	}
	return strings.Join(lines, "\n")
}

func renderTSV(dm *DataMap) string {
	var keys []string
	inOrder := make(map[string]bool)
	for _, k := range FieldOrder {
		if _, ok := dm.get(k); ok {
			keys = append(keys, k)
			inOrder[k] = true
		}
	}
	for _, k := range dm.keys {
		if !inOrder[k] {
			keys = append(keys, k)
		}
	}
	var header, row []string
	for _, k := range keys {
		header = append(header, k)
		v, _ := dm.get(k)
		row = append(row, serializeValue(v, k))
	}
	return strings.Join(header, "\t") + "\n" + strings.Join(row, "\t")
}

func labelFor(k string) string {
	if l, ok := labels[k]; ok {
		return l
	}
	return k
}

var citationFields = map[string]bool{
	"backward_citations": true, "backward_citations_family": true,
	"forward_citations": true, "forward_citations_family": true,
}
var nonPatentFields = map[string]bool{"non_patent_citations": true}
var similarFields = map[string]bool{"similar_documents": true}
var familyFields = map[string]bool{"family_applications": true}

func serializeValue(v interface{}, fieldKey string) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []string:
		return strings.Join(val, "; ")
	case []interface{}:
		var parts []string
		for _, item := range val {
			if m, ok := item.(map[string]interface{}); ok {
				parts = append(parts, serializeMap(m, fieldKey))
			} else {
				parts = append(parts, fmt.Sprintf("%v", item))
			}
		}
		return strings.Join(parts, "; ")
	case []parser.Citation:
		var parts []string
		for _, c := range val {
			parts = append(parts, serializeCitation(c))
		}
		return strings.Join(parts, "; ")
	case []parser.NonPatentCitation:
		var parts []string
		for _, c := range val {
			parts = append(parts, serializeNonPatent(c))
		}
		return strings.Join(parts, "; ")
	case []parser.SimilarDocument:
		var parts []string
		for _, c := range val {
			parts = append(parts, serializeSimilar(c))
		}
		return strings.Join(parts, "; ")
	case []parser.FamilyApplication:
		var parts []string
		for _, c := range val {
			parts = append(parts, serializeFamilyApp(c))
		}
		return strings.Join(parts, "; ")
	case []parser.StructuredClaim:
		var parts []string
		for _, c := range val {
			parts = append(parts, c.Number+". "+c.Text)
		}
		return strings.Join(parts, "; ")
	case []parser.StructuredDescription:
		var parts []string
		for _, p := range val {
			parts = append(parts, "["+p.Number+"] "+p.Text)
		}
		return strings.Join(parts, "; ")
	}
	return fmt.Sprintf("%v", v)
}

func serializeMap(m map[string]interface{}, fieldKey string) string {
	if citationFields[fieldKey] {
		pub, _ := m["publication_number"].(string)
		marker := ""
		if b, _ := m["cited_by_examiner"].(bool); b {
			marker += "*"
		}
		if b, _ := m["cited_by_third_party"].(bool); b {
			marker += "†"
		}
		if marker != "" {
			pub += marker
		}
		parts := []string{pub}
		priority, _ := m["priority_date"].(string)
		pubDate, _ := m["publication_date"].(string)
		if priority != "" || pubDate != "" {
			parts = append(parts, "("+priority+" / "+pubDate+")")
		}
		if a, _ := m["assignee"].(string); a != "" {
			parts = append(parts, a)
		}
		if t, _ := m["title"].(string); t != "" {
			parts = append(parts, t)
		}
		return strings.Join(parts, " | ")
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func serializeCitation(c parser.Citation) string {
	pub := c.PublicationNumber
	marker := ""
	if c.CitedByExaminer {
		marker += "*"
	}
	if c.CitedByThirdParty {
		marker += "†"
	}
	if marker != "" {
		pub += marker
	}
	parts := []string{pub}
	if c.PriorityDate != "" || c.PublicationDate != "" {
		parts = append(parts, "("+c.PriorityDate+" / "+c.PublicationDate+")")
	}
	if c.Assignee != "" {
		parts = append(parts, c.Assignee)
	}
	if c.Title != "" {
		parts = append(parts, c.Title)
	}
	return strings.Join(parts, " | ")
}

func serializeNonPatent(c parser.NonPatentCitation) string {
	marker := ""
	if c.CitedByExaminer {
		marker += "*"
	}
	if c.CitedByThirdParty {
		marker += "†"
	}
	if marker != "" {
		return "[" + marker + "] " + c.Title
	}
	return c.Title
}

func serializeSimilar(c parser.SimilarDocument) string {
	pub := c.PublicationNumber
	if pub == "" {
		pub = c.ScholarID
	}
	parts := []string{pub}
	if c.PublicationDate != "" {
		parts = append(parts, "("+c.PublicationDate+")")
	}
	if c.Title != "" {
		parts = append(parts, c.Title)
	}
	return strings.Join(parts, " | ")
}

func serializeFamilyApp(a parser.FamilyApplication) string {
	var parts []string
	if a.PatentNumber != "" {
		parts = append(parts, a.PatentNumber)
	}
	if a.CountryCode != "" {
		parts = append(parts, a.CountryCode)
	}
	if a.ApplicationNumber != "" {
		parts = append(parts, a.ApplicationNumber)
	}
	if a.FilingDate != "" {
		parts = append(parts, a.FilingDate)
	}
	status := strings.TrimSpace(strings.TrimRight(a.LegalStatusCat+"/"+a.LegalStatus, "/"))
	if status != "/" && status != "" {
		parts = append(parts, status)
	}
	return strings.Join(parts, " | ")
}

// PrintField prints a single field value (--field option).
func PrintField(v interface{}) {
	if v == nil {
		return
	}
	switch val := v.(type) {
	case string:
		fmt.Println(val)
	case []string:
		for _, s := range val {
			fmt.Println(s)
		}
	case []parser.Citation:
		b, _ := json.MarshalIndent(val, "", "  ")
		fmt.Println(string(b))
	case []parser.NonPatentCitation:
		b, _ := json.MarshalIndent(val, "", "  ")
		fmt.Println(string(b))
	case []parser.SimilarDocument:
		b, _ := json.MarshalIndent(val, "", "  ")
		fmt.Println(string(b))
	case []parser.FamilyApplication:
		b, _ := json.MarshalIndent(val, "", "  ")
		fmt.Println(string(b))
	default:
		b, _ := json.MarshalIndent(val, "", "  ")
		fmt.Println(string(b))
	}
}

var fmtExt = map[string]string{
	"json": ".json",
	"text": ".txt",
	"tsv":  ".tsv",
}

// SaveToDir writes rendered output to outputDir/<patentNumber><ext>.
func SaveToDir(dm *DataMap, fmt_, outputDir, patentNumber string) (string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", err
	}
	ext := fmtExt[fmt_]
	if ext == "" {
		ext = ".json"
	}
	path := filepath.Join(outputDir, patentNumber+ext)

	content := Render(dm, fmt_, false)

	// Use UTF-8 BOM for text/tsv (Excel/Notepad compatibility)
	var data []byte
	if fmt_ == "text" || fmt_ == "tsv" {
		data = append([]byte{0xEF, 0xBB, 0xBF}, []byte(content)...)
	} else {
		data = []byte(content)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}
	return path, nil
}

// PrintErrorJSON writes a structured JSON error envelope to stdout.
func PrintErrorJSON(errType, message string) {
	type errInfo struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	}
	type envelope struct {
		OK    bool    `json:"ok"`
		Error errInfo `json:"error"`
	}
	b, _ := json.MarshalIndent(envelope{
		OK:    false,
		Error: errInfo{Type: errType, Message: message},
	}, "", "  ")
	fmt.Println(string(b))
}
