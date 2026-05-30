package mcpserver_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/area99/patent-cli/internal/formatter"
	"github.com/area99/patent-cli/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// newTestSession starts the MCP server with an in-memory transport and returns
// a connected client session.
func newTestSession(t *testing.T) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	srv := mcpserver.Build("test")
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)

	t1, t2 := mcp.NewInMemoryTransports()
	if _, err := srv.Connect(ctx, t1, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}
	sess, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = sess.Close() })
	return sess
}

// TestToolsList verifies the server advertises exactly the two expected tools.
func TestToolsList(t *testing.T) {
	sess := newTestSession(t)
	res, err := sess.ListTools(context.Background(), &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("tools/list: %v", err)
	}
	names := make(map[string]bool)
	for _, tool := range res.Tools {
		names[tool.Name] = true
	}
	for _, want := range []string{"patent_lookup", "patent_fields"} {
		if !names[want] {
			t.Errorf("missing tool %q; got %v", want, res.Tools)
		}
	}
	if len(res.Tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(res.Tools))
	}
}

// TestPatentFields verifies the tool lists at least the standard field names.
func TestPatentFields(t *testing.T) {
	sess := newTestSession(t)
	res, err := sess.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "patent_fields",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("patent_fields call: %v", err)
	}
	if res.IsError {
		t.Fatalf("patent_fields returned IsError=true")
	}
	if len(res.Content) == 0 {
		t.Fatal("patent_fields returned empty content")
	}
	text := res.Content[0].(*mcp.TextContent).Text
	for _, field := range formatter.FieldOrder {
		if !strings.Contains(text, field) {
			t.Errorf("field %q not found in patent_fields output", field)
		}
	}
	for _, field := range formatter.StructuredFieldNames {
		if !strings.Contains(text, field) {
			t.Errorf("structured field %q not found in patent_fields output", field)
		}
	}
}

// TestPatentLookup_MissingID verifies that an empty patent_id returns an error.
func TestPatentLookup_MissingID(t *testing.T) {
	sess := newTestSession(t)
	res, err := sess.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "patent_lookup",
		Arguments: map[string]any{"patent_id": ""},
	})
	// The SDK packs handler errors into CallToolResult.IsError, not as protocol errors.
	if err != nil {
		t.Fatalf("unexpected protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true for empty patent_id, got false")
	}
}

// TestPatentLookup_InputSchema verifies patent_id is marked required in the schema.
func TestPatentLookup_InputSchema(t *testing.T) {
	sess := newTestSession(t)
	res, err := sess.ListTools(context.Background(), &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("tools/list: %v", err)
	}
	var lookupTool *mcp.Tool
	for _, tool := range res.Tools {
		if tool.Name == "patent_lookup" {
			lookupTool = tool
			break
		}
	}
	if lookupTool == nil {
		t.Fatal("patent_lookup tool not found")
	}

	schema, err := json.Marshal(lookupTool.InputSchema)
	if err != nil {
		t.Fatalf("marshal input schema: %v", err)
	}
	var s map[string]any
	if err := json.Unmarshal(schema, &s); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}

	required, _ := s["required"].([]any)
	found := false
	for _, r := range required {
		if r == "patent_id" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("patent_id not in required list; schema: %s", schema)
	}

	props, _ := s["properties"].(map[string]any)
	for _, field := range []string{"patent_id", "language", "fields"} {
		if _, ok := props[field]; !ok {
			t.Errorf("property %q missing from input schema", field)
		}
	}
}
