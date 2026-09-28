package registry

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DefaultRegistryURL = "https://registry.okf-memory.dev"

type BundleManifest struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	Hash        string `json:"hash"`
	DownloadURL string `json:"download_url"`
}

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = DefaultRegistryURL
	}
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Resolve(slugOrURL string) (*BundleManifest, error) {
	if strings.HasPrefix(slugOrURL, "github.com/") || strings.HasPrefix(slugOrURL, "https://") {
		return c.resolveGitURL(slugOrURL)
	}

	cleanSlug := strings.Trim(strings.TrimSpace(slugOrURL), "/")
	endpoint := fmt.Sprintf("%s/bundles/%s.json", c.BaseURL, cleanSlug)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid registry request: %w", err)
	}
	req.Header.Set("User-Agent", "okf-agent-memory-cli")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("registry connection error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("bundle %q not found in registry %s", cleanSlug, c.BaseURL)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %s", resp.Status)
	}

	var m BundleManifest
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, fmt.Errorf("malformed registry manifest: %w", err)
	}

	if strings.HasPrefix(m.DownloadURL, "/") {
		m.DownloadURL = c.BaseURL + m.DownloadURL
	}

	return &m, nil
}

func (c *Client) resolveGitURL(rawURL string) (*BundleManifest, error) {
	parsedURL := rawURL
	if !strings.HasPrefix(parsedURL, "http://") && !strings.HasPrefix(parsedURL, "https://") {
		parsedURL = "https://" + parsedURL
	}
	u, err := url.Parse(parsedURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Git URL: %w", err)
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("git URL must follow github.com/owner/repo format")
	}
	owner, repo := parts[0], parts[1]
	id := fmt.Sprintf("%s/%s", owner, repo)

	archiveURL := fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/main.tar.gz", owner, repo)
	return &BundleManifest{
		ID:          id,
		Version:     "main",
		Hash:        "",
		DownloadURL: archiveURL,
	}, nil
}

func (c *Client) DownloadAndExtract(manifest *BundleManifest, targetDir string) error {
	req, err := http.NewRequest("GET", manifest.DownloadURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "okf-agent-memory-cli")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download bundle: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bundle download failed with HTTP %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading bundle stream: %w", err)
	}

	if manifest.Hash != "" {
		calculated := fmt.Sprintf("sha256:%x", sha256.Sum256(data))
		if !strings.EqualFold(calculated, manifest.Hash) {
			return fmt.Errorf("checksum mismatch: expected %s, got %s", manifest.Hash, calculated)
		}
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("cannot create vendor dir: %w", err)
	}

	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to decompress gzip: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar extract error: %w", err)
		}

		cleanName := filepath.Clean(hdr.Name)
		if strings.HasPrefix(cleanName, "..") || filepath.IsAbs(cleanName) {
			continue
		}

		destPath := filepath.Join(targetDir, cleanName)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destPath, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
				return err
			}
			outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, hdr.FileInfo().Mode())
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}

	return nil
}
