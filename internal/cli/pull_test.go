package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/okf-memory/okf-agent-memory/pkg/registry"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCmdPull_ScopedSuccess(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	content := "# Django Rules"
	tw.WriteHeader(&tar.Header{Name: "index.md", Mode: 0644, Size: int64(len(content))})
	tw.Write([]byte(content))
	tw.Close()
	gw.Close()
	data := buf.Bytes()
	hash := fmt.Sprintf("sha256:%x", sha256.Sum256(data))

	mockURL := "https://mock.registry.okf-memory.dev"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bundles/peter/django-5-rules.json" {
			fmt.Fprintf(w, `{"id":"peter/django-5-rules","version":"1.0.0","hash":%q,"download_url":"/bundle.tar.gz"}`, hash)
		} else if r.URL.Path == "/bundle.tar.gz" {
			w.Write(data)
		} else {
			http.NotFound(w, r)
		}
	})

	origFactory := newRegistryClient
	defer func() { newRegistryClient = origFactory }()
	newRegistryClient = func(baseURL string) *registry.Client {
		c := registry.NewClient(baseURL)
		c.HTTPClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			return rec.Result(), nil
		})
		return c
	}

	workDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	cmd, ok := FindCommand("pull")
	if !ok {
		t.Fatalf("pull command not found in registry")
	}

	cmd.Run([]string{"--registry", mockURL, "peter/django-5-rules"})

	targetFile := filepath.Join(workDir, ".okf", "vendor", "peter", "django-5-rules", "index.md")
	if _, err := os.Stat(targetFile); err != nil {
		t.Fatalf("expected pulled bundle file at %s: %v", targetFile, err)
	}

	lockFile := filepath.Join(workDir, "okf.lock")
	if _, err := os.Stat(lockFile); err != nil {
		t.Fatalf("expected okf.lock at %s", lockFile)
	}

	// Test vendor list
	vendorCmd, ok := FindCommand("vendor")
	if !ok {
		t.Fatalf("vendor command not found in registry")
	}
	vendorCmd.Run([]string{"list"})

	// Test vendor remove
	vendorCmd.Run([]string{"remove", "peter/django-5-rules"})
	if _, err := os.Stat(targetFile); !os.IsNotExist(err) {
		t.Errorf("expected vendor file to be deleted: %v", err)
	}
}

