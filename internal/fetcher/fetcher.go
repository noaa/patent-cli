package fetcher

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

const baseURL = "https://patents.google.com/patent"

const retryWait429 = 5 * time.Second
const maxRetries = 3

var pub6DigitRE = regexp.MustCompile(`^(US)(\d{4})(\d{6})([A-Z]\d*)?$`)
var nonAlphanumRE = regexp.MustCompile(`[^A-Z0-9]`)

// PatentNotFoundError is returned when the patent page returns 404.
type PatentNotFoundError struct {
	PatentNumber string
}

func (e *PatentNotFoundError) Error() string {
	return fmt.Sprintf("patent not found: %s", e.PatentNumber)
}

// BotBlockedError is returned when Google detects automated access.
type BotBlockedError struct{}

func (e *BotBlockedError) Error() string {
	return "request blocked by Google (bot detection / rate limiting); try again later or increase delay_ms in config"
}

// FetchError wraps network or HTTP errors.
type FetchError struct {
	Message string
}

func (e *FetchError) Error() string {
	return e.Message
}

// Options controls HTTP behavior.
type Options struct {
	Timeout  time.Duration
	Proxies  map[string]string
	CABundle string
	Language string // if set, appends /<language> to the patent URL (e.g. "en" for machine translation)
}

// NormalizeForURL converts a patent number to the Google Patents URL format.
// US publication numbers with a 6-digit sequence are zero-padded to 7 digits.
func NormalizeForURL(patentNumber string) string {
	clean := nonAlphanumRE.ReplaceAllString(strings.ToUpper(patentNumber), "")
	m := pub6DigitRE.FindStringSubmatch(clean)
	if m != nil {
		country, year, seq6, kind := m[1], m[2], m[3], m[4]
		for len(seq6) < 7 {
			seq6 = "0" + seq6
		}
		return country + year + seq6 + kind
	}
	return clean
}

func newClient(opts Options) (*http.Client, error) {
	transport := &http.Transport{}

	if opts.CABundle != "" {
		caCert, err := os.ReadFile(opts.CABundle)
		if err != nil {
			return nil, &FetchError{Message: fmt.Sprintf("failed to read CA bundle: %v", err)}
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(caCert)
		transport.TLSClientConfig = &tls.Config{RootCAs: pool}
	}

	if len(opts.Proxies) > 0 {
		transport.Proxy = func(req *http.Request) (*url.URL, error) {
			if p, ok := opts.Proxies[req.URL.Scheme]; ok {
				return url.Parse(p)
			}
			return http.ProxyFromEnvironment(req)
		}
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 15 * time.Second
	}

	return &http.Client{Transport: transport, Timeout: timeout}, nil
}

func get(targetURL string, opts Options) (*http.Response, error) {
	client, err := newClient(opts)
	if err != nil {
		return nil, err
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("GET", targetURL, nil)
		if err != nil {
			return nil, &FetchError{Message: err.Error()}
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; patent-cli/1.0)")

		resp, err := client.Do(req)
		if err != nil {
			return nil, &FetchError{Message: err.Error()}
		}

		if resp.StatusCode == 429 && attempt < maxRetries {
			resp.Body.Close()
			time.Sleep(retryWait429)
			continue
		}
		return resp, nil
	}
	return nil, &FetchError{Message: "max retries exceeded"}
}

// FetchHTML fetches the Google Patents page and returns its HTML.
func FetchHTML(patentNumber string, opts Options) (string, error) {
	normalized := NormalizeForURL(patentNumber)
	targetURL := fmt.Sprintf("%s/%s", baseURL, normalized)
	if opts.Language != "" {
		targetURL = fmt.Sprintf("%s/%s/%s", baseURL, normalized, opts.Language)
	}

	resp, err := get(targetURL, opts)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return "", &PatentNotFoundError{PatentNumber: patentNumber}
	}
	if resp.StatusCode != 200 {
		return "", &FetchError{Message: fmt.Sprintf("HTTP %d for %s", resp.StatusCode, targetURL)}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", &FetchError{Message: err.Error()}
	}
	html := string(body)
	if isBotBlocked(html) {
		return "", &BotBlockedError{}
	}
	return html, nil
}

// isBotBlocked reports whether the HTML is Google's bot-detection / rate-limit page.
// Google returns HTTP 200 for this page, so status code alone is insufficient.
func isBotBlocked(html string) bool {
	// <title>Sorry...</title> is the canonical marker on the block page.
	// Also catch the fallback text Google embeds in the body.
	return strings.Contains(html, "<title>Sorry...</title>") ||
		strings.Contains(html, "Our systems have detected unusual traffic")
}

// FetchBinary downloads a URL to destPath (used for PDFs and images).
func FetchBinary(srcURL, destPath string, opts Options) error {
	resp, err := get(srcURL, opts)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return &FetchError{Message: fmt.Sprintf("HTTP %d for %s", resp.StatusCode, srcURL)}
	}

	f, err := os.Create(destPath)
	if err != nil {
		return &FetchError{Message: err.Error()}
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return &FetchError{Message: err.Error()}
	}
	return nil
}
