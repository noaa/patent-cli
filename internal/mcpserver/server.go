package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/area99/patent-cli/internal/fetcher"
	"github.com/area99/patent-cli/internal/formatter"
	"github.com/area99/patent-cli/internal/parser"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// LookupParams is the input schema for the patent_lookup tool.
type LookupParams struct {
	PatentID string `json:"patent_id" jsonschema:"Patent number to look up (e.g. US12514139B2 or EP1234567A1)"`
	Language string `json:"language,omitempty" jsonschema:"Language code for machine translation (e.g. 'en'). Useful for non-English patents."`
	Fields   string `json:"fields,omitempty" jsonschema:"Comma-separated list of fields to return. Omit for all fields. Use the patent_fields tool to see available field names."`
}

// Build creates and returns the configured MCP server.
func Build(serverVersion string) *mcp.Server {
	srv := mcp.NewServer(&mcp.Implementation{
		Name:    "patent-cli",
		Version: serverVersion,
	}, nil)

	mcp.AddTool(srv,
		&mcp.Tool{
			Name:        "patent_lookup",
			Description: "Fetch patent metadata from Google Patents by patent number. Returns title, abstract, claims, assignee, filing date, legal status, and more.",
		},
		handleLookup,
	)

	mcp.AddTool(srv,
		&mcp.Tool{
			Name:        "patent_fields",
			Description: "List all field names available in patent_lookup results, with human-readable labels.",
		},
		handleFields,
	)

	return srv
}

// handleLookup fetches patent data and returns it as JSON.
func handleLookup(_ context.Context, _ *mcp.CallToolRequest, input LookupParams) (*mcp.CallToolResult, any, error) {
	if strings.TrimSpace(input.PatentID) == "" {
		return nil, nil, fmt.Errorf("patent_id is required")
	}

	opts := fetcher.Options{Timeout: 15 * time.Second}
	opts.Language = input.Language

	html, err := fetcher.FetchHTML(input.PatentID, opts)
	if err != nil {
		switch err.(type) {
		case *fetcher.PatentNotFoundError:
			return nil, nil, fmt.Errorf("patent not found: %s", input.PatentID)
		case *fetcher.BotBlockedError:
			return nil, nil, fmt.Errorf("bot-block detected: %v", err)
		default:
			return nil, nil, fmt.Errorf("network error: %v", err)
		}
	}

	data := parser.ParseAll(html)
	dm := formatter.ToDataMap(data)

	// Resolve requested fields.
	var fieldsList []string
	if input.Fields != "" {
		for _, f := range strings.Split(input.Fields, ",") {
			f = strings.TrimSpace(f)
			if f != "" {
				fieldsList = append(fieldsList, f)
			}
		}
		// Opt-in structured fields.
		for _, f := range fieldsList {
			for _, sf := range formatter.StructuredFieldNames {
				if f == sf {
					formatter.AddStructuredFields(dm, data)
					break
				}
			}
		}
	}

	out := formatter.SelectFields(dm, fieldsList)

	// Render as compact JSON for the MCP text response.
	rendered := formatter.Render(out, "json", true, false)

	// Verify it's valid JSON before returning.
	var check any
	if err := json.Unmarshal([]byte(rendered), &check); err != nil {
		return nil, nil, fmt.Errorf("internal render error: %v", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: rendered}},
	}, nil, nil
}

// handleFields lists available field names.
func handleFields(_ context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	var sb strings.Builder
	sb.WriteString("Standard fields:\n")
	for _, key := range formatter.FieldOrder {
		label := formatter.LabelFor(key)
		sb.WriteString(fmt.Sprintf("  %-25s %s\n", key, label))
	}
	sb.WriteString("\nOpt-in structured fields (pass in 'fields' parameter of patent_lookup):\n")
	for _, key := range formatter.StructuredFieldNames {
		label := formatter.LabelFor(key)
		sb.WriteString(fmt.Sprintf("  %-25s %s\n", key, label))
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: sb.String()}},
	}, nil, nil
}
