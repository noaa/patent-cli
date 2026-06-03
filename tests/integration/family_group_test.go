//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// familyGroupResult is the JSON envelope for family-group output.
type familyGroupResult struct {
	OK     bool `json:"ok"`
	Groups []struct {
		ID      int      `json:"id"`
		Patents []string `json:"patents"`
	} `json:"groups"`
	Errors []struct {
		Patent  string `json:"patent"`
		Message string `json:"message"`
	} `json:"errors"`
	Summary struct {
		TotalInput  int `json:"total_input"`
		TotalGroups int `json:"total_groups"`
		TotalErrors int `json:"total_errors"`
		FetchCount  int `json:"fetch_count"`
	} `json:"summary"`
}

// TestFamilyGroup_Grouping verifies that known family members are grouped together
// and a non-family patent is placed in its own group.
// Uses US8725880B2 family (US8725880B2, US8704863B2, US8423058B2) + US9735861B2.
func TestFamilyGroup_Grouping(t *testing.T) {
	out, code := runCLI(t, "family-group",
		"US8725880B2", "US8704863B2", "US8423058B2", "US9735861B2",
		"--quiet",
	)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout: %s", code, out)
	}

	var result familyGroupResult
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("invalid JSON: %v\nstdout: %s", err, out)
	}
	if !result.OK {
		t.Error("ok = false, want true")
	}
	if result.Summary.TotalInput != 4 {
		t.Errorf("summary.total_input = %d, want 4", result.Summary.TotalInput)
	}
	if result.Summary.TotalGroups != 2 {
		t.Errorf("summary.total_groups = %d, want 2", result.Summary.TotalGroups)
	}
	if result.Summary.TotalErrors != 0 {
		t.Errorf("summary.total_errors = %d, want 0", result.Summary.TotalErrors)
	}

	// Find the group containing US8725880B2.
	var familyGroup *struct {
		ID      int
		Patents []string
	}
	for _, g := range result.Groups {
		for _, p := range g.Patents {
			if p == "US8725880B2" {
				familyGroup = &struct {
					ID      int
					Patents []string
				}{g.ID, g.Patents}
				break
			}
		}
	}
	if familyGroup == nil {
		t.Fatal("US8725880B2 not found in any group")
	}
	if len(familyGroup.Patents) != 3 {
		t.Errorf("family group has %d members, want 3: %v", len(familyGroup.Patents), familyGroup.Patents)
	}
	// All three family members must be in the same group.
	inGroup := make(map[string]bool)
	for _, p := range familyGroup.Patents {
		inGroup[p] = true
	}
	for _, want := range []string{"US8725880B2", "US8704863B2", "US8423058B2"} {
		if !inGroup[want] {
			t.Errorf("%s missing from family group", want)
		}
	}
}

// TestFamilyGroup_FetchCount verifies the skip optimization:
// fetching the first family member should identify all others without re-fetching them.
func TestFamilyGroup_FetchCount(t *testing.T) {
	out, code := runCLI(t, "family-group",
		"US8725880B2", "US8704863B2", "US8423058B2", "US9735861B2",
		"--quiet",
	)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout: %s", code, out)
	}

	var result familyGroupResult
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("invalid JSON: %v\nstdout: %s", err, out)
	}
	// 4 inputs, 3 are family → only 2 fetches needed (first family member + solo)
	if result.Summary.FetchCount != 2 {
		t.Errorf("fetch_count = %d, want 2 (family skip optimization)", result.Summary.FetchCount)
	}
}

// TestFamilyGroup_SinglePatent verifies that a solo patent gets its own group.
func TestFamilyGroup_SinglePatent(t *testing.T) {
	out, code := runCLI(t, "family-group", "US9735861B2", "--quiet")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout: %s", code, out)
	}

	var result familyGroupResult
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("invalid JSON: %v\nstdout: %s", err, out)
	}
	if result.Summary.TotalGroups != 1 {
		t.Errorf("total_groups = %d, want 1", result.Summary.TotalGroups)
	}
	if len(result.Groups[0].Patents) != 1 {
		t.Errorf("solo group has %d members, want 1", len(result.Groups[0].Patents))
	}
}

// TestFamilyGroup_TextFormat verifies text output contains group headers and patent numbers.
func TestFamilyGroup_TextFormat(t *testing.T) {
	out, code := runCLI(t, "family-group",
		"US8725880B2", "US9735861B2",
		"--format", "text", "--quiet",
	)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout: %s", code, out)
	}

	s := string(out)
	if !strings.Contains(s, "Group 1") {
		t.Error("text output missing 'Group 1'")
	}
	if !strings.Contains(s, "US8725880B2") {
		t.Error("text output missing US8725880B2")
	}
	if !strings.Contains(s, "Summary:") {
		t.Error("text output missing 'Summary:' line")
	}
}

// TestFamilyGroup_TSVFormat verifies TSV output has a header and correct columns.
func TestFamilyGroup_TSVFormat(t *testing.T) {
	out, code := runCLI(t, "family-group",
		"US8725880B2", "US9735861B2",
		"--format", "tsv", "--quiet",
	)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout: %s", code, out)
	}

	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("TSV has only %d line(s), want at least 2 (header + data)", len(lines))
	}
	if lines[0] != "group_id\tpatent_number" {
		t.Errorf("TSV header = %q, want %q", lines[0], "group_id\tpatent_number")
	}
	for i, line := range lines[1:] {
		cols := strings.Split(line, "\t")
		if len(cols) != 2 {
			t.Errorf("line %d: got %d columns, want 2: %q", i+1, len(cols), line)
		}
	}
}

// TestFamilyGroup_TSVNoHeader verifies --no-header omits the TSV header row.
func TestFamilyGroup_TSVNoHeader(t *testing.T) {
	out, code := runCLI(t, "family-group",
		"US8725880B2",
		"--format", "tsv", "--no-header", "--quiet",
	)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout: %s", code, out)
	}

	first := strings.SplitN(string(out), "\n", 2)[0]
	if first == "group_id\tpatent_number" {
		t.Error("--no-header should suppress the TSV header row")
	}
}

// TestFamilyGroup_InputFile verifies --input-file reads patent numbers from a file.
func TestFamilyGroup_InputFile(t *testing.T) {
	tmp := t.TempDir()
	inputPath := filepath.Join(tmp, "patents.txt")
	content := "# family members\nUS8725880B2\nUS8704863B2\n\n# standalone\nUS9735861B2\n"
	if err := os.WriteFile(inputPath, []byte(content), 0644); err != nil {
		t.Fatalf("write input file: %v", err)
	}

	out, code := runCLI(t, "family-group", "--input-file", inputPath, "--quiet")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout: %s", code, out)
	}

	var result familyGroupResult
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("invalid JSON: %v\nstdout: %s", err, out)
	}
	if result.Summary.TotalInput != 3 {
		t.Errorf("total_input = %d, want 3", result.Summary.TotalInput)
	}
	if result.Summary.TotalGroups != 2 {
		t.Errorf("total_groups = %d, want 2", result.Summary.TotalGroups)
	}
}

// TestFamilyGroup_OutputDir verifies --output-dir saves a file named family_groups.json.
func TestFamilyGroup_OutputDir(t *testing.T) {
	tmp := t.TempDir()

	_, code := runCLI(t, "family-group",
		"US8725880B2", "US9735861B2",
		"--output-dir", tmp, "--quiet",
	)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	dest := filepath.Join(tmp, "family_groups.json")
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("output file not found: %v", err)
	}
	if !json.Valid(data) {
		t.Errorf("saved file is not valid JSON: %s", data)
	}
}

// TestFamilyGroup_Minify checks that --minify produces single-line JSON.
func TestFamilyGroup_Minify(t *testing.T) {
	out, code := runCLI(t, "family-group", "US8725880B2", "--minify", "--quiet")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout: %s", code, out)
	}

	newlines := 0
	for _, b := range out {
		if b == '\n' {
			newlines++
		}
	}
	if newlines > 1 {
		t.Errorf("--minify output has %d newlines, want ≤1", newlines)
	}
	if !json.Valid(out) {
		t.Error("--minify output is not valid JSON")
	}
}
