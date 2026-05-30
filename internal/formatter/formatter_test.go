package formatter

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/area99/patent-cli/internal/parser"
)

func sampleDataMap() *DataMap {
	data := parser.PatentData{
		PublicationNumber: "US8725880B2",
		Title:             "Test Patent Title",
		Assignee:          "ACME Corp",
		FilingDate:        "2010-09-20",
	}
	return ToDataMap(data)
}

func TestRender_JSON_Envelope(t *testing.T) {
	dm := sampleDataMap()
	out := Render(dm, "json", false, false)

	var env struct {
		OK      bool            `json:"ok"`
		Results json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}
	if !env.OK {
		t.Error("ok should be true")
	}
	if len(env.Results) == 0 {
		t.Error("results should not be empty")
	}
}

func TestRender_JSON_ResultsFields(t *testing.T) {
	dm := sampleDataMap()
	out := Render(dm, "json", false, false)

	var env struct {
		Results map[string]interface{} `json:"results"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if env.Results["title"] != "Test Patent Title" {
		t.Errorf("results.title = %v, want %q", env.Results["title"], "Test Patent Title")
	}
	if env.Results["assignee"] != "ACME Corp" {
		t.Errorf("results.assignee = %v, want %q", env.Results["assignee"], "ACME Corp")
	}
}

func TestRender_JSON_Minify(t *testing.T) {
	dm := sampleDataMap()
	compact := Render(dm, "json", true, false)
	indented := Render(dm, "json", false, false)

	if strings.Contains(compact, "\n") {
		t.Error("minified output should not contain newlines")
	}
	if len(compact) >= len(indented) {
		t.Error("minified output should be shorter than indented")
	}
	// both must be valid JSON
	if !json.Valid([]byte(compact)) {
		t.Error("minified output is not valid JSON")
	}
}

func TestRender_Text_NoEnvelope(t *testing.T) {
	dm := sampleDataMap()
	out := Render(dm, "text", false, false)

	if strings.HasPrefix(out, "{") {
		t.Error("text output should not start with '{' (no JSON envelope)")
	}
	if !strings.Contains(out, "Test Patent Title") {
		t.Error("text output should contain the title")
	}
}

func TestRender_TSV_HasHeader(t *testing.T) {
	dm := sampleDataMap()
	out := Render(dm, "tsv", false, false)

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatalf("TSV output should have at least a header and data row, got %d lines", len(lines))
	}
	if !strings.Contains(lines[0], "publication_number") {
		t.Error("TSV header should contain 'publication_number'")
	}
}

func TestSelectFields(t *testing.T) {
	dm := sampleDataMap()
	filtered := SelectFields(dm, []string{"title", "assignee"})

	if _, ok := filtered.Get("title"); !ok {
		t.Error("filtered map should contain 'title'")
	}
	if _, ok := filtered.Get("assignee"); !ok {
		t.Error("filtered map should contain 'assignee'")
	}
	if _, ok := filtered.Get("filing_date"); ok {
		t.Error("filtered map should not contain 'filing_date'")
	}
}

func TestSelectFields_WarningsCarriedOver(t *testing.T) {
	dm := sampleDataMap()
	dm.warnings = []Warning{{Field: "claims_structured", Code: "TRANSLATED_PAGE_NO_TYPE_INFO", Message: "test"}}

	filtered := SelectFields(dm, []string{"title"})
	if len(filtered.warnings) != 1 || filtered.warnings[0].Code != "TRANSLATED_PAGE_NO_TYPE_INFO" {
		t.Error("SelectFields must carry warnings from source DataMap to output DataMap")
	}
}

func TestWarnings_NoTypeInfo(t *testing.T) {
	// Translated page: claims present but none have Type set
	claims := []parser.StructuredClaim{
		{Number: "1", Text: "A method comprising doing something substantial enough to be a real claim."},
		{Number: "2", Text: "The method of claim 1, further comprising another step that is quite long."},
	}
	warnings := claimsStructuredWarnings(claims)

	found := false
	for _, w := range warnings {
		if w.Code == "TRANSLATED_PAGE_NO_TYPE_INFO" {
			found = true
		}
	}
	if !found {
		t.Error("expected TRANSLATED_PAGE_NO_TYPE_INFO warning for claims without type")
	}
}

func TestWarnings_WithTypeInfo_NoWarning(t *testing.T) {
	// Non-translated page: claims have Type set → no warning
	claims := []parser.StructuredClaim{
		{Number: "1", Type: "independent", Text: "A method comprising doing something substantial enough."},
		{Number: "2", Type: "dependent", DependsOn: []string{"1"}, Text: "The method of claim 1, further comprising another step."},
	}
	warnings := claimsStructuredWarnings(claims)

	for _, w := range warnings {
		if w.Code == "TRANSLATED_PAGE_NO_TYPE_INFO" {
			t.Error("unexpected TRANSLATED_PAGE_NO_TYPE_INFO warning when type info is present")
		}
	}
}

func TestWarnings_SuspiciouslyShortText(t *testing.T) {
	claims := []parser.StructuredClaim{
		{Number: "1", Text: "delete"},
		{Number: "2", Text: "A method comprising a substantially long claim text that is fine."},
	}
	warnings := claimsStructuredWarnings(claims)

	found := false
	for _, w := range warnings {
		if w.Code == "SUSPICIOUSLY_SHORT_CLAIM_TEXT" {
			found = true
			if !strings.Contains(w.Message, "1") {
				t.Error("warning message should mention claim 1")
			}
		}
	}
	if !found {
		t.Error("expected SUSPICIOUSLY_SHORT_CLAIM_TEXT warning for claim with 'delete' text")
	}
}

func TestWarnings_AppearsInJSONEnvelope(t *testing.T) {
	data := parser.PatentData{
		PublicationNumber: "KR102345678B1",
		ClaimsStructured: []parser.StructuredClaim{
			{Number: "1", Text: "delete"},
			{Number: "2", Text: "A device comprising components as described in claim 1 with sufficient length."},
		},
	}
	dm := ToDataMap(data)
	AddStructuredFields(dm, data)

	out := Render(dm, "json", false, false)

	var env struct {
		OK       bool            `json:"ok"`
		Results  json.RawMessage `json:"results"`
		Warnings []Warning       `json:"_warnings"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}
	if len(env.Warnings) == 0 {
		t.Error("expected _warnings in JSON envelope when claims have quality issues")
	}
}

func TestWarnings_AbsentInTextOutput(t *testing.T) {
	data := parser.PatentData{
		PublicationNumber: "KR102345678B1",
		ClaimsStructured: []parser.StructuredClaim{
			{Number: "1", Text: "delete"},
		},
	}
	dm := ToDataMap(data)
	AddStructuredFields(dm, data)

	out := Render(dm, "text", false, false)
	if strings.Contains(out, "_warnings") {
		t.Error("_warnings should not appear in text output")
	}
}

func TestWarnings_AbsentWhenNoIssues(t *testing.T) {
	data := parser.PatentData{
		PublicationNumber: "US8725880B2",
		ClaimsStructured: []parser.StructuredClaim{
			{Number: "1", Type: "independent", Text: "A method comprising a substantially long and well-formed claim text."},
		},
	}
	dm := ToDataMap(data)
	AddStructuredFields(dm, data)

	out := Render(dm, "json", false, false)

	var env struct {
		Warnings *[]Warning `json:"_warnings"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if env.Warnings != nil {
		t.Errorf("_warnings should be absent when no issues detected, got: %v", *env.Warnings)
	}
}

// ── AddStructuredFields ───────────────────────────────────────────────────────

func TestAddStructuredFields_PopulatesDataMap(t *testing.T) {
	data := parser.PatentData{
		PublicationNumber: "US8725880B2",
		ClaimsStructured: []parser.StructuredClaim{
			{Number: "1", Type: "independent", Text: "A method comprising a novel step."},
			{Number: "2", Type: "dependent", DependsOn: []string{"1"}, Text: "The method of claim 1, further comprising."},
		},
		DescriptionStructured: []parser.StructuredDescription{
			{Number: "1", ID: "p-0001", Text: "This invention relates to a novel device."},
		},
	}
	dm := ToDataMap(data)
	AddStructuredFields(dm, data)

	if _, ok := dm.Get("claims_structured"); !ok {
		t.Error("claims_structured should be present after AddStructuredFields")
	}
	if _, ok := dm.Get("description_structured"); !ok {
		t.Error("description_structured should be present after AddStructuredFields")
	}
}

func TestAddStructuredFields_EmptyData_SetsEmptySlices(t *testing.T) {
	data := parser.PatentData{PublicationNumber: "US9999999B2"}
	dm := ToDataMap(data)
	AddStructuredFields(dm, data)

	v, ok := dm.Get("claims_structured")
	if !ok {
		t.Fatal("claims_structured key should be present even when data is empty")
	}
	claims, ok := v.([]parser.StructuredClaim)
	if !ok {
		t.Fatalf("claims_structured value type = %T, want []parser.StructuredClaim", v)
	}
	if len(claims) != 0 {
		t.Errorf("expected empty claims slice, got %d items", len(claims))
	}
}

func TestAddStructuredFields_NotInDefaultOutput(t *testing.T) {
	data := parser.PatentData{
		PublicationNumber: "US8725880B2",
		ClaimsStructured: []parser.StructuredClaim{
			{Number: "1", Type: "independent", Text: "A method comprising something."},
		},
	}
	dm := ToDataMap(data)
	// Without calling AddStructuredFields, claims_structured must not appear.
	if _, ok := dm.Get("claims_structured"); ok {
		t.Error("claims_structured should not be in default DataMap (opt-in only)")
	}
}

// ── StructuredFieldNames ──────────────────────────────────────────────────────

func TestStructuredFieldNames_Contents(t *testing.T) {
	want := map[string]bool{
		"claims_structured":     true,
		"description_structured": true,
	}
	if len(StructuredFieldNames) != len(want) {
		t.Errorf("StructuredFieldNames has %d entries, want %d", len(StructuredFieldNames), len(want))
	}
	for _, name := range StructuredFieldNames {
		if !want[name] {
			t.Errorf("unexpected entry in StructuredFieldNames: %q", name)
		}
	}
}

// ── TSV serialization of structured types ────────────────────────────────────

func TestRender_TSV_StructuredClaims(t *testing.T) {
	data := parser.PatentData{
		PublicationNumber: "US8725880B2",
		ClaimsStructured: []parser.StructuredClaim{
			{Number: "1", Type: "independent", Text: "Claim one text."},
			{Number: "2", Type: "dependent", DependsOn: []string{"1"}, Text: "Claim two text."},
		},
	}
	dm := ToDataMap(data)
	AddStructuredFields(dm, data)

	out := Render(dm, "tsv", false, false)
	if !strings.Contains(out, "claims_structured") {
		t.Error("TSV output should contain claims_structured header")
	}
	if !strings.Contains(out, "Claim one text") {
		t.Error("TSV output should contain claim text")
	}
}

func TestRender_TSV_StructuredDescription(t *testing.T) {
	data := parser.PatentData{
		PublicationNumber: "US8725880B2",
		DescriptionStructured: []parser.StructuredDescription{
			{Number: "1", ID: "p-0001", Text: "First paragraph."},
			{Number: "2", ID: "p-0002", Text: "Second paragraph."},
		},
	}
	dm := ToDataMap(data)
	AddStructuredFields(dm, data)

	out := Render(dm, "tsv", false, false)
	if !strings.Contains(out, "description_structured") {
		t.Error("TSV output should contain description_structured header")
	}
	if !strings.Contains(out, "First paragraph") {
		t.Error("TSV output should contain description text")
	}
}

func TestRender_JSON_StructuredClaims_HasTypeAndDependsOn(t *testing.T) {
	data := parser.PatentData{
		PublicationNumber: "US8725880B2",
		ClaimsStructured: []parser.StructuredClaim{
			{Number: "1", Type: "independent", Text: "A method comprising something."},
			{Number: "2", Type: "dependent", DependsOn: []string{"1"}, Text: "The method of claim 1."},
		},
	}
	dm := ToDataMap(data)
	AddStructuredFields(dm, data)

	out := Render(dm, "json", false, false)

	var env struct {
		Results struct {
			ClaimsStructured []struct {
				Number    string   `json:"number"`
				Type      string   `json:"type"`
				DependsOn []string `json:"depends_on"`
				Text      string   `json:"text"`
			} `json:"claims_structured"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}
	claims := env.Results.ClaimsStructured
	if len(claims) != 2 {
		t.Fatalf("got %d claims in JSON, want 2", len(claims))
	}
	if claims[0].Type != "independent" {
		t.Errorf("claims[0].type = %q, want independent", claims[0].Type)
	}
	if claims[1].Type != "dependent" {
		t.Errorf("claims[1].type = %q, want dependent", claims[1].Type)
	}
	if len(claims[1].DependsOn) != 1 || claims[1].DependsOn[0] != "1" {
		t.Errorf("claims[1].depends_on = %v, want [1]", claims[1].DependsOn)
	}
}

func TestPrintErrorJSON(t *testing.T) {
	// Capture stderr (PrintErrorJSON writes to stderr)
	orig := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	PrintErrorJSON("NOT_FOUND", "patent not found: US123")

	w.Close()
	os.Stderr = orig

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var env struct {
		OK    bool `json:"ok"`
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &env); err != nil {
		t.Fatalf("PrintErrorJSON output is not valid JSON: %v\noutput: %s", err, buf.String())
	}
	if env.OK {
		t.Error("ok should be false")
	}
	if env.Error.Type != "NOT_FOUND" {
		t.Errorf("error.type = %q, want NOT_FOUND", env.Error.Type)
	}
	if env.Error.Message != "patent not found: US123" {
		t.Errorf("error.message = %q, want %q", env.Error.Message, "patent not found: US123")
	}
}
