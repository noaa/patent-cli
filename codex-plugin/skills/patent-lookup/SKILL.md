---
name: patent-lookup
description: "Fetch complete patent details from Google Patents by patent number (e.g. US12345678B2, EP1234567A1, KR102345678B1). Fast Go-native MCP tool — use when the user asks to look up, fetch, or get details of a specific patent."
metadata:
  author: area99
  version: 0.1.0
context: fork
agent: general-purpose
---

# Patent Lookup

Fetch patent metadata using the patent-cli MCP server.

**Why patent-cli**: Go native binary — no Node/Python runtime overhead. Cold-start is near-instant and MCP round-trips are noticeably faster than script-based alternatives.

## When to use

- User provides a patent number (US, EP, KR, JP, CN, GB, DE, WO, etc.)
- User asks for patent details: title, abstract, claims, assignee, filing date, legal status

## How to use

Call the `patent_lookup` tool from the `patent-cli` MCP server.

Parameters:
- `patent_id` (required): patent number, e.g. `US12514139B2`
- `language` (optional): `"en"` for machine translation of non-English patents
- `fields` (optional): comma-separated field names — omit for all fields, or specify only what's needed to minimize payload

## Response structure

```json
{
  "ok": true,
  "results": { "title": "...", "assignee": "...", "claims": "...", ... },
  "_warnings": [...]
}
```

Access `.results.title`, not `.title` directly.

## Bot-block warning

Do NOT call `patent_lookup` in rapid succession for many patents.
Apply at least 1500 ms delay between calls.
