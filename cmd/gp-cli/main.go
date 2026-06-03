package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/area99/patent-cli/internal/config"
	"github.com/area99/patent-cli/internal/fetcher"
	"github.com/area99/patent-cli/internal/formatter"
	"github.com/area99/patent-cli/internal/mcpserver"
	"github.com/area99/patent-cli/internal/parser"
	"github.com/area99/patent-cli/internal/updater"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

const version = "0.1.3"

const (
	exitOK           = 0
	exitGeneralError = 1
	exitNotFound     = 4
	exitServerError  = 6
)

var (
	verbose bool
	quiet   bool
	minify  bool
)

// nonEnglishCountryCodes lists country prefixes where patents are typically in a non-English language.
var nonEnglishCountryCodes = map[string]bool{
	"KR": true, "JP": true, "CN": true, "DE": true, "FR": true,
	"IT": true, "ES": true, "RU": true, "NL": true, "SE": true,
	"FI": true, "DK": true, "NO": true, "PL": true, "CZ": true,
	"HU": true, "PT": true, "RO": true, "TW": true, "BR": true,
	"MX": true, "AR": true, "IN": true, "IL": true, "TR": true,
}

func main() {
	root := &cobra.Command{
		Use:   "gp-cli",
		Short: "Google Patents CLI — fetch patent metadata by patent number",
		Long: `Google Patents CLI — fetch patent metadata by patent number.

Commands:
  lookup        Fetch metadata for a patent number (or bulk via --input-file)
  download      Download the patent PDF (or bulk via --input-file)
  images        Download high-resolution figure images (or bulk via --input-file)
  family-group  Group a list of patents by patent family
  fields        List all available output fields
  configure     Set proxy / CA-cert options

Quick start:
  gp-cli lookup US12514139B2
  gp-cli lookup US20250350789 --format text
  gp-cli lookup US20250350789 --fields title,assignee
  gp-cli download US9735861 --output-dir ./pdfs
  gp-cli images US11125686B2 --output-dir ./figs
  gp-cli lookup --input-file patents.txt --format tsv --output-dir ./results
  gp-cli family-group US8725880B2 US8704863B2 US9735861B2
  gp-cli family-group --input-file patents.txt --format text
  gp-cli fields`,
		Version:          version,
		SilenceUsage:     true,
		SilenceErrors:    true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {},
	}

	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Print progress and debug logs to stderr")
	root.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress progress messages on stderr")
	root.PersistentFlags().BoolVar(&minify, "minify", false, "Compact JSON output (no indentation)")

	root.AddCommand(
		lookupCmd(),
		downloadCmd(),
		imagesCmd(),
		familyGroupCmd(),
		fieldsCmd(),
		configureCmd(),
		updateCmd(),
		mcpCmd(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func logf(format string, args ...interface{}) {
	if verbose {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}

func progressf(format string, args ...interface{}) {
	if !quiet {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}

func exitError(errType, message string, code int) {
	formatter.PrintErrorJSON(errType, message)
	os.Exit(code)
}

func needsStructured(single string, fields []string) bool {
	for _, name := range formatter.StructuredFieldNames {
		if single == name {
			return true
		}
		for _, f := range fields {
			if f == name {
				return true
			}
		}
	}
	return false
}

// buildOpts constructs request options from config without applying any delay.
func buildOpts(timeout time.Duration) fetcher.Options {
	cfg, _ := config.Load()
	reqOpts := config.GetRequestOptions(cfg)
	return fetcher.Options{
		Timeout:  timeout,
		Proxies:  reqOpts.Proxies,
		CABundle: reqOpts.CABundle,
	}
}

// sleepDelay applies a pre-request delay.
// In bulk mode with no explicit delay, jitters randomly between 1000–1500 ms to avoid bot detection.
// An explicit --delay flag or config delay_ms always takes precedence.
func sleepDelay(explicitMs int, isBulk bool) {
	cfg, _ := config.Load()
	var ms int
	switch {
	case explicitMs > 0:
		ms = explicitMs
	case isBulk:
		ms = 1000 + rand.Intn(501)
	default:
		ms = cfg.Request.DelayMs
	}
	if ms > 0 {
		logf("delay: %dms", ms)
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
}

// readPatentFile reads patent numbers from a file, one per line.
// Blank lines and lines beginning with '#' are skipped.
func readPatentFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var numbers []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		numbers = append(numbers, line)
	}
	return numbers, scanner.Err()
}

// isNonEnglishPatent returns true when the patent number's country prefix suggests
// the native page language is not English.
func isNonEnglishPatent(patentNumber string) bool {
	upper := strings.ToUpper(strings.TrimSpace(patentNumber))
	if len(upper) >= 2 {
		return nonEnglishCountryCodes[upper[:2]]
	}
	return false
}

// ── lookup ────────────────────────────────────────────────────────────────────

func lookupCmd() *cobra.Command {
	var (
		fmt_        string
		singleField string
		multiFields []string
		timeout     int
		outputDir   string
		language    string
		delayMs     int
		noHeader    bool
		inputFile   string
	)

	cmd := &cobra.Command{
		Use:   "lookup PATENT_NUMBER",
		Short: "Fetch Google Patents metadata for PATENT_NUMBER",
		Long: `Fetch Google Patents metadata for PATENT_NUMBER.
Use --input-file to process multiple patents from a file (one number per line).
In bulk mode a random 1000–1500 ms delay is applied automatically between requests.

Examples:
  gp-cli lookup US20250350789
  gp-cli lookup US12514139B2 --format text
  gp-cli lookup US20250350789 --field title
  gp-cli lookup US20250350789 --fields title,assignee,filing_date
  gp-cli lookup US20250350789 --format tsv
  gp-cli lookup US12514139B2 --output-dir ./output
  gp-cli lookup US12514139B2 --format text --output-dir ./output
  gp-cli lookup US12514139B2 --delay 2000        # 2 s delay — use in loops to avoid bot detection
  gp-cli lookup --input-file patents.txt --format tsv --output-dir ./results
  gp-cli lookup --input-file patents.txt --fields title,assignee --format json`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			patentNumbers, err := resolvePatentNumbers(args, inputFile)
			if err != nil {
				return err
			}
			isBulk := len(patentNumbers) > 1

			// Parse --fields list once, outside the loop.
			var fieldsList []string
			for _, token := range multiFields {
				for _, f := range strings.Split(token, ",") {
					f = strings.TrimSpace(f)
					if f != "" {
						fieldsList = append(fieldsList, f)
					}
				}
			}
			requestsStructured := needsStructured(singleField, fieldsList)

			var okCount, skipCount, errCount int
			firstTSVRow := true

		PATENTS:
			for i, patentNumber := range patentNumbers {
				if isBulk {
					progressf("[%d/%d] %s", i+1, len(patentNumbers), patentNumber)
				}

				// No delay before the very first request; random jitter on subsequent ones.
				if i > 0 || !isBulk {
					sleepDelay(delayMs, isBulk && i > 0)
				}

				opts := buildOpts(time.Duration(timeout) * time.Second)
				opts.Language = language

				html, fetchErr := fetcher.FetchHTML(patentNumber, opts)
				if fetchErr != nil {
					errType, message, code := classifyFetchError(fetchErr, patentNumber)
					if isBulk {
						formatter.PrintErrorJSON(errType, message)
						if errType == "NOT_FOUND" {
							progressf("  → NOT_FOUND (skipped)")
							skipCount++
						} else {
							progressf("  → %s: %s (skipped)", errType, message)
							errCount++
						}
						continue PATENTS
					}
					exitError(errType, message, code)
				}

				data := parser.ParseAll(html)
				dm := formatter.ToDataMap(data)

				if requestsStructured {
					formatter.AddStructuredFields(dm, data)
				}

				// Hint: non-English patent without --language en loses claim structure.
				if requestsStructured && language == "" && isNonEnglishPatent(patentNumber) {
					for _, w := range dm.Warnings() {
						if w.Code == "TRANSLATED_PAGE_NO_TYPE_INFO" {
							fmt.Fprintf(os.Stderr,
								"Hint [%s]: non-English patent — use --language en for English machine translation (claim type/dependency data unavailable on translated pages)\n",
								patentNumber)
							break
						}
					}
				}

				// --field: single plain-text value.
				if singleField != "" {
					v, ok := dm.Get(singleField)
					if !ok {
						fmt.Fprintf(os.Stderr, "Unknown field: %q. Run 'gp-cli fields' for available fields.\n", singleField)
						if isBulk {
							errCount++
							continue PATENTS
						}
						os.Exit(exitGeneralError)
					}
					if outputDir != "" {
						if err := os.MkdirAll(outputDir, 0755); err != nil {
							fmt.Fprintf(os.Stderr, "Save error: %v\n", err)
							if isBulk {
								errCount++
								continue PATENTS
							}
							os.Exit(exitGeneralError)
						}
						dest := filepath.Join(outputDir, patentNumber+".txt")
						if err := os.WriteFile(dest, []byte(formatter.FieldToString(v)), 0644); err != nil {
							fmt.Fprintf(os.Stderr, "Save error: %v\n", err)
							if isBulk {
								errCount++
								continue PATENTS
							}
							os.Exit(exitGeneralError)
						}
						progressf("Saved: %s", dest)
					} else {
						formatter.PrintField(v)
					}
					okCount++
					continue PATENTS
				}

				// Validate requested field names.
				for _, f := range fieldsList {
					if _, ok := dm.Get(f); !ok {
						fmt.Fprintf(os.Stderr, "Unknown field: %q. Run 'gp-cli fields' for available fields.\n", f)
						if isBulk {
							errCount++
							continue PATENTS
						}
						os.Exit(exitGeneralError)
					}
				}

				out := formatter.SelectFields(dm, fieldsList)

				// In bulk TSV mode: omit header after the first successful row.
				effectiveNoHeader := noHeader
				if isBulk && fmt_ == "tsv" && !firstTSVRow {
					effectiveNoHeader = true
				}

				if outputDir == "" {
					fmt.Println(formatter.Render(out, fmt_, minify, effectiveNoHeader))
				} else {
					saved, saveErr := formatter.SaveToDir(out, fmt_, outputDir, patentNumber)
					if saveErr != nil {
						fmt.Fprintf(os.Stderr, "Save error: %v\n", saveErr)
						if isBulk {
							errCount++
							continue PATENTS
						}
						os.Exit(exitGeneralError)
					}
					progressf("Saved: %s", saved)
				}

				if fmt_ == "tsv" {
					firstTSVRow = false
				}
				okCount++
			}

			if isBulk {
				progressf("Bulk complete: %d ok, %d skipped (NOT_FOUND), %d errors", okCount, skipCount, errCount)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&fmt_, "format", "f", "json", "Output format: json (default), text, or tsv")
	cmd.Flags().StringVar(&singleField, "field", "", "Print a single field value as plain text (overrides --format)")
	cmd.Flags().StringArrayVar(&multiFields, "fields", nil, "Comma-separated field list. Repeatable: --fields title --fields abstract")
	cmd.Flags().IntVarP(&timeout, "timeout", "t", 15, "HTTP request timeout in seconds")
	cmd.Flags().StringVar(&language, "language", "", "Fetch patent in specified language via Google machine translation (e.g. 'en')")
	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Save result to DIR; filename is derived from the patent number. Suppresses stdout output.")
	cmd.Flags().BoolVar(&noHeader, "no-header", false, "Omit the header row from TSV output (useful when appending in a loop)")
	cmd.Flags().IntVar(&delayMs, "delay", 0, "Wait N milliseconds before the request. Use in loops to avoid bot detection (overrides delay_ms in config)")
	cmd.Flags().StringVar(&inputFile, "input-file", "", "File with patent numbers to process in bulk (one per line; # comments and blank lines ignored)")
	return cmd
}

// ── download ──────────────────────────────────────────────────────────────────

func downloadCmd() *cobra.Command {
	var (
		outputDir string
		timeout   int
		delayMs   int
		inputFile string
	)

	cmd := &cobra.Command{
		Use:   "download PATENT_NUMBER",
		Short: "Download the patent PDF for PATENT_NUMBER (saved as <number>.pdf)",
		Long: `Download the patent PDF for PATENT_NUMBER (saved as <number>.pdf).
Use --input-file to download PDFs for multiple patents from a file (one number per line).
In bulk mode a random 1000–1500 ms delay is applied automatically between requests.

Examples:
  gp-cli download US9735861
  gp-cli download US9735861 --output-dir ./pdfs
  gp-cli download --input-file patents.txt --output-dir ./pdfs`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			patentNumbers, err := resolvePatentNumbers(args, inputFile)
			if err != nil {
				return err
			}
			isBulk := len(patentNumbers) > 1

			if err := os.MkdirAll(outputDir, 0755); err != nil {
				exitError("ERROR", fmt.Sprintf("failed to create directory: %v", err), exitGeneralError)
			}

			var okCount, skipCount, errCount int

		PATENTS:
			for i, patentNumber := range patentNumbers {
				if isBulk {
					progressf("[%d/%d] %s", i+1, len(patentNumbers), patentNumber)
				}

				if i > 0 || !isBulk {
					sleepDelay(delayMs, isBulk && i > 0)
				}

				opts := buildOpts(15 * time.Second)
				html, fetchErr := fetcher.FetchHTML(patentNumber, opts)
				if fetchErr != nil {
					errType, message, code := classifyFetchError(fetchErr, patentNumber)
					if isBulk {
						formatter.PrintErrorJSON(errType, message)
						if errType == "NOT_FOUND" {
							progressf("  → NOT_FOUND (skipped)")
							skipCount++
						} else {
							progressf("  → %s: %s (skipped)", errType, message)
							errCount++
						}
						continue PATENTS
					}
					exitError(errType, message, code)
				}

				data := parser.ParseAll(html)
				pdfURL := data.PDFURL
				if pdfURL == "" {
					if isBulk {
						formatter.PrintErrorJSON("NOT_FOUND", "PDF link not found for: "+patentNumber)
						progressf("  → NOT_FOUND: no PDF link (skipped)")
						skipCount++
						continue PATENTS
					}
					exitError("NOT_FOUND", "PDF link not found for: "+patentNumber, exitNotFound)
				}

				pubNumber := data.PublicationNumber
				if pubNumber == "" {
					pubNumber = strings.ToUpper(strings.ReplaceAll(patentNumber, "-", ""))
				}

				dest := filepath.Join(outputDir, pubNumber+".pdf")
				logf("PDF URL: %s", pdfURL)
				progressf("Downloading: %s", pdfURL)

				dlOpts := buildOpts(time.Duration(timeout) * time.Second)
				if dlErr := fetcher.FetchBinary(pdfURL, dest, dlOpts); dlErr != nil {
					if isBulk {
						formatter.PrintErrorJSON("ERROR", fmt.Sprintf("PDF download failed: %v", dlErr))
						progressf("  → download failed: %v (skipped)", dlErr)
						errCount++
						continue PATENTS
					}
					exitError("ERROR", fmt.Sprintf("PDF download failed: %v", dlErr), exitGeneralError)
				}

				progressf("Saved: %s", dest)
				okCount++
			}

			if isBulk {
				progressf("Bulk complete: %d ok, %d skipped (NOT_FOUND), %d errors", okCount, skipCount, errCount)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", ".", "Directory to save the PDF")
	cmd.Flags().IntVarP(&timeout, "timeout", "t", 60, "HTTP request timeout in seconds")
	cmd.Flags().IntVar(&delayMs, "delay", 0, "Wait N milliseconds before the request. Use in loops to avoid bot detection (overrides delay_ms in config)")
	cmd.Flags().StringVar(&inputFile, "input-file", "", "File with patent numbers to download in bulk (one per line; # comments and blank lines ignored)")
	return cmd
}

// ── images ────────────────────────────────────────────────────────────────────

func imagesCmd() *cobra.Command {
	var (
		outputDir string
		timeout   int
		delayMs   int
		inputFile string
	)

	cmd := &cobra.Command{
		Use:   "images PATENT_NUMBER",
		Short: "Download high-resolution figure images for PATENT_NUMBER",
		Long: `Download high-resolution figure images for PATENT_NUMBER.
Files are saved as <PATENT_NUMBER>_fig01.png, <PATENT_NUMBER>_fig02.png, ...
Use --input-file to download figures for multiple patents from a file (one number per line).
In bulk mode a random 1000–1500 ms delay is applied automatically between patents.

Examples:
  gp-cli images US11125686B2
  gp-cli images KR102355140B1 --output-dir ./figs
  gp-cli images --input-file patents.txt --output-dir ./figures`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			patentNumbers, err := resolvePatentNumbers(args, inputFile)
			if err != nil {
				return err
			}
			isBulk := len(patentNumbers) > 1

			if err := os.MkdirAll(outputDir, 0755); err != nil {
				exitError("ERROR", fmt.Sprintf("failed to create directory: %v", err), exitGeneralError)
			}

			var okCount, skipCount, errCount int

		PATENTS:
			for i, patentNumber := range patentNumbers {
				if isBulk {
					progressf("[%d/%d] %s", i+1, len(patentNumbers), patentNumber)
				}

				if i > 0 || !isBulk {
					sleepDelay(delayMs, isBulk && i > 0)
				}

				opts := buildOpts(15 * time.Second)
				html, fetchErr := fetcher.FetchHTML(patentNumber, opts)
				if fetchErr != nil {
					errType, message, code := classifyFetchError(fetchErr, patentNumber)
					if isBulk {
						formatter.PrintErrorJSON(errType, message)
						if errType == "NOT_FOUND" {
							progressf("  → NOT_FOUND (skipped)")
							skipCount++
						} else {
							progressf("  → %s: %s (skipped)", errType, message)
							errCount++
						}
						continue PATENTS
					}
					exitError(errType, message, code)
				}

				data := parser.ParseAll(html)
				pubNumber := data.PublicationNumber
				if pubNumber == "" {
					pubNumber = strings.ToUpper(strings.ReplaceAll(patentNumber, "-", ""))
				}

				urls := parser.ParseImageURLs(html)
				if len(urls) == 0 {
					if isBulk {
						formatter.PrintErrorJSON("NOT_FOUND", "no figure images found for: "+patentNumber)
						progressf("  → NOT_FOUND: no images (skipped)")
						skipCount++
						continue PATENTS
					}
					exitError("NOT_FOUND", "no figure images found for: "+patentNumber, exitNotFound)
				}

				progressf("Found %d figure image(s) for %s.", len(urls), pubNumber)

				dlOpts := buildOpts(time.Duration(timeout) * time.Second)
				patentOK := true
				for j, imgURL := range urls {
					filename := fmt.Sprintf("%s_fig%02d.png", pubNumber, j+1)
					dest := filepath.Join(outputDir, filename)
					logf("Image URL: %s", imgURL)
					progressf("  [%d/%d] %s", j+1, len(urls), filename)

					if dlErr := fetcher.FetchBinary(imgURL, dest, dlOpts); dlErr != nil {
						fmt.Fprintf(os.Stderr, "  Warning: failed to download image %d: %v\n", j+1, dlErr)
						patentOK = false
						continue
					}
					progressf("Saved: %s", dest)
				}

				if patentOK {
					okCount++
				} else {
					errCount++
				}
			}

			if isBulk {
				progressf("Bulk complete: %d ok, %d skipped (NOT_FOUND), %d errors", okCount, skipCount, errCount)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", ".", "Directory to save images")
	cmd.Flags().IntVarP(&timeout, "timeout", "t", 30, "HTTP request timeout in seconds per image")
	cmd.Flags().IntVar(&delayMs, "delay", 0, "Wait N milliseconds before the request. Use in loops to avoid bot detection (overrides delay_ms in config)")
	cmd.Flags().StringVar(&inputFile, "input-file", "", "File with patent numbers to process in bulk (one per line; # comments and blank lines ignored)")
	return cmd
}

// ── fields ────────────────────────────────────────────────────────────────────

func fieldsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fields",
		Short: "List all available output fields and their labels",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, key := range formatter.FieldOrder {
				label := formatter.LabelFor(key)
				fmt.Printf("  %-25s %s\n", key, label)
			}
			fmt.Println()
			fmt.Println("Opt-in structured fields (use --field or --fields to request):")
			for _, key := range formatter.StructuredFieldNames {
				label := formatter.LabelFor(key)
				fmt.Printf("  %-25s %s\n", key, label)
			}
			return nil
		},
	}
}

// ── configure ─────────────────────────────────────────────────────────────────

func configureCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "configure",
		Short: "Interactively set proxy and CA-cert options and save to config file",
		Long: `Interactively set proxy and CA-cert options and save to config file.

Press Enter with no value to skip a field.
Existing values are shown as defaults.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			existing, _ := config.Load()
			scanner := bufio.NewScanner(os.Stdin)

			configPath, err := config.ConfigPath()
			if err != nil {
				return err
			}

			fmt.Println("Google Patent CLI — Configuration")
			fmt.Println(strings.Repeat("─", 40))
			fmt.Println("Config file:", configPath)
			fmt.Println()

			httpsProxy := prompt(scanner, "HTTPS proxy URL", existing.Proxy.HTTPS)
			httpProxy := prompt(scanner, "HTTP  proxy URL", existing.Proxy.HTTP)
			caBundle := prompt(scanner, "CA bundle file path", existing.SSL.CABundle)

			delayDefault := ""
			if existing.Request.DelayMs > 0 {
				delayDefault = fmt.Sprintf("%d", existing.Request.DelayMs)
			}
			delayStr := prompt(scanner, "Request delay in ms (0 = disabled)", delayDefault)
			delayMs := 0
			if delayStr != "" {
				fmt.Sscanf(delayStr, "%d", &delayMs)
				if delayMs < 0 {
					delayMs = 0
				}
			}

			newCfg := config.Config{
				Proxy:   config.ProxyConfig{HTTPS: httpsProxy, HTTP: httpProxy},
				SSL:     config.SSLConfig{CABundle: caBundle},
				Request: config.RequestConfig{DelayMs: delayMs},
			}

			if httpsProxy == "" && httpProxy == "" && caBundle == "" && delayMs == 0 {
				fmt.Println("\nNo values entered — config file not saved.")
				return nil
			}

			if err := config.Save(newCfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
			fmt.Println("\nConfig saved:", configPath)
			fmt.Println()
			if httpsProxy != "" {
				fmt.Println("  proxy.https =", httpsProxy)
			}
			if httpProxy != "" {
				fmt.Println("  proxy.http =", httpProxy)
			}
			if caBundle != "" {
				fmt.Println("  ssl.ca_bundle =", caBundle)
			}
			if delayMs > 0 {
				fmt.Printf("  request.delay_ms = %d\n", delayMs)
			}
			return nil
		},
	}
}

// ── update ────────────────────────────────────────────────────────────────────

func updateCmd() *cobra.Command {
	var checkOnly bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check for a newer release and update gp-cli in-place",
		Long: `Check GitHub for a newer release of gp-cli and, if found, download and
replace the running binary in-place.

Examples:
  gp-cli update           # check and update automatically
  gp-cli update --check   # only print version info, do not update`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(os.Stderr, "Current version: %s\n", version)
			fmt.Fprintf(os.Stderr, "Checking for updates...\n")

			latestTag, err := updater.LatestTag()
			if err != nil {
				return fmt.Errorf("could not reach GitHub: %w", err)
			}

			if !updater.HasUpdate(version, latestTag) {
				fmt.Fprintf(os.Stderr, "Already up to date (%s).\n", version)
				return nil
			}

			fmt.Fprintf(os.Stderr, "New version available: %s\n", latestTag)

			if checkOnly {
				fmt.Fprintf(os.Stderr, "Run 'gp-cli update' (without --check) to install.\n")
				return nil
			}

			if err := updater.Do(latestTag); err != nil {
				return fmt.Errorf("update failed: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Updated to %s. Restart your terminal if needed.\n", latestTag)
			return nil
		},
	}

	cmd.Flags().BoolVar(&checkOnly, "check", false, "Only check for a new version; do not download or install")
	return cmd
}

// ── mcp ───────────────────────────────────────────────────────────────────────

func mcpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Run as an MCP server over stdio (for Claude Code / Codex / Antigravity integration)",
		Long: `Start gp-cli as a Model Context Protocol (MCP) server.

Communicates over stdin/stdout using newline-delimited JSON-RPC.
Intended to be launched by an MCP host (Claude Code, Codex, Antigravity CLI, etc.)
via the mcpServers configuration, not run directly.

Exposed tools:
  patent_lookup   Fetch patent metadata by patent number
  patent_fields   List all available output field names

Example plugin.json entry:
  "mcpServers": {
    "patent-cli": { "command": "gp-cli", "args": ["mcp"] }
  }`,
		RunE: func(cmd *cobra.Command, args []string) error {
			srv := mcpserver.Build(version)
			session, err := srv.Connect(context.Background(), &mcp.StdioTransport{}, nil)
			if err != nil {
				return fmt.Errorf("mcp server connect: %w", err)
			}
			if err := session.Wait(); err != nil && !isClosedErr(err) {
				return fmt.Errorf("mcp server: %w", err)
			}
			return nil
		},
	}
}

// ── family-group ──────────────────────────────────────────────────────────────

type fgGroup struct {
	ID      int      `json:"id"`
	Patents []string `json:"patents"`
}

type fgError struct {
	Patent  string `json:"patent"`
	Message string `json:"message"`
}

type fgSummary struct {
	TotalInput  int `json:"total_input"`
	TotalGroups int `json:"total_groups"`
	TotalErrors int `json:"total_errors"`
	FetchCount  int `json:"fetch_count"`
}

type fgResult struct {
	OK      bool      `json:"ok"`
	Groups  []fgGroup `json:"groups"`
	Errors  []fgError `json:"errors,omitempty"`
	Summary fgSummary `json:"summary"`
}

func familyGroupCmd() *cobra.Command {
	var (
		fmt_      string
		timeout   int
		outputDir string
		delayMs   int
		noHeader  bool
		inputFile string
	)

	cmd := &cobra.Command{
		Use:   "family-group PATENT_NUMBER [PATENT_NUMBER...]",
		Short: "Group a list of patents by patent family",
		Long: `Group a list of patents by patent family.

For each patent in the input list, fetches family_applications to determine
which patents share the same patent family. Patents already identified as
family members are skipped — minimizing the total number of API calls.

A random 1000–1500 ms jitter is applied between requests automatically.
Use --delay to set an explicit delay.

Examples:
  gp-cli family-group US8725880B2 US8704863B2 US9735861B2
  gp-cli family-group --input-file patents.txt
  gp-cli family-group --input-file patents.txt --format text
  gp-cli family-group --input-file patents.txt --format tsv`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var patentNumbers []string
			if inputFile != "" {
				if len(args) > 0 {
					return fmt.Errorf("cannot combine PATENT_NUMBER arguments with --input-file")
				}
				nums, err := readPatentFile(inputFile)
				if err != nil {
					return fmt.Errorf("reading input file: %w", err)
				}
				if len(nums) == 0 {
					return fmt.Errorf("no patent numbers found in %q", inputFile)
				}
				patentNumbers = nums
			} else {
				if len(args) == 0 {
					return fmt.Errorf("at least one PATENT_NUMBER is required; use --input-file to read from a file")
				}
				patentNumbers = args
			}

			// norm → original: for dedup and matching
			normToOrig := make(map[string]string, len(patentNumbers))
			// preserve input order, deduplicate
			var ordered []string
			for _, p := range patentNumbers {
				n := normPatent(p)
				if _, seen := normToOrig[n]; !seen {
					normToOrig[n] = p
					ordered = append(ordered, p)
				}
			}
			patentNumbers = ordered

			var groups [][]string
			assigned := make(map[string]int) // norm → group index (-1 = error)
			var fetchErrors []fgError
			fetchCount := 0

			for _, patentNumber := range patentNumbers {
				norm := normPatent(patentNumber)
				if _, ok := assigned[norm]; ok {
					continue // already grouped or errored from a previous fetch
				}

				// delay before every request except the first
				if fetchCount > 0 {
					sleepDelay(delayMs, true)
				}
				progressf("[fetch %d] %s", fetchCount+1, patentNumber)
				fetchCount++

				opts := buildOpts(time.Duration(timeout) * time.Second)
				html, fetchErr := fetcher.FetchHTML(patentNumber, opts)
				if fetchErr != nil {
					errType, message, _ := classifyFetchError(fetchErr, patentNumber)
					progressf("  → %s: %s (skipped)", errType, message)
					fetchErrors = append(fetchErrors, fgError{Patent: patentNumber, Message: message})
					assigned[norm] = -1
					continue
				}

				data := parser.ParseAll(html)

				// Build set of normalized patent numbers in this family
				familyNorms := map[string]bool{norm: true}
				for _, fa := range data.FamilyApplications {
					if fa.PatentNumber != "" {
						familyNorms[normPatent(fa.PatentNumber)] = true
					}
				}

				// Collect all input patents that belong to this family
				groupIdx := len(groups)
				var members []string
				for _, p := range patentNumbers {
					pNorm := normPatent(p)
					if _, alreadyAssigned := assigned[pNorm]; alreadyAssigned {
						continue
					}
					if familyNorms[pNorm] {
						assigned[pNorm] = groupIdx
						members = append(members, p)
					}
				}
				if len(members) == 0 {
					// patent itself not found in its own family list (edge case)
					assigned[norm] = groupIdx
					members = []string{patentNumber}
				}
				groups = append(groups, members)
				progressf("  → group %d: %s", groupIdx+1, strings.Join(members, ", "))
			}

			// Build result struct
			result := fgResult{OK: true}
			result.Summary = fgSummary{
				TotalInput:  len(patentNumbers),
				TotalGroups: len(groups),
				TotalErrors: len(fetchErrors),
				FetchCount:  fetchCount,
			}
			for i, g := range groups {
				result.Groups = append(result.Groups, fgGroup{ID: i + 1, Patents: g})
			}
			if len(fetchErrors) > 0 {
				result.Errors = fetchErrors
			}

			// Render output
			output, err := renderFamilyGroup(result, fmt_, minify, noHeader)
			if err != nil {
				return err
			}

			if outputDir != "" {
				ext := map[string]string{"json": ".json", "text": ".txt", "tsv": ".tsv"}[fmt_]
				if ext == "" {
					ext = ".json"
				}
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					return fmt.Errorf("failed to create directory: %w", err)
				}
				dest := filepath.Join(outputDir, "family_groups"+ext)
				if err := os.WriteFile(dest, []byte(output), 0644); err != nil {
					return fmt.Errorf("failed to save: %w", err)
				}
				progressf("Saved: %s", dest)
			} else {
				fmt.Print(output)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&fmt_, "format", "f", "json", "Output format: json (default), text, or tsv")
	cmd.Flags().IntVarP(&timeout, "timeout", "t", 15, "HTTP request timeout in seconds")
	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Save result to DIR as family_groups.<ext>; suppresses stdout")
	cmd.Flags().IntVar(&delayMs, "delay", 0, "Wait N milliseconds between requests (overrides config delay_ms)")
	cmd.Flags().BoolVar(&noHeader, "no-header", false, "Omit header row from TSV output")
	cmd.Flags().StringVar(&inputFile, "input-file", "", "File with patent numbers (one per line; # comments and blank lines ignored)")
	return cmd
}

// renderFamilyGroup converts an fgResult to a string in the requested format.
func renderFamilyGroup(r fgResult, format string, compact bool, noHeader bool) (string, error) {
	var sb strings.Builder
	switch format {
	case "json":
		indent := "  "
		if compact {
			indent = ""
		}
		enc := json.NewEncoder(&sb)
		enc.SetIndent("", indent)
		if err := enc.Encode(r); err != nil {
			return "", err
		}
	case "text":
		for _, g := range r.Groups {
			noun := "patents"
			if len(g.Patents) == 1 {
				noun = "patent"
			}
			fmt.Fprintf(&sb, "Group %d (%d %s):\n", g.ID, len(g.Patents), noun)
			for _, p := range g.Patents {
				fmt.Fprintf(&sb, "  %s\n", p)
			}
			sb.WriteString("\n")
		}
		if len(r.Errors) > 0 {
			fmt.Fprintf(&sb, "Errors (%d):\n", len(r.Errors))
			for _, e := range r.Errors {
				fmt.Fprintf(&sb, "  %s: %s\n", e.Patent, e.Message)
			}
			sb.WriteString("\n")
		}
		fmt.Fprintf(&sb, "Summary: %d input, %d group(s), %d error(s), %d fetch(es)\n",
			r.Summary.TotalInput, r.Summary.TotalGroups, r.Summary.TotalErrors, r.Summary.FetchCount)
	case "tsv":
		if !noHeader {
			sb.WriteString("group_id\tpatent_number\n")
		}
		for _, g := range r.Groups {
			for _, p := range g.Patents {
				fmt.Fprintf(&sb, "%d\t%s\n", g.ID, p)
			}
		}
	default:
		return "", fmt.Errorf("unknown format %q: use json, text, or tsv", format)
	}
	return sb.String(), nil
}

// normPatent normalizes a patent number for family comparison:
// uppercase, alphanumeric only (strips spaces, hyphens, dots).
func normPatent(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ── helpers ───────────────────────────────────────────────────────────────────

func prompt(scanner *bufio.Scanner, label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("%s: ", label)
	}
	if scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			return defaultVal
		}
		return line
	}
	return defaultVal
}

// isClosedErr returns true for errors that indicate a normal MCP session shutdown
// (client closed the connection or EOF on stdin).
func isClosedErr(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "EOF") ||
		strings.Contains(s, "server is closing") ||
		strings.Contains(s, "use of closed")
}

// resolvePatentNumbers returns the list of patent numbers to process.
// If inputFile is set, numbers are read from the file; otherwise args[0] is used.
func resolvePatentNumbers(args []string, inputFile string) ([]string, error) {
	if inputFile != "" {
		if len(args) > 0 {
			return nil, fmt.Errorf("cannot combine PATENT_NUMBER argument with --input-file")
		}
		nums, err := readPatentFile(inputFile)
		if err != nil {
			return nil, fmt.Errorf("reading input file: %w", err)
		}
		if len(nums) == 0 {
			return nil, fmt.Errorf("no patent numbers found in %q", inputFile)
		}
		return nums, nil
	}
	if len(args) != 1 {
		return nil, fmt.Errorf("accepts 1 arg(s), received %d; use --input-file to process multiple patents", len(args))
	}
	return []string{args[0]}, nil
}

// classifyFetchError maps a fetch error to (errType, message, exitCode).
func classifyFetchError(err error, patentNumber string) (errType, message string, code int) {
	switch err.(type) {
	case *fetcher.PatentNotFoundError:
		return "NOT_FOUND", "patent not found: " + patentNumber, exitNotFound
	case *fetcher.BotBlockedError:
		return "SERVER_ERROR", err.Error(), exitServerError
	default:
		return "NETWORK_ERROR", fmt.Sprintf("network error: %v", err), exitGeneralError
	}
}
