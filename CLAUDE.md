# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```sh
go build ./...                          # compile all packages
go build -o gp-cli ./cmd/gp-cli/       # build the binary
go run ./cmd/gp-cli/ lookup US12514139B2
go vet ./...
```

## Testing

```sh
# Unit tests (no network)
go test ./internal/fetcher/ ./internal/formatter/ -v

# Integration tests (hits live Google Patents — ~11s)
go test -tags integration ./tests/integration/ -v -timeout 300s
```

Unit tests cover `NormalizeForURL` (11 cases) and formatter output (`Render`, `PrintErrorJSON`, `SelectFields`).
Integration tests build the binary and run it against real patents — one per country (US/EP/KR/JP/CN/GB/DE/AU/BR/MX/TW/WO) from the US8725880B2 family, plus NOT_FOUND/minify/field-filter cases.

The release binary is built by GitHub Actions (`.github/workflows/release.yml`) on `v*` tag pushes; it cross-compiles for darwin/arm64, darwin/amd64, linux/amd64, and windows/amd64.

## Architecture

All user-facing logic lives in `cmd/gp-cli/main.go` as Cobra subcommands. The four internal packages each have a single responsibility:

| Package | Role |
|---------|------|
| `internal/fetcher` | HTTP client (`FetchHTML`, `FetchBinary`). Normalizes patent numbers for the Google Patents URL format (US 6-digit sequences are zero-padded to 7 digits). Supports proxy and custom CA via `Options`. `Options.Language` appends `/<lang>` to the URL for Google machine translation. Detects bot-block pages (HTTP 200 with "Sorry..." title or "unusual traffic" body) via `isBotBlocked` and returns `BotBlockedError`. |
| `internal/parser` | Parses raw HTML → `PatentData`. Uses `goquery` for DOM traversal and regex for `<meta>` / canonical URL extraction. `ParseImageURLs` deduplicates thumbnail vs. high-res by taking the last occurrence of each filename. Handles translated (`/en`) pages by stripping `span.google-src-text` / `span.notranslate` before text extraction. Populates `ClaimsStructured` (`[]StructuredClaim`) and `DescriptionStructured` (`[]StructuredDescription`) on every parse, but these are only surfaced in output when explicitly requested. `StructuredClaim` carries `type` ("independent"/"dependent") and `depends_on` (parent claim numbers) derived from `<claim-ref idref="CLM-XXXXX">` tags in US/EP patent-office HTML; omitted for translated pages where this markup is absent. |
| `internal/formatter` | Converts `PatentData` → `DataMap` (ordered map) → rendered output (`json`/`text`/`tsv`). `text` and `tsv` files are written with a UTF-8 BOM for Excel/Notepad compatibility. `DataMap` has a custom `MarshalJSON` to preserve insertion order. `StructuredFieldNames` lists opt-in fields (`claims_structured`, `description_structured`) excluded from default output; `AddStructuredFields` appends them on demand and automatically runs `claimsStructuredWarnings` to populate `DataMap.warnings`. JSON output wraps these as `_warnings` at the envelope level (omitted when empty). |
| `internal/config` | Minimal hand-rolled TOML parser/writer for `~/.patent-cli.toml`. Sections: `[proxy]`, `[ssl]`, `[request]`. |
| `internal/updater` | Self-update: queries GitHub Releases API, downloads the platform-appropriate asset (tar.gz or zip), extracts the binary, and atomically replaces the running executable. Windows handles the rename-while-running constraint by moving the current exe to `.old` first. |

### Request flow

```
main.go subcommand
  └─ loadRequestOpts()          ← applies config.delay_ms sleep here
       └─ fetcher.FetchHTML()   ← GET patents.google.com/patent/<normalized>
            └─ parser.ParseAll()
                 └─ formatter.ToDataMap() → Render()
```

`loadRequestOpts` is the single place where config (proxy, CA bundle, delay) is applied to every outbound request.

## Config file

`~/.patent-cli.toml` — all fields optional, defaults to no proxy / no delay:

```toml
[proxy]
https = "http://proxy.corp:8080"
http  = "http://proxy.corp:8080"

[ssl]
ca_bundle = "/path/to/ca.pem"

[request]
delay_ms = 500   # sleep before each HTTP request; 0 = disabled
```

## Release & install scripts

Three install scripts are uploaded as release assets alongside the binaries:

| Script | Platform |
|--------|----------|
| `install.sh` | macOS / Linux |
| `install.ps1` | Windows PowerShell |
| `install.bat` | Windows CMD (requires Windows 10 build 1803+) |

All scripts install to a user-local directory (`~/.local/bin` on Unix, `%LOCALAPPDATA%\gp-cli` on Windows) and add it to the user's `PATH` without requiring admin rights.

## Key conventions

- **Patent number normalization**: done in `fetcher.NormalizeForURL`; strip non-alphanumeric, uppercase, then zero-pad US 6-digit sequences. All other formats pass through unchanged.
- **Error output**: structured `{"ok": false, "error": {"type": "…", "message": "…"}}` JSON goes to **stderr**. Exit codes: 0=ok, 1=general, 4=not-found, 6=server-error. Progress messages go to stderr and are suppressed by `--quiet`. Bot-block detection returns `SERVER_ERROR` (exit 6).
- **`--language` flag** (`lookup` command): appends `/<lang>` to the Google Patents URL to fetch machine-translated pages (e.g. `--language en`). Parser automatically strips original-language spans from translated pages.
- **Opt-in structured fields**: `claims_structured` and `description_structured` are not included in default output. They are populated and returned only when explicitly requested via `--field claims_structured` or `--fields claims_structured`. Listed separately in `gp-cli fields` output under "Opt-in structured fields". `claims_structured` schema: `{number, type?, depends_on?, text}` — `type`/`depends_on` present for US/EP patents only.
- **`_warnings` field**: present in JSON envelope alongside `ok` and `results` when `claims_structured` is requested and data quality issues are detected. Two codes: `TRANSLATED_PAGE_NO_TYPE_INFO` (no claim has `type` — translated page), `SUSPICIOUSLY_SHORT_CLAIM_TEXT` (claim text ≤ 20 chars). Omitted entirely (`omitempty`) when no issues found. Not emitted in `text`/`tsv` output.
- **`--output-dir` flag** (`lookup` command): saves result to file and suppresses stdout. `--field` with `--output-dir` saves the field value as a `.txt` file.
- **`--no-header` flag** (`lookup` command): omits the TSV header row; useful when appending rows from a loop: first call without `--no-header`, subsequent calls with `--no-header`.
- **images filenames**: saved as `{PATENT_NUMBER}_fig01.png`, `{PATENT_NUMBER}_fig02.png`, … to avoid collisions when downloading figures from multiple patents into the same directory.
- **GitHub repo constant**: hardcoded as `"noaa/patent-cli"` in `install.*` scripts and `internal/updater/updater.go`. Update all four locations when the repo is renamed.
- **Version constant**: `const version = "0.1.0"` in `cmd/gp-cli/main.go`. The release workflow injects it at build time via `-ldflags "-X main.version=${{ github.ref_name }}"`.
