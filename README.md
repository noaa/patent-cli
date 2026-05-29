# gp-cli — Google Patents CLI

A command-line tool to query patent metadata and download PDFs and drawing images from Google Patents.

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

## Usage

### Look Up Patent Metadata

```sh
# JSON output (default)
gp-cli lookup US12514139B2

# Human-readable text
gp-cli lookup US12514139B2 --format text

# Select specific fields
gp-cli lookup US12514139B2 --fields title,assignee,filing_date

# Single field
gp-cli lookup US12514139B2 --field title

# Compact JSON (no indentation)
gp-cli lookup US12514139B2 --minify

# Suppress progress messages (useful in scripts)
gp-cli lookup US12514139B2 --quiet

# WO/PCT patent
gp-cli lookup WO2022123456A1

# KR patent
gp-cli lookup KR102355140B1
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
```

### List Available Fields

```sh
gp-cli fields
```

---

## Output Formats

| Option | Description |
|--------|-------------|
| `--format json` | JSON (default) — wrapped in `{"ok": true, "results": {...}}` |
| `--format text` | Label + value text |
| `--format tsv` | Tab-separated (paste into Excel) |
| `--output-dir DIR` | Save output to a file |
| `--minify` | Compact JSON output (no indentation) |

## Global Flags

| Flag | Description |
|------|-------------|
| `--quiet`, `-q` | Suppress progress messages on stderr |
| `--minify` | Compact JSON output (no indentation) |
| `--verbose`, `-v` | Print debug logs to stderr |

---

## Configuration (Proxy / CA Certificate)

For corporate networks with a proxy:

```sh
gp-cli configure
```

Config file location: `~/.patent-cli.toml`

---

## Version

```sh
gp-cli --version
```
