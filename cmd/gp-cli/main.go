package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/area99/patent-cli/internal/config"
	"github.com/area99/patent-cli/internal/fetcher"
	"github.com/area99/patent-cli/internal/formatter"
	"github.com/area99/patent-cli/internal/parser"
	"github.com/area99/patent-cli/internal/updater"
	"github.com/spf13/cobra"
)

const version = "0.1.0"

var verbose bool

func main() {
	root := &cobra.Command{
		Use:   "gp-cli",
		Short: "Google Patents CLI — fetch patent metadata by patent number",
		Long: `Google Patents CLI — fetch patent metadata by patent number.

Commands:
  lookup      Fetch metadata for a patent number
  download    Download the patent PDF
  images      Download high-resolution figure images
  fields      List all available output fields
  configure   Set proxy / CA-cert options

Quick start:
  gp-cli lookup US12514139B2
  gp-cli lookup US20250350789 --format text
  gp-cli lookup US20250350789 --fields title,assignee
  gp-cli download US9735861 --output-dir ./pdfs
  gp-cli images US11125686B2 --output-dir ./figs
  gp-cli fields`,
		Version:          version,
		SilenceUsage:     true,
		SilenceErrors:    true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {},
	}

	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Print progress and debug logs to stderr")

	root.AddCommand(
		lookupCmd(),
		downloadCmd(),
		imagesCmd(),
		fieldsCmd(),
		configureCmd(),
		updateCmd(),
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

func loadRequestOpts(timeout time.Duration) fetcher.Options {
	cfg, _ := config.Load()
	reqOpts := config.GetRequestOptions(cfg)
	if cfg.Request.DelayMs > 0 {
		time.Sleep(time.Duration(cfg.Request.DelayMs) * time.Millisecond)
	}
	return fetcher.Options{
		Timeout:  timeout,
		Proxies:  reqOpts.Proxies,
		CABundle: reqOpts.CABundle,
	}
}

// ── lookup ────────────────────────────────────────────────────────────────────

func lookupCmd() *cobra.Command {
	var (
		fmt_        string
		singleField string
		multiFields []string
		timeout     int
		outputDir   string
	)

	cmd := &cobra.Command{
		Use:   "lookup PATENT_NUMBER",
		Short: "Fetch Google Patents metadata for PATENT_NUMBER",
		Long: `Fetch Google Patents metadata for PATENT_NUMBER.

Examples:
  gp-cli lookup US20250350789
  gp-cli lookup US12514139B2 --format text
  gp-cli lookup US20250350789 --field title
  gp-cli lookup US20250350789 --fields title,assignee,filing_date
  gp-cli lookup US20250350789 --format tsv
  gp-cli lookup US12514139B2 --output-dir ./output
  gp-cli lookup US12514139B2 --format text --output-dir ./output`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			patentNumber := args[0]
			logf("lookup: %s", patentNumber)

			opts := loadRequestOpts(time.Duration(timeout) * time.Second)
			html, err := fetcher.FetchHTML(patentNumber, opts)
			if err != nil {
				switch err.(type) {
				case *fetcher.PatentNotFoundError:
					formatter.PrintError("Patent not found", patentNumber)
				default:
					formatter.PrintError(fmt.Sprintf("Network error: %v", err), patentNumber)
				}
				os.Exit(1)
			}

			data := parser.ParseAll(html)
			dm := formatter.ToDataMap(data)

			// --field: single plain value
			if singleField != "" {
				v, ok := dm.Get(singleField)
				if !ok {
					fmt.Fprintf(os.Stderr, "Unknown field: %q. Run 'gp-cli fields' for available fields.\n", singleField)
					os.Exit(1)
				}
				formatter.PrintField(v)
				return nil
			}

			// --fields: filter to selected fields
			var fieldsList []string
			for _, token := range multiFields {
				for _, f := range strings.Split(token, ",") {
					f = strings.TrimSpace(f)
					if f != "" {
						fieldsList = append(fieldsList, f)
					}
				}
			}
			if len(fieldsList) > 0 {
				for _, f := range fieldsList {
					if _, ok := dm.Get(f); !ok {
						fmt.Fprintf(os.Stderr, "Unknown field: %q. Run 'gp-cli fields' for available fields.\n", f)
						os.Exit(1)
					}
				}
			}

			out := formatter.SelectFields(dm, fieldsList)
			fmt.Println(formatter.Render(out, fmt_))

			if outputDir != "" {
				saved, err := formatter.SaveToDir(out, fmt_, outputDir, patentNumber)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Save error: %v\n", err)
					os.Exit(1)
				}
				fmt.Fprintf(os.Stderr, "Saved: %s\n", saved)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&fmt_, "format", "f", "json", "Output format: json (default), text, or tsv")
	cmd.Flags().StringVar(&singleField, "field", "", "Print a single field value as plain text (overrides --format)")
	cmd.Flags().StringArrayVar(&multiFields, "fields", nil, "Comma-separated field list. Repeatable: --fields title --fields abstract")
	cmd.Flags().IntVarP(&timeout, "timeout", "t", 15, "HTTP request timeout in seconds")
	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Save result to DIR; filename is derived from the patent number")
	return cmd
}

// ── download ──────────────────────────────────────────────────────────────────

func downloadCmd() *cobra.Command {
	var (
		outputDir string
		timeout   int
	)

	cmd := &cobra.Command{
		Use:   "download PATENT_NUMBER",
		Short: "Download the patent PDF for PATENT_NUMBER (saved as <number>.pdf)",
		Long: `Download the patent PDF for PATENT_NUMBER (saved as <number>.pdf).

Examples:
  gp-cli download US9735861
  gp-cli download US9735861 --output-dir ./pdfs`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			patentNumber := args[0]
			opts := loadRequestOpts(15 * time.Second)

			html, err := fetcher.FetchHTML(patentNumber, opts)
			if err != nil {
				switch err.(type) {
				case *fetcher.PatentNotFoundError:
					formatter.PrintError("Patent not found", patentNumber)
				default:
					formatter.PrintError(fmt.Sprintf("Network error: %v", err), patentNumber)
				}
				os.Exit(1)
			}

			data := parser.ParseAll(html)
			pdfURL := data.PDFURL
			if pdfURL == "" {
				fmt.Fprintln(os.Stderr, "Error: PDF link not found for this patent.")
				os.Exit(1)
			}

			pubNumber := data.PublicationNumber
			if pubNumber == "" {
				pubNumber = strings.ToUpper(strings.ReplaceAll(patentNumber, "-", ""))
			}
			filename := pubNumber + ".pdf"

			if err := os.MkdirAll(outputDir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
				os.Exit(1)
			}
			dest := outputDir + "/" + filename

			logf("PDF URL: %s", pdfURL)
			fmt.Fprintf(os.Stderr, "Downloading: %s\n", pdfURL)

			dlOpts := loadRequestOpts(time.Duration(timeout) * time.Second)
			if err := fetcher.FetchBinary(pdfURL, dest, dlOpts); err != nil {
				formatter.PrintError(fmt.Sprintf("PDF download failed: %v", err), patentNumber)
				os.Exit(1)
			}

			fmt.Println("Saved:", dest)
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", ".", "Directory to save the PDF")
	cmd.Flags().IntVarP(&timeout, "timeout", "t", 60, "HTTP request timeout in seconds")
	return cmd
}

// ── images ────────────────────────────────────────────────────────────────────

func imagesCmd() *cobra.Command {
	var (
		outputDir string
		timeout   int
	)

	cmd := &cobra.Command{
		Use:   "images PATENT_NUMBER",
		Short: "Download high-resolution figure images for PATENT_NUMBER",
		Long: `Download high-resolution figure images for PATENT_NUMBER.
Files are saved as fig01.png, fig02.png, ...

Examples:
  gp-cli images US11125686B2
  gp-cli images KR102355140B1 --output-dir ./figs`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			patentNumber := args[0]
			opts := loadRequestOpts(15 * time.Second)

			html, err := fetcher.FetchHTML(patentNumber, opts)
			if err != nil {
				switch err.(type) {
				case *fetcher.PatentNotFoundError:
					formatter.PrintError("Patent not found", patentNumber)
				default:
					formatter.PrintError(fmt.Sprintf("Network error: %v", err), patentNumber)
				}
				os.Exit(1)
			}

			data := parser.ParseAll(html)
			pubNumber := data.PublicationNumber
			if pubNumber == "" {
				pubNumber = strings.ToUpper(strings.ReplaceAll(patentNumber, "-", ""))
			}

			urls := parser.ParseImageURLs(html)
			if len(urls) == 0 {
				fmt.Fprintln(os.Stderr, "No figure images found.")
				os.Exit(1)
			}

			if err := os.MkdirAll(outputDir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
				os.Exit(1)
			}

			fmt.Fprintf(os.Stderr, "Found %d figure image(s) for %s.\n", len(urls), pubNumber)

			dlOpts := loadRequestOpts(time.Duration(timeout) * time.Second)
			for i, imgURL := range urls {
				filename := fmt.Sprintf("fig%02d.png", i+1)
				dest := outputDir + "/" + filename
				logf("Image URL: %s", imgURL)
				fmt.Fprintf(os.Stderr, "  [%d/%d] %s\n", i+1, len(urls), filename)

				if err := fetcher.FetchBinary(imgURL, dest, dlOpts); err != nil {
					fmt.Fprintf(os.Stderr, "  Warning: failed to download image %d: %v\n", i+1, err)
					continue
				}
				fmt.Println("Saved:", dest)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", ".", "Directory to save images")
	cmd.Flags().IntVarP(&timeout, "timeout", "t", 30, "HTTP request timeout in seconds per image")
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

Config file: ~/.patent-cli.toml
Press Enter with no value to skip a field.
Existing values are shown as defaults.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			existing, _ := config.Load()
			scanner := bufio.NewScanner(os.Stdin)

			fmt.Println("Google Patent CLI — Configuration")
			fmt.Println(strings.Repeat("─", 40))
			fmt.Println("Config file:", config.ConfigPath())
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
			fmt.Println("\nConfig saved:", config.ConfigPath())
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

// ── configure ─────────────────────────────────────────────────────────────────

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
