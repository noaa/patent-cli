package parser

import (
	"strings"
	"testing"
)

// ── parseClaimsStructured ─────────────────────────────────────────────────────

// US-format HTML: outer div.claims → outer div.claim (no num) → inner div[num]
// with <claim-ref idref="CLM-XXXXX"> for dependency detection.
const usClaimsHTML = `<html><body>
<div class="claims">
  <div class="claim">
    <div id="CLM-00001" num="00001" class="claim independent-claim">
      <claim-text>1. A method comprising doing something useful and novel.</claim-text>
    </div>
  </div>
  <div class="claim">
    <div id="CLM-00002" num="00002" class="claim dependent-claim">
      <claim-text>2. The method of <claim-ref idref="CLM-00001">claim 1</claim-ref>, further comprising an additional step.</claim-text>
    </div>
  </div>
  <div class="claim">
    <div id="CLM-00003" num="00003" class="claim dependent-claim">
      <claim-text>3. The method of <claim-ref idref="CLM-00001">claim 1</claim-ref> and <claim-ref idref="CLM-00002">claim 2</claim-ref>, further comprising yet another step.</claim-text>
    </div>
  </div>
</div>
</body></html>`

func TestParseClaimsStructured_US(t *testing.T) {
	data := ParseAll(usClaimsHTML)
	claims := data.ClaimsStructured

	if len(claims) != 3 {
		t.Fatalf("got %d claims, want 3", len(claims))
	}

	// Claim 1: independent
	if claims[0].Number != "1" {
		t.Errorf("claims[0].Number = %q, want %q", claims[0].Number, "1")
	}
	if claims[0].Type != "independent" {
		t.Errorf("claims[0].Type = %q, want %q", claims[0].Type, "independent")
	}
	if len(claims[0].DependsOn) != 0 {
		t.Errorf("claims[0].DependsOn = %v, want empty", claims[0].DependsOn)
	}

	// Claim 2: dependent on claim 1
	if claims[1].Number != "2" {
		t.Errorf("claims[1].Number = %q, want %q", claims[1].Number, "2")
	}
	if claims[1].Type != "dependent" {
		t.Errorf("claims[1].Type = %q, want %q", claims[1].Type, "dependent")
	}
	if len(claims[1].DependsOn) != 1 || claims[1].DependsOn[0] != "1" {
		t.Errorf("claims[1].DependsOn = %v, want [1]", claims[1].DependsOn)
	}

	// Claim 3: dependent on claims 1 and 2
	if claims[2].Type != "dependent" {
		t.Errorf("claims[2].Type = %q, want dependent", claims[2].Type)
	}
	if len(claims[2].DependsOn) != 2 {
		t.Errorf("claims[2].DependsOn = %v, want [1 2]", claims[2].DependsOn)
	}
}

func TestParseClaimsStructured_LeadingZerosStripped(t *testing.T) {
	data := ParseAll(usClaimsHTML)
	for _, c := range data.ClaimsStructured {
		if strings.HasPrefix(c.Number, "0") {
			t.Errorf("claim number %q has leading zero", c.Number)
		}
	}
}

func TestParseClaimsStructured_TextNonEmpty(t *testing.T) {
	data := ParseAll(usClaimsHTML)
	for _, c := range data.ClaimsStructured {
		if strings.TrimSpace(c.Text) == "" {
			t.Errorf("claim %s has empty text", c.Number)
		}
	}
}

// ── parseClaimsStructuredTranslated ──────────────────────────────────────────

// Translated-page HTML: <claim num="N"> custom elements with notranslate/google-src-text spans.
const translatedClaimsHTML = `<html><body>
<claim num="1"><span class="notranslate"><span class="google-src-text">독립청구항 원문</span>A device comprising a first component.</span></claim>
<claim num="2"><span class="notranslate"><span class="google-src-text">종속청구항 원문</span>The device of claim 1, further comprising a second component.</span></claim>
<claim num="3">   </claim>
</body></html>`

func TestParseClaimsStructured_Translated(t *testing.T) {
	data := ParseAll(translatedClaimsHTML)
	claims := data.ClaimsStructured

	// Claim 3 has only whitespace — should be omitted.
	if len(claims) != 2 {
		t.Fatalf("got %d claims, want 2 (empty claim omitted)", len(claims))
	}
	if claims[0].Number != "1" {
		t.Errorf("claims[0].Number = %q, want 1", claims[0].Number)
	}
	if claims[0].Type != "" {
		t.Errorf("claims[0].Type = %q, want empty (translated page has no type info)", claims[0].Type)
	}
	if !strings.Contains(claims[0].Text, "first component") {
		t.Errorf("claims[0].Text should contain English text; got %q", claims[0].Text)
	}
	if strings.Contains(claims[0].Text, "원문") {
		t.Errorf("claims[0].Text should not contain original-language text; got %q", claims[0].Text)
	}
}

// ── parseClaimsTranslated (plain []string) ────────────────────────────────────

func TestParseClaimsTranslated_PlainText(t *testing.T) {
	data := ParseAll(translatedClaimsHTML)
	claims := data.Claims

	if len(claims) != 2 {
		t.Fatalf("got %d plain claims, want 2", len(claims))
	}
	if strings.Contains(claims[0], "원문") {
		t.Errorf("claim text should not contain original-language text; got %q", claims[0])
	}
	if !strings.Contains(claims[0], "first component") {
		t.Errorf("claim text should contain English translation; got %q", claims[0])
	}
}

// ── parseDescriptionStructured ────────────────────────────────────────────────

const nativeDescriptionHTML = `<html><body>
<div itemprop="description">
  <div class="description-paragraph" num="0001" id="p-0001">This invention relates to a novel device.</div>
  <div class="description-paragraph" num="0002" id="p-0002">The device includes a first component and a second component.</div>
  <div class="description-paragraph" num="0003" id="p-0003">   </div>
</div>
</body></html>`

func TestParseDescriptionStructured_Native(t *testing.T) {
	data := ParseAll(nativeDescriptionHTML)
	descs := data.DescriptionStructured

	// Paragraph 0003 is whitespace-only — should be omitted.
	if len(descs) != 2 {
		t.Fatalf("got %d description paragraphs, want 2", len(descs))
	}
	if descs[0].Number != "1" {
		t.Errorf("descs[0].Number = %q, want 1", descs[0].Number)
	}
	if descs[0].ID != "p-0001" {
		t.Errorf("descs[0].ID = %q, want p-0001", descs[0].ID)
	}
	if !strings.Contains(descs[0].Text, "novel device") {
		t.Errorf("descs[0].Text = %q, expected to contain 'novel device'", descs[0].Text)
	}
	if descs[1].Number != "2" {
		t.Errorf("descs[1].Number = %q, want 2", descs[1].Number)
	}
}

// ── parseDescriptionStructuredTranslated ─────────────────────────────────────

const translatedDescriptionHTML = `<html><body>
<div itemprop="description">
  <p>First paragraph in English. <span class="notranslate">첫 번째 단락 원문.</span></p>
  <p>   </p>
  <p>Second paragraph in English.</p>
</div>
</body></html>`

func TestParseDescriptionStructured_Translated(t *testing.T) {
	data := ParseAll(translatedDescriptionHTML)
	descs := data.DescriptionStructured

	// Empty paragraph stripped; two non-empty paragraphs remain.
	if len(descs) != 2 {
		t.Fatalf("got %d description paragraphs, want 2", len(descs))
	}
	// Sequential numbering starting at "1".
	if descs[0].Number != "1" {
		t.Errorf("descs[0].Number = %q, want 1", descs[0].Number)
	}
	if descs[1].Number != "2" {
		t.Errorf("descs[1].Number = %q, want 2", descs[1].Number)
	}
	// Original-language text stripped.
	if strings.Contains(descs[0].Text, "원문") {
		t.Errorf("descs[0].Text should not contain original-language text; got %q", descs[0].Text)
	}
	if !strings.Contains(descs[0].Text, "First paragraph") {
		t.Errorf("descs[0].Text should contain English text; got %q", descs[0].Text)
	}
	// Translated paragraphs have no ID (sequential-only numbering).
	if descs[0].ID != "" {
		t.Errorf("descs[0].ID = %q, want empty for translated paragraphs", descs[0].ID)
	}
}

// ── translated abstract fallback ──────────────────────────────────────────────

const translatedAbstractHTML = `<html><head>
<meta name="description" content="A device for performing novel operations.">
</head><body>
<section itemprop="abstract">
  <div class="abstract">
    <span class="notranslate">원래 초록 내용</span>
  </div>
</section>
</body></html>`

func TestParseAbstract_Translated_FallsBackToMeta(t *testing.T) {
	data := ParseAll(translatedAbstractHTML)

	if data.Abstract == "" {
		t.Fatal("abstract should not be empty for translated page")
	}
	if strings.Contains(data.Abstract, "원래") {
		t.Errorf("abstract should not contain original-language text; got %q", data.Abstract)
	}
	if !strings.Contains(data.Abstract, "novel operations") {
		t.Errorf("abstract should contain meta description text; got %q", data.Abstract)
	}
}

// ── empty HTML edge cases ─────────────────────────────────────────────────────

func TestParseClaimsStructured_EmptyHTML(t *testing.T) {
	data := ParseAll("<html><body></body></html>")
	if data.ClaimsStructured == nil {
		// nil is fine — consistent with no claims found
		return
	}
	if len(data.ClaimsStructured) != 0 {
		t.Errorf("expected 0 structured claims for empty HTML, got %d", len(data.ClaimsStructured))
	}
}

func TestParseDescriptionStructured_EmptyHTML(t *testing.T) {
	data := ParseAll("<html><body></body></html>")
	if len(data.DescriptionStructured) != 0 {
		t.Errorf("expected 0 description paragraphs for empty HTML, got %d", len(data.DescriptionStructured))
	}
}
