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
	out := Render(dm, "json", false)

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
	out := Render(dm, "json", false)

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
	compact := Render(dm, "json", true)
	indented := Render(dm, "json", false)

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
	out := Render(dm, "text", false)

	if strings.HasPrefix(out, "{") {
		t.Error("text output should not start with '{' (no JSON envelope)")
	}
	if !strings.Contains(out, "Test Patent Title") {
		t.Error("text output should contain the title")
	}
}

func TestRender_TSV_HasHeader(t *testing.T) {
	dm := sampleDataMap()
	out := Render(dm, "tsv", false)

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

func TestPrintErrorJSON(t *testing.T) {
	// Capture stdout
	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintErrorJSON("NOT_FOUND", "patent not found: US123")

	w.Close()
	os.Stdout = orig

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
