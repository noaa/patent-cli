package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const repo = "noaa/patent-cli"

type release struct {
	TagName string  `json:"tag_name"`
	Assets  []asset `json:"assets"`
}

type asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// LatestTag returns the tag name of the latest GitHub release.
func LatestTag() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %s", resp.Status)
	}

	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", fmt.Errorf("failed to parse release JSON: %w", err)
	}
	if rel.TagName == "" {
		return "", fmt.Errorf("no releases found at github.com/%s", repo)
	}
	return rel.TagName, nil
}

// HasUpdate returns true when the latest release tag differs from current.
// current should be the bare version string (e.g. "0.1.0"); tag may have a
// leading "v" (e.g. "v0.2.0") — both forms are normalised before comparison.
func HasUpdate(current, latestTag string) bool {
	norm := func(s string) string { return strings.TrimPrefix(s, "v") }
	return norm(current) != norm(latestTag)
}

// Do downloads the latest release binary and replaces the running executable.
func Do(latestTag string) error {
	assetName := assetFilename()
	url, err := downloadURL(latestTag, assetName)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Downloading %s ...\n", url)

	tmp, err := os.CreateTemp("", "gp-cli-update-*")
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w", err)
	}
	defer os.Remove(tmp.Name())

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %s", resp.Status)
	}

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}
	tmp.Close()

	newBin, err := extractBinary(tmp.Name(), assetName)
	if err != nil {
		return err
	}
	defer os.Remove(newBin)

	return replaceExecutable(newBin)
}

// assetFilename returns the release asset name for the current OS/arch.
func assetFilename() string {
	switch runtime.GOOS {
	case "windows":
		return "gp-cli-windows-amd64.exe.zip"
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "gp-cli-darwin-arm64.tar.gz"
		}
		return "gp-cli-darwin-amd64.tar.gz"
	default:
		return "gp-cli-linux-amd64.tar.gz"
	}
}

// downloadURL fetches the release metadata and returns the browser download URL
// for the requested asset.
func downloadURL(tag, assetName string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", repo, tag)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()

	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", fmt.Errorf("failed to parse release JSON: %w", err)
	}

	for _, a := range rel.Assets {
		if a.Name == assetName {
			return a.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("asset %q not found in release %s", assetName, tag)
}

// extractBinary unpacks the archive and writes the binary to a temp file,
// returning its path. The caller is responsible for removing it.
func extractBinary(archivePath, assetName string) (string, error) {
	if strings.HasSuffix(assetName, ".zip") {
		return extractZip(archivePath)
	}
	return extractTarGz(archivePath)
}

func extractTarGz(archivePath string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("not a valid gzip archive: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		// pick the first regular file (the binary)
		tmp, err := os.CreateTemp("", "gp-cli-bin-*")
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(tmp, tr); err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return "", err
		}
		tmp.Close()
		if err := os.Chmod(tmp.Name(), 0755); err != nil {
			return "", err
		}
		return tmp.Name(), nil
	}
	return "", fmt.Errorf("no binary found in archive")
}

func extractZip(archivePath string) (string, error) {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("not a valid zip archive: %w", err)
	}
	defer zr.Close()

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		tmp, err := os.CreateTemp("", "gp-cli-bin-*.exe")
		if err != nil {
			rc.Close()
			return "", err
		}
		if _, err := io.Copy(tmp, rc); err != nil {
			rc.Close()
			tmp.Close()
			os.Remove(tmp.Name())
			return "", err
		}
		rc.Close()
		tmp.Close()
		return tmp.Name(), nil
	}
	return "", fmt.Errorf("no binary found in archive")
}

// replaceExecutable atomically replaces the running binary with newBin.
func replaceExecutable(newBin string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine current executable path: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return err
	}

	// Write the new binary next to the current one, then rename.
	dir := filepath.Dir(exe)
	staged := filepath.Join(dir, ".gp-cli-update-new")

	src, err := os.Open(newBin)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(staged, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("cannot write update (check permissions for %s): %w", dir, err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		os.Remove(staged)
		return err
	}
	dst.Close()

	// On Windows, rename the running exe out of the way first.
	if runtime.GOOS == "windows" {
		old := exe + ".old"
		os.Remove(old)
		if err := os.Rename(exe, old); err != nil {
			os.Remove(staged)
			return fmt.Errorf("cannot rename current binary: %w", err)
		}
	}

	if err := os.Rename(staged, exe); err != nil {
		os.Remove(staged)
		return fmt.Errorf("cannot replace binary: %w", err)
	}

	// Best-effort cleanup of the .old file on Windows.
	if runtime.GOOS == "windows" {
		os.Remove(exe + ".old")
	}

	return nil
}
