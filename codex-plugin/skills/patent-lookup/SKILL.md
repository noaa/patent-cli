---
name: patent-lookup
description: "Fetch Google Patents metadata by patent number with patent_lookup, and use gp-cli commands for PDFs, figure images, bulk runs, language translation, configuration, and updates."
metadata:
  author: area99
  version: 0.1.0
---

# Patent Lookup

Fetch patent metadata using the patent-cli MCP server. For operations that the
MCP server does not expose, run the `gp-cli` command directly.

**Why patent-cli**: Pure HTTP fetch, no headless browser. Single MCP call returns results directly — no dataset name + separate query step required by browser-based alternatives.

## When to use

- User provides a patent number (US, EP, KR, JP, CN, GB, DE, WO, etc.)
- User asks for patent metadata or text: title, abstract, claims, assignee, filing date, publication date, citations, family applications, CPC codes, description, or patent URL
- User asks to download a patent PDF or drawing figures
- User asks to process a file of patent numbers in bulk
- User asks to configure proxy, CA bundle, or request delay settings

## MCP tools

Call the `patent_lookup` tool from the `patent-cli` MCP server.

Parameters:
- `patent_id` (required): patent number, e.g. `US12514139B2`
- `language` (optional): `"en"` for machine translation of non-English patents
- `fields` (optional): comma-separated field names — omit for all fields, or specify only what's needed to minimize payload

Only these MCP tools are exposed:
- `patent_lookup`: fetch metadata and text fields
- `patent_fields`: list available field names

Do not assume MCP can download PDFs or figure images. Use the CLI commands below
for file downloads and bulk file workflows.

## Response structure

```json
{
  "ok": true,
  "results": { "title": "...", "assignee": "...", "claims": [...], ... },
  "_warnings": [...]
}
```

Access `.results.title`, not `.title` directly.

## CLI commands

Use `gp-cli lookup` when the user needs shell output, saved files, TSV, a single
plain-text field, or bulk processing:

```sh
gp-cli lookup US12514139B2
gp-cli lookup US12514139B2 --format text
gp-cli lookup US12514139B2 --field title
gp-cli lookup US12514139B2 --fields title,assignee,filing_date
gp-cli lookup KR102355140B1 --language en
gp-cli lookup --input-file patents.txt --format tsv --output-dir ./results
```

Use `gp-cli download` for patent PDFs:

```sh
gp-cli download US12514139B2 --output-dir ./pdfs
gp-cli download --input-file patents.txt --output-dir ./pdfs
```

Use `gp-cli images` for high-resolution drawing figures:

```sh
gp-cli images US12514139B2 --output-dir ./figures
gp-cli images --input-file patents.txt --output-dir ./figures
```

Use support commands when needed:

```sh
gp-cli fields
gp-cli configure
gp-cli update --check
gp-cli update
```

## Output fields

Common metadata fields include `publication_number`, `application_number`,
`kind_code`, `country`, `title`, `abstract`, `inventors`, `assignee`,
`filing_date`, `publication_date`, `cpc_codes`, `claims`, `description`,
`patent_url`, citation fields, `similar_documents`, and
`family_applications`.

`claims_structured` and `description_structured` are opt-in fields. Request
them explicitly through `fields` in MCP or `--fields` / `--field` in the CLI.

There is no top-level `legal_status`, `priority_date`, or `pdf_url` output
field. Legal status appears inside `family_applications` entries when Google
Patents provides it. PDF and figure files are downloaded with `gp-cli download`
and `gp-cli images`.

## Bot-block warning

Do NOT call `patent_lookup` in rapid succession for many patents.
Apply at least 1500 ms delay between manual calls.

For CLI bulk mode with `--input-file`, `gp-cli` automatically applies a random
1000-1500 ms delay between requests. For manual shell loops, pass `--delay
1500` or higher.
