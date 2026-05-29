//go:build integration

// Integration tests — hit live Google Patents.
// Run with: go test -tags integration ./tests/integration/ -v -timeout 300s
package integration

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	repoRoot := filepath.Join(wd, "..", "..")

	tmp, err := os.MkdirTemp("", "gp-cli-inttest-*")
	if err != nil {
		panic(err)
	}
	binaryPath = filepath.Join(tmp, "gp-cli")

	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/gp-cli/")
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		panic("build failed: " + string(out))
	}

	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

// runCLI executes the binary and returns (stdout, exit code).
func runCLI(t *testing.T, args ...string) ([]byte, int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return out, exitErr.ExitCode()
		}
		t.Fatalf("exec error: %v", err)
	}
	return out, 0
}

// patentResult is the top-level JSON envelope for a successful lookup.
type patentResult struct {
	OK      bool `json:"ok"`
	Results struct {
		PublicationNumber string `json:"publication_number"`
		Title             string `json:"title"`
		Country           string `json:"country"`
		Assignee          string `json:"assignee"`
	} `json:"results"`
}

// patentError is the top-level JSON envelope for an error response.
type patentError struct {
	OK    bool `json:"ok"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// TestLookup_ByCountry tests one patent per country from the US8725880B2 family.
func TestLookup_ByCountry(t *testing.T) {
	cases := []struct {
		country string
		patent  string
	}{
		{"US", "US8725880B2"},
		{"EP", "EP2556640B1"},
		{"KR", "KR101436225B1"},
		{"JP", "JP5596849B2"},
		{"CN", "CN102859962B"},
		{"GB", "GB2495814B"},
		{"DE", "DE112010005457B4"},
		{"AU", "AU2010350744B2"},
		{"BR", "BR112012025382B1"},
		{"MX", "MX2012011620A"},
		{"TW", "TWI551112B"},
		{"WO", "WO2011126505A1"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.country+"/"+tc.patent, func(t *testing.T) {
			out, code := runCLI(t, "lookup", tc.patent, "--format", "json", "--quiet")

			if code != 0 {
				t.Fatalf("exit code = %d, want 0\nstdout: %s", code, out)
			}

			var result patentResult
			if err := json.Unmarshal(out, &result); err != nil {
				t.Fatalf("stdout is not valid JSON: %v\nstdout: %s", err, out)
			}
			if !result.OK {
				t.Errorf("ok = false, want true")
			}
			if result.Results.Title == "" {
				t.Errorf("results.title is empty")
			}
		})
	}
}

// TestLookup_NotFound checks structured error and exit code 4 for unknown patents.
func TestLookup_NotFound(t *testing.T) {
	out, code := runCLI(t, "lookup", "US0000000X9")

	if code != 4 {
		t.Errorf("exit code = %d, want 4 (NOT_FOUND)", code)
	}

	var result patentError
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nstdout: %s", err, out)
	}
	if result.OK {
		t.Error("ok should be false for not-found")
	}
	if result.Error.Type != "NOT_FOUND" {
		t.Errorf("error.type = %q, want NOT_FOUND", result.Error.Type)
	}
}

// TestLookup_Minify checks that --minify produces single-line JSON.
func TestLookup_Minify(t *testing.T) {
	out, code := runCLI(t, "lookup", "US8725880B2", "--format", "json", "--minify", "--quiet")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout: %s", code, out)
	}

	lines := 0
	for _, b := range out {
		if b == '\n' {
			lines++
		}
	}
	if lines > 1 {
		t.Errorf("--minify output has %d newlines, want ≤1", lines)
	}
	if !json.Valid(out) {
		t.Error("--minify output is not valid JSON")
	}
}

// TestLookup_FieldFilter checks that --fields limits output keys.
func TestLookup_FieldFilter(t *testing.T) {
	out, code := runCLI(t, "lookup", "US8725880B2", "--fields", "title,assignee", "--quiet")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	var result struct {
		OK      bool                   `json:"ok"`
		Results map[string]interface{} `json:"results"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := result.Results["title"]; !ok {
		t.Error("results should contain 'title'")
	}
	if _, ok := result.Results["assignee"]; !ok {
		t.Error("results should contain 'assignee'")
	}
	if _, ok := result.Results["filing_date"]; ok {
		t.Error("results should not contain 'filing_date' when not requested")
	}
}
