//go:build integration

package integration

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// mcpResult is the generic JSON-RPC response envelope.
type mcpResult struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// runMCP sends a sequence of newline-delimited JSON-RPC messages to gp-cli mcp
// and collects response lines.
//
// It keeps stdin open (via a pipe) until all expected responses arrive or
// timeout expires, then closes stdin so the server shuts down gracefully.
// expectedIDs is the set of request IDs we wait for before closing stdin.
func runMCP(t *testing.T, messages []string, expectedIDs []int, timeout time.Duration) []mcpResult {
	t.Helper()

	cmd := exec.Command(binaryPath, "mcp")

	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	cmd.Stdin = stdinR

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	cmd.Stdout = stdoutW

	if err := cmd.Start(); err != nil {
		t.Fatalf("start mcp: %v", err)
	}
	_ = stdinR.Close()
	_ = stdoutW.Close()

	// Write messages to the server.
	for _, msg := range messages {
		fmt.Fprintln(stdinW, msg)
	}

	// Collect responses in background, close stdin once all expected IDs arrive.
	results := make(chan mcpResult, 64)
	go func() {
		defer close(results)
		scanner := bufio.NewScanner(stdoutR)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == "" {
				continue
			}
			var r mcpResult
			if err := json.Unmarshal([]byte(line), &r); err != nil {
				continue
			}
			results <- r
		}
	}()

	// Wait until all expected response IDs are received, then close stdin.
	deadline := time.After(timeout)
	seen := make(map[int]bool)
	var collected []mcpResult

	for len(seen) < len(expectedIDs) {
		select {
		case r, ok := <-results:
			if !ok {
				goto done
			}
			collected = append(collected, r)
			if r.ID != nil {
				seen[*r.ID] = true
			}
		case <-deadline:
			goto done
		}
	}

done:
	_ = stdinW.Close()
	// Drain remaining responses.
	for r := range results {
		collected = append(collected, r)
	}
	_ = cmd.Wait()
	return collected
}

// mcpExchange sends initialize + notifications/initialized + the given request
// and waits for both the initialize response (id=0) and the request response.
func mcpExchange(t *testing.T, requestID int, method string, params any, timeout time.Duration) []mcpResult {
	t.Helper()
	paramsJSON, _ := json.Marshal(params)

	messages := []string{
		`{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"integration-test","version":"0.0.1"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"%s","params":%s}`, requestID, method, paramsJSON),
	}
	return runMCP(t, messages, []int{0, requestID}, timeout)
}

// findByID returns the response with the given id, or nil.
func findByID(results []mcpResult, id int) *mcpResult {
	for i := range results {
		if results[i].ID != nil && *results[i].ID == id {
			return &results[i]
		}
	}
	return nil
}

// TestMCP_Initialize verifies the server responds with a valid initialize result.
func TestMCP_Initialize(t *testing.T) {
	results := runMCP(t, []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.0.1"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
	}, []int{1}, 5*time.Second)

	r := findByID(results, 1)
	if r == nil {
		t.Fatal("no response for initialize request")
	}
	if r.Error != nil {
		t.Fatalf("initialize error: %s", r.Error.Message)
	}

	var initResult struct {
		ProtocolVersion string `json:"protocolVersion"`
		ServerInfo      struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"serverInfo"`
	}
	if err := json.Unmarshal(r.Result, &initResult); err != nil {
		t.Fatalf("unmarshal initialize result: %v", err)
	}
	if initResult.ServerInfo.Name != "patent-cli" {
		t.Errorf("serverInfo.name = %q, want patent-cli", initResult.ServerInfo.Name)
	}
}

// TestMCP_ToolsList verifies the server advertises patent_lookup and patent_fields.
func TestMCP_ToolsList(t *testing.T) {
	results := mcpExchange(t, 2, "tools/list", map[string]any{}, 5*time.Second)

	r := findByID(results, 2)
	if r == nil {
		t.Fatal("no response for tools/list")
	}
	if r.Error != nil {
		t.Fatalf("tools/list error: %s", r.Error.Message)
	}

	var listResult struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(r.Result, &listResult); err != nil {
		t.Fatalf("unmarshal tools/list result: %v", err)
	}

	names := make(map[string]bool)
	for _, tool := range listResult.Tools {
		names[tool.Name] = true
	}
	for _, want := range []string{"patent_lookup", "patent_fields"} {
		if !names[want] {
			t.Errorf("tool %q not listed; got %v", want, listResult.Tools)
		}
	}
}

// TestMCP_PatentFields verifies the patent_fields tool returns known field names.
func TestMCP_PatentFields(t *testing.T) {
	results := mcpExchange(t, 2, "tools/call",
		map[string]any{"name": "patent_fields", "arguments": map[string]any{}},
		5*time.Second)

	r := findByID(results, 2)
	if r == nil {
		t.Fatal("no response for patent_fields call")
	}
	if r.Error != nil {
		t.Fatalf("patent_fields error: %s", r.Error.Message)
	}

	var callResult struct {
		IsError bool `json:"isError"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(r.Result, &callResult); err != nil {
		t.Fatalf("unmarshal patent_fields result: %v", err)
	}
	if callResult.IsError {
		t.Fatal("patent_fields returned isError=true")
	}
	if len(callResult.Content) == 0 {
		t.Fatal("patent_fields returned empty content")
	}
	text := callResult.Content[0].Text
	for _, field := range []string{"publication_number", "title", "assignee", "claims"} {
		if !strings.Contains(text, field) {
			t.Errorf("field %q not found in patent_fields output", field)
		}
	}
}

// TestMCP_PatentLookup fetches a known patent and checks key fields.
func TestMCP_PatentLookup(t *testing.T) {
	results := mcpExchange(t, 2, "tools/call",
		map[string]any{
			"name": "patent_lookup",
			"arguments": map[string]any{
				"patent_id": "US8725880B2",
				"fields":    "title,assignee,filing_date",
			},
		},
		30*time.Second)

	r := findByID(results, 2)
	if r == nil {
		t.Fatal("no response for patent_lookup call")
	}
	if r.Error != nil {
		t.Fatalf("patent_lookup protocol error: %s", r.Error.Message)
	}

	var callResult struct {
		IsError bool `json:"isError"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(r.Result, &callResult); err != nil {
		t.Fatalf("unmarshal patent_lookup result: %v", err)
	}
	if callResult.IsError {
		t.Fatalf("patent_lookup returned isError=true: %s", callResult.Content[0].Text)
	}

	var envelope struct {
		OK      bool `json:"ok"`
		Results struct {
			Title      string `json:"title"`
			Assignee   string `json:"assignee"`
			FilingDate string `json:"filing_date"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(callResult.Content[0].Text), &envelope); err != nil {
		t.Fatalf("unmarshal patent data: %v\nraw: %s", err, callResult.Content[0].Text)
	}
	if !envelope.OK {
		t.Fatal("patent result ok=false")
	}
	if !strings.Contains(envelope.Results.Assignee, "Apple") {
		t.Errorf("assignee = %q, want to contain 'Apple'", envelope.Results.Assignee)
	}
	if envelope.Results.FilingDate == "" {
		t.Error("filing_date is empty")
	}
}

// TestMCP_PatentLookup_NotFound verifies a non-existent patent returns isError=true.
func TestMCP_PatentLookup_NotFound(t *testing.T) {
	results := mcpExchange(t, 2, "tools/call",
		map[string]any{
			"name":      "patent_lookup",
			"arguments": map[string]any{"patent_id": "US0000000Z9"},
		},
		15*time.Second)

	r := findByID(results, 2)
	if r == nil {
		t.Fatal("no response for patent_lookup (not-found case)")
	}

	var callResult struct {
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(r.Result, &callResult); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if !callResult.IsError {
		t.Error("expected isError=true for non-existent patent")
	}
}
