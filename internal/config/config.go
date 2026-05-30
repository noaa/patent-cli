package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ConfigPath returns the path to the config file.
// macOS: ~/Library/Application Support/patent-cli/config.toml
// Linux: ~/.config/patent-cli/config.toml
func ConfigPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolving user config directory: %w", err)
	}
	return filepath.Join(base, "patent-cli", "config.toml"), nil
}

type ProxyConfig struct {
	HTTPS string
	HTTP  string
}

type SSLConfig struct {
	CABundle string
}

type RequestConfig struct {
	DelayMs int
}

type Config struct {
	Proxy   ProxyConfig
	SSL     SSLConfig
	Request RequestConfig
}

type RequestOptions struct {
	Proxies  map[string]string
	CABundle string
}

// Load reads the config file; returns empty Config if file not found.
func Load() (Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	return parseToml(string(data)), nil
}

// Save writes config to the user config directory.
func Save(cfg Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	return os.WriteFile(path, []byte(dumpToml(cfg)), 0600)
}

// GetRequestOptions extracts HTTP proxy/TLS options from Config.
func GetRequestOptions(cfg Config) RequestOptions {
	opts := RequestOptions{Proxies: make(map[string]string)}
	if cfg.Proxy.HTTPS != "" {
		opts.Proxies["https"] = cfg.Proxy.HTTPS
	}
	if cfg.Proxy.HTTP != "" {
		opts.Proxies["http"] = cfg.Proxy.HTTP
	}
	opts.CABundle = cfg.SSL.CABundle
	return opts
}

// parseToml is a minimal TOML parser for the two-section config format.
func parseToml(s string) Config {
	var cfg Config
	scanner := bufio.NewScanner(strings.NewReader(s))
	var section string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(line[1 : len(line)-1])
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, `"`)
		switch section {
		case "proxy":
			switch key {
			case "https":
				cfg.Proxy.HTTPS = val
			case "http":
				cfg.Proxy.HTTP = val
			}
		case "ssl":
			if key == "ca_bundle" {
				cfg.SSL.CABundle = val
			}
		case "request":
			if key == "delay_ms" {
				if n, err := strconv.Atoi(val); err == nil && n >= 0 {
					cfg.Request.DelayMs = n
				}
			}
		}
	}
	return cfg
}

// dumpToml serializes Config to TOML text.
func dumpToml(cfg Config) string {
	var sb strings.Builder
	if cfg.Proxy.HTTPS != "" || cfg.Proxy.HTTP != "" {
		fmt.Fprintln(&sb, "[proxy]")
		if cfg.Proxy.HTTPS != "" {
			fmt.Fprintf(&sb, "https = %q\n", cfg.Proxy.HTTPS)
		}
		if cfg.Proxy.HTTP != "" {
			fmt.Fprintf(&sb, "http = %q\n", cfg.Proxy.HTTP)
		}
		fmt.Fprintln(&sb)
	}
	if cfg.SSL.CABundle != "" {
		fmt.Fprintln(&sb, "[ssl]")
		fmt.Fprintf(&sb, "ca_bundle = %q\n", cfg.SSL.CABundle)
		fmt.Fprintln(&sb)
	}
	if cfg.Request.DelayMs > 0 {
		fmt.Fprintln(&sb, "[request]")
		fmt.Fprintf(&sb, "delay_ms = %d\n", cfg.Request.DelayMs)
		fmt.Fprintln(&sb)
	}
	return sb.String()
}
