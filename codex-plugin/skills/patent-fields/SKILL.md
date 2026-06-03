---
name: patent-fields
description: "List field names available from patent_lookup and gp-cli lookup. Use before filtering with fields or --fields, and to explain which patent data the CLI can return."
metadata:
  author: area99
  version: 0.1.0
---

# Patent Fields

List all available field names in `patent_lookup` and `gp-cli lookup` results.

## When to use

- User asks "what fields are available?"
- Before calling `patent_lookup` with a `fields` filter
- Before running `gp-cli lookup --fields ...` or `gp-cli lookup --field ...`
- When the user asks whether a specific data point is returned as a top-level field

## How to use

Call the `patent_fields` tool from the `patent-cli` MCP server (no parameters required).

Equivalent CLI command:

```sh
gp-cli fields
```

Returns two categories:
- **Standard fields**: title, assignee, abstract, claims, filing_date, etc.
- **Opt-in structured fields**: `claims_structured`, `description_structured` (must be requested explicitly via the `fields` parameter of `patent_lookup`)

## Field caveats

- There is no top-level `legal_status`, `priority_date`, or `pdf_url` field.
- Legal status is available only within `family_applications` entries when present.
- Use `patent_url` for the Google Patents page URL.
- Use `gp-cli download` to save PDFs and `gp-cli images` to save drawing figures.
