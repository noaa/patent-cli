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

If MCP tools are unavailable but the `gp-cli` binary is available, use:

```sh
gp-cli fields
```

## Common field groups

Frequently used metadata fields:
- `publication_number`, `application_number`, `kind_code`, `country`
- `title`, `abstract`, `inventors`, `assignee`
- `filing_date`, `publication_date`
- `cpc_codes`, `claims`, `description`, `patent_url`
- `backward_citations`, `forward_citations`, `similar_documents`
- `family_applications`

Opt-in fields:
- `claims_structured`: per-claim number, type, dependency, and text when Google Patents source markup supports it.
- `description_structured`: structured description paragraphs.

There is no top-level `legal_status`, `priority_date`, or `pdf_url` field.
Legal status appears inside `family_applications` entries when Google Patents
provides it. Use `gp-cli download` for PDF files and `gp-cli images` for
drawing figures.
Use `gp-cli family-group` to group a patent-number list by patent family;
family grouping is a CLI workflow, not a `patent_lookup` output field.
