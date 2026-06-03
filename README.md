# gp-cli — Google Patents CLI

`gp-cli` is a fast Google Patents CLI built for patent review, claim analysis,
and AI coding-agent workflows.

What it does:

- Fetches structured patent metadata, claims, descriptions, citations, and family applications
- Downloads patent PDFs and high-resolution drawing figures
- Processes patent-number lists in bulk with request delays for safer automation
- Groups input patents by patent family while skipping already discovered family members
- Exposes MCP tools for Claude Code, Codex CLI, Gemini CLI, and other agent runtimes

Designed for terminal use first: clean JSON/TSV/text output, script-friendly
exit codes, and no headless browser dependency.

> 한국어 문서: [README.ko.md](README.ko.md)

---

## Installation

### macOS

Open Terminal and run:

```sh
curl -fsSL https://raw.githubusercontent.com/noaa/patent-cli/main/install.sh | sh
```

Open a **new terminal window** after installation to use the `gp-cli` command.

---

### Windows

**Option A — PowerShell** (Start menu → search "PowerShell"):

```powershell
irm https://raw.githubusercontent.com/noaa/patent-cli/main/install.ps1 | iex
```

**Option B — Command Prompt (CMD)** (Start menu → search "cmd"):

```cmd
curl -fsSL https://raw.githubusercontent.com/noaa/patent-cli/main/install.bat -o "%TEMP%\gp-cli-install.bat" && "%TEMP%\gp-cli-install.bat"
```

> Requires Windows 10 build 1803 or later (curl and tar are built in).

Open a **new terminal window** after installation to use the `gp-cli` command.

---

### Manual Installation

If the automatic installer doesn't work, download the binary directly from the [Releases page](https://github.com/noaa/patent-cli/releases):

| OS | File |
|----|------|
| macOS (M1/M2/M3) | `gp-cli-darwin-arm64.tar.gz` |
| macOS (Intel) | `gp-cli-darwin-amd64.tar.gz` |
| Windows | `gp-cli-windows-amd64.exe.zip` |
| Linux | `gp-cli-linux-amd64.tar.gz` |

Extract the archive and place `gp-cli` (or `gp-cli.exe`) anywhere in your `PATH`.

---

### Build from Source

Requires [Go 1.21+](https://go.dev/dl/).

```sh
git clone https://github.com/noaa/patent-cli.git
cd patent-cli
go build -o gp-cli ./cmd/gp-cli/
```

Move the binary to a directory in your `PATH`:

```sh
# macOS / Linux
mv gp-cli ~/.local/bin/

# Windows (PowerShell)
Move-Item gp-cli.exe $env:LOCALAPPDATA\gp-cli\gp-cli.exe
```

---

## Claude Code Plugin

> **Why patent-cli?** Pure HTTP fetch — no headless browser overhead.
> Single MCP call returns results directly; no separate query step needed.
> Significantly faster than browser-based patent plugins.

### Installation

```sh
# 1. Install the gp-cli binary (if not already installed)
curl -fsSL https://github.com/noaa/patent-cli/releases/latest/download/install.sh | sh

# 2. Add the marketplace (required before plugin install)
claude plugin marketplace add noaa/patent-cli

# 3. Install the Claude Code plugin
claude plugin install https://github.com/noaa/patent-cli
```

### Available skills and MCP tools

| Skill | MCP Tool | Description |
|-------|----------|-------------|
| `/patent-lookup` | `patent_lookup` | Fetch patent details by number |
| `/patent-fields` | `patent_fields` | List all available fields |

---

## Codex Plugin

```sh
# 1. Install the gp-cli binary (if not already installed)
curl -fsSL https://github.com/noaa/patent-cli/releases/latest/download/install.sh | sh

# 2. Clone this repository
git clone https://github.com/noaa/patent-cli.git
cd patent-cli

# 3. Register the local Codex marketplace
codex plugin marketplace add ./codex-marketplace

# 4. Install the plugin from that marketplace
codex plugin add patent-cli@patent-cli-local

# 5. Verify installation
codex plugin list
```

Same skills and MCP tools as the Claude Code plugin. Codex app and CLI share the same bundle — installing via CLI makes the plugin available in the desktop app automatically.

For local development, edit files under `codex-plugin/`, then reinstall:

```sh
codex plugin add patent-cli@patent-cli-local
```

Start a new Codex thread after reinstalling so the updated skills and MCP configuration are loaded.

---

## Antigravity Plugin

```sh
# 1. Install the gp-cli binary (if not already installed)
curl -fsSL https://github.com/noaa/patent-cli/releases/latest/download/install.sh | sh

# 2. Add MCP server to ~/.config/antigravity/settings.json
# Add the following under "mcpServers":
#   "patent-cli": { "command": "gp-cli", "args": ["mcp"], "transport": "stdio" }

# 3. Copy skills to local skills directory
cp -r antigravity-plugin/skills/* ~/.config/antigravity/skills/
```

---

## Claude Desktop Extension

Drag-and-drop installation via `.mcpb` bundle — no binary pre-install required; the platform binary is bundled inside.

```sh
# Download the .mcpb for your platform from GitHub Releases, then:
# Claude Desktop → Settings → Extensions → drag patent-cli-<platform>.mcpb onto the window
```

Or install via deep link (replace `<version>` and `<platform>`):
```
claude://install-extension?url=https://github.com/noaa/patent-cli/releases/download/<version>/patent-cli-<platform>.mcpb
```

Available `.mcpb` assets per release:

| File | Platform |
|------|----------|
| `patent-cli-darwin-arm64.mcpb` | macOS Apple Silicon |
| `patent-cli-darwin-amd64.mcpb` | macOS Intel |
| `patent-cli-windows-amd64.mcpb` | Windows x64 |

---

## Usage

### Look Up Patent Metadata

```sh
# JSON output (default)
gp-cli lookup US12514139B2

# Human-readable text
gp-cli lookup US12514139B2 --format text

# Select specific fields
gp-cli lookup US12514139B2 --fields title,assignee,filing_date

# Single field value (plain text)
gp-cli lookup US12514139B2 --field title

# Compact JSON (no indentation)
gp-cli lookup US12514139B2 --minify

# Suppress progress messages (useful in scripts)
gp-cli lookup US12514139B2 --quiet

# Fetch machine-translated English version of a non-English patent
gp-cli lookup KR102355140B1 --language en

# Structured claims and description (opt-in fields)
gp-cli lookup US12514139B2 --fields claims_structured,description_structured

# Save result to a file (suppresses stdout)
gp-cli lookup US12514139B2 --output-dir ./output

# Save a single field to a file
gp-cli lookup US12514139B2 --field claims --output-dir ./output
```

### Download PDF

```sh
gp-cli download US12514139B2
gp-cli download US12514139B2 --output-dir ./pdfs
```

### Download Drawing Images

```sh
gp-cli images US12514139B2
gp-cli images US12514139B2 --output-dir ./figures
# Saved as US12514139B2_fig01.png, US12514139B2_fig02.png, ...
```

### Group Patents by Family

```sh
# Group explicit patent numbers
gp-cli family-group US8725880B2 US8704863B2 US9735861B2

# Read patent numbers from a file
gp-cli family-group --input-file patents.txt --format text

# TSV output for spreadsheets
gp-cli family-group --input-file patents.txt --format tsv --output-dir ./groups
```

`family-group` fetches each patent's `family_applications` and groups input
patents that belong to the same family. If a later input patent was already
identified as a member of a fetched family, it is skipped to reduce Google
Patents requests. A random 1000-1500 ms delay is applied between fetches; use
`--delay MS` to set an explicit delay.

### List Available Fields

```sh
gp-cli fields
```

---

## Output Formats

| Option | Description |
|--------|-------------|
| `--format json` | JSON (default) — `lookup` uses `{"ok": true, "results": {...}}`; `family-group` uses `{"ok": true, "groups": [...], "summary": {...}}` |
| `--format text` | Label + value text |
| `--format tsv` | Tab-separated (paste into Excel) |
| `--output-dir DIR` | Save output to a file; suppresses stdout |
| `--no-header` | Omit TSV header row (useful when appending rows in a loop) |
| `--minify` | Compact JSON output (no indentation) |

## Global Flags

| Flag | Description |
|------|-------------|
| `--quiet`, `-q` | Suppress progress messages on stderr |
| `--minify` | Compact JSON output (no indentation) |
| `--verbose`, `-v` | Print debug logs to stderr |
| `--language LANG` | Fetch via Google machine translation (e.g. `en`). Useful for non-English patents. |

---

## Error Handling

Errors are written to **stderr** as structured JSON and the process exits with a typed exit code.

```json
{
  "ok": false,
  "error": {
    "type": "NOT_FOUND",
    "message": "patent not found: US99999999X1"
  }
}
```

| Exit code | Meaning |
|-----------|---------|
| `0` | Success |
| `1` | General error |
| `4` | Patent not found |
| `6` | Server error (bot-block, 5xx) |

Because errors go to stderr, stdout is always clean data — safe to redirect to a file or pipe to `jq` without contamination.

```sh
# Safe: only TSV data reaches the file; errors appear in the terminal
while IFS= read -r num; do
  gp-cli lookup "$num" --format tsv --quiet
done < list.txt > results.tsv
```

---

## Scripting & Pipelines

### Batch TSV with a single header row

```sh
first=1
while IFS= read -r num; do
  if [ $first -eq 1 ]; then
    gp-cli lookup "$num" --fields publication_number,title,assignee --format tsv --quiet --delay 1000
    first=0
  else
    gp-cli lookup "$num" --fields publication_number,title,assignee --format tsv --quiet --no-header --delay 1000
  fi
done < patent_list.txt > summary.tsv
```

### Extract citation patent numbers with `jq`

```sh
gp-cli lookup US8725880B2 --field backward_citations --quiet \
  | jq -r '.[].publication_number'
```

### Filter independent claims

```sh
gp-cli lookup US12514139B2 --fields claims_structured --quiet \
  | jq '[.results.claims_structured[] | select(.type == "independent")]'
```

### Get claim 1 and all claims that directly depend on it

```sh
gp-cli lookup US12514139B2 --fields claims_structured --quiet \
  | jq '[.results.claims_structured[] | select(.number == "1" or (.depends_on // [] | any(. == "1")))]'
```

### Batch figure download (no filename collisions)

```sh
# Files saved as US11125686B2_fig01.png, EP3025568B1_fig01.png, ...
for num in US11125686B2 EP3025568B1; do
  gp-cli images "$num" --output-dir ./figures --quiet --delay 1000
done
```

### Group a patent list by family

```sh
gp-cli family-group --input-file patent_list.txt --format tsv --quiet > family_groups.tsv
```

JSON output includes a `summary.fetch_count` value so scripts can verify the
skip optimization:

```sh
gp-cli family-group US8725880B2 US8704863B2 US9735861B2 --minify --quiet \
  | jq '.summary.fetch_count'
```

---

## Opt-in Structured Fields

Two fields are excluded from default output and must be requested explicitly:

| Field | Schema |
|-------|--------|
| `claims_structured` | `{"number": "1", "type": "independent"\|"dependent", "depends_on": ["N"], "text": "…"}` |
| `description_structured` | `{"number": "1", "id": "…", "text": "…"}` |

`type` and `depends_on` are populated for US/EP patents (detected from HTML `<claim-ref>` tags). They are omitted for translated pages and non-US/EP patents where this markup is absent.

```sh
gp-cli lookup US12514139B2 --fields claims_structured
gp-cli lookup US12514139B2 --fields claims_structured,description_structured
gp-cli fields   # lists all available fields including opt-in ones
```

### Data quality warnings (`_warnings`)

When `claims_structured` is requested, the JSON envelope may include a `_warnings` array alongside `ok` and `results`. The array is omitted entirely when no issues are detected.

```json
{
  "ok": true,
  "results": { "claims_structured": [...] },
  "_warnings": [
    {
      "field": "claims_structured",
      "code": "TRANSLATED_PAGE_NO_TYPE_INFO",
      "message": "Claim type and dependency data unavailable; page was served as a machine translation"
    },
    {
      "field": "claims_structured",
      "code": "SUSPICIOUSLY_SHORT_CLAIM_TEXT",
      "message": "Claim(s) 1 have text ≤ 20 chars; likely translation artifacts — verify against source"
    }
  ]
}
```

| Warning code | Meaning |
|---|---|
| `TRANSLATED_PAGE_NO_TYPE_INFO` | No claim has `type`; page was machine-translated — `depends_on` also absent |
| `SUSPICIOUSLY_SHORT_CLAIM_TEXT` | One or more claim texts are ≤ 20 chars; likely a translation artifact |

---

## Configuration (Proxy / CA Certificate)

For corporate networks with a proxy:

```sh
gp-cli configure
```

Config file location: macOS `~/Library/Application Support/patent-cli/config.toml`, Linux `~/.config/patent-cli/config.toml`

```toml
[proxy]
https = "http://proxy.corp:8080"
http  = "http://proxy.corp:8080"

[ssl]
ca_bundle = "/path/to/ca.pem"

[request]
delay_ms = 500   # sleep before each request — use in loops to avoid bot detection
```

---

## Version & Update

```sh
gp-cli --version
gp-cli update           # check and update automatically
gp-cli update --check   # only print version info
```

---

## Development

```sh
# Build
go build ./...
go build -o gp-cli ./cmd/gp-cli/

# Unit tests (no network)
go test ./internal/fetcher/ ./internal/formatter/ ./internal/parser/ -v

# Integration tests (hits live Google Patents — ~15 s)
go test -tags integration ./tests/integration/ -v -timeout 300s

# MCP server (stdio — input prompt means ready)
gp-cli mcp
```

Unit tests cover: patent number normalization, bot-block detection, JSON/text/TSV rendering, structured field warnings, and HTML parsing of US/translated claims and description paragraphs.

Integration tests build the binary and run it against real patents — one per country (US/EP/KR/JP/CN/GB/DE/AU/BR/MX/TW/WO), plus NOT\_FOUND, `--minify`, `--fields`, `--language`, and `claims_structured` / `description_structured` cases.
