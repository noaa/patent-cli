---
name: patent-fields
description: "List all available field names for patent_lookup. Use when the user asks what fields or data are available, or before filtering with specific fields."
metadata:
  author: area99
  version: 0.1.0
context: fork
agent: general-purpose
---

# Patent Fields

List all available field names in patent_lookup results.

## When to use

- User asks "what fields are available?"
- Before calling `patent_lookup` with a `fields` filter

## How to use

Call the `patent_fields` tool from the `patent-cli` MCP server (no parameters required).

Returns two categories:
- **Standard fields**: title, assignee, abstract, claims, filing_date, etc.
- **Opt-in structured fields**: `claims_structured`, `description_structured` (must be requested explicitly via the `fields` parameter of `patent_lookup`)
