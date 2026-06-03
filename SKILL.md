---
name: google-patent-cli
description: gp-cli — Rules and constraints for using the Google Patents CLI as an AI agent tool
---

# gp-cli — AI Agent Usage Guide

`gp-cli` is a CLI tool that retrieves structured patent metadata from Google Patents.
This document describes the rules for AI coding agents to use this tool correctly.

---

## Core Usage Patterns

```sh
# Single lookup
gp-cli lookup US12514139B2
gp-cli lookup US12514139B2 --format text
gp-cli lookup US12514139B2 --fields title,assignee,claims
gp-cli lookup KR102345678B1 --language en --fields title,claims_structured

# Batch lookup (--input-file)
gp-cli lookup --input-file patents.txt --format tsv --output-dir ./results
gp-cli download --input-file patents.txt --output-dir ./pdfs
gp-cli images --input-file patents.txt --output-dir ./figures

# Group a patent list by family
gp-cli family-group --input-file patents.txt --format tsv --output-dir ./groups
gp-cli family-group US8725880B2 US8704863B2 US9735861B2 --format text

# Download drawing images
gp-cli images US11125686B2 --output-dir ./figures

# Download PDF
gp-cli download US9735861B2 --output-dir ./pdfs
```

---

## JSON Response Structure — Required Parsing Path

`gp-cli lookup` JSON output always uses an **envelope** structure.
Access `.results.claims_structured`, not `.claims_structured` directly.

```json
{
  "ok": true,
  "results": {
    "title": "...",
    "assignee": "...",
    "claims_structured": [ ... ],
    ...
  },
  "_warnings": [ ... ]   // data quality warnings; key omitted entirely when absent
}
```

```sh
# jq examples
gp-cli lookup US12514139B2 | jq '.results.title'
gp-cli lookup US12514139B2 --fields claims_structured | jq '.results.claims_structured[] | select(.type == "independent")'
```

`gp-cli family-group` is a CLI-only workflow and uses a different JSON envelope:

```json
{
  "ok": true,
  "groups": [
    { "id": 1, "patents": ["US8725880B2", "US8704863B2"] }
  ],
  "summary": {
    "total_input": 3,
    "total_groups": 2,
    "total_errors": 0,
    "fetch_count": 2
  }
}
```

Use `.summary.fetch_count` to verify that already discovered family members
were skipped.

---

## Exit Code Handling

| Code | Meaning | Agent behavior |
|------|---------|----------------|
| 0 | Success | Continue normally |
| 1 | General error | Diagnose cause, retry or abort |
| 4 | NOT_FOUND | **Do not abort the script** — treat as a natural "no data" and move to the next item |
| 6 | Server error / bot-block | Increase delay and retry; abort after repeated failures |

Error JSON is always written to **stderr**. Stdout is always clean — safe to pipe.

```sh
# Pattern for handling exit 4 as a skip in manual loops
while IFS= read -r pn; do
  gp-cli lookup "$pn" --format tsv --no-header >> results.tsv 2>>errors.log || true
done < patents.txt
```

---

## Bot-Block Prevention — Mandatory Rules

**Always apply a delay when querying multiple patents in a loop or with `--input-file`.**

- With `--input-file`: a **1000–1500 ms random delay** is applied automatically — no extra configuration needed.
- With a manual loop: specify `--delay 1500` or higher explicitly.
- Querying 100+ patents in rapid succession without a delay will trigger Google's IP block, failing the entire job.

```sh
# Safe manual loop
while IFS= read -r pn; do
  gp-cli lookup "$pn" --delay 1500 --output-dir ./out
done < patents.txt

# Recommended: --input-file (delay applied automatically)
gp-cli lookup --input-file patents.txt --output-dir ./out
```

The `family-group` command also applies delay between fetches and skips later
input patents already identified as family members:

```sh
gp-cli family-group --input-file patents.txt --format tsv --delay 1500
```

When a bot-block occurs (exit 6, `SERVER_ERROR`):
1. Stop immediately
2. Wait at least 5 minutes
3. Retry with `--delay 3000` or higher

---

## `claims_structured` Constraints

### Support Matrix
| Condition | `type` / `depends_on` included |
|-----------|-------------------------------|
| US / EP native page | ✅ Included (`independent` / `dependent`, parent claim numbers) |
| `--language en` translated page | ❌ Text only — `type` and `depends_on` absent |
| KR / JP / CN etc. native (no translation) | ❌ Text only |

### Recommended Usage for Non-English Patents
```sh
# When using claims_structured with KR/JP/CN patents
gp-cli lookup KR102345678B1 --language en --fields claims_structured
```
- Querying a non-English patent without `--language en` will produce a `TRANSLATED_PAGE_NO_TYPE_INFO` warning.
- `type` and `depends_on` are always absent on translated pages — this is a limitation of Google Patents' source markup, not a tool bug.

### `_warnings` Codes
| Code | Meaning |
|------|---------|
| `TRANSLATED_PAGE_NO_TYPE_INFO` | Translated or non-English native page — no dependency structure available |
| `SUSPICIOUSLY_SHORT_CLAIM_TEXT` | Claim text ≤ 20 chars — likely a translation artifact |

---

## `--input-file` Format

```
# Comments are ignored
US9735861B2
EP3123456B1
KR102345678B1   # inline comments are NOT supported — entire line is treated as the patent number
JP6789012B2

CN109876543B
```

- Blank lines are ignored
- Lines starting with `#` are ignored
- Inline comments are not supported — the entire line is treated as a patent number

---

## Bulk Mode Output (TSV)

When collecting multiple patents into a TSV file:

```sh
# With --input-file: header is managed automatically (first row only)
gp-cli lookup --input-file patents.txt --format tsv --output-dir ./results
# Each patent saved separately: results/US9735861B2.tsv, ...

# Aggregating to stdout (header managed automatically)
gp-cli lookup --input-file patents.txt --format tsv > family_summary.tsv
```

For family grouping, TSV output has `group_id` and `patent_number` columns:

```sh
gp-cli family-group --input-file patents.txt --format tsv > family_groups.tsv
```

---

## Common Fields

```sh
gp-cli fields   # list all available fields
```

Frequently used fields: `title`, `assignee`, `abstract`, `claims`, `filing_date`,
`publication_date`, `backward_citations`, `forward_citations`,
`family_applications`, `patent_url`

There is no top-level `priority_date`, `legal_status`, or `pdf_url` field.
Legal status appears inside `family_applications` entries when Google Patents
provides it. Use `gp-cli download` for PDFs.

Opt-in structured fields (must be requested explicitly):
- `claims_structured` — per-claim number, type, dependency, and text
- `description_structured` — structured text of description paragraphs
