package registry_test

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
	"strings"
	"testing"

	"github.com/okf-memory/okf-agent-memory/pkg/registry"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func makeTestTarGz(t *testing.T, files map[string]string) ([]byte, string) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	tw.Close()
	gw.Close()

	data := buf.Bytes()
	hash := fmt.Sprintf("sha256:%x", sha256.Sum256(data))
	return data, hash
}

func TestClient_ResolveScopedAndTopLevel(t *testing.T) {
	tarData, hash := makeTestTarGz(t, map[string]string{
		"index.md":          "# Django Bundle",
		"decisions/auth.md": "---\ntype: Decision\ntitle: Auth\n---\nUse Django Session",
	})

	baseURL := "https://registry.okf-memory.dev"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/bundles/peter/django-5-rules.json":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"id":"peter/django-5-rules","version":"1.0.0","hash":%q,"download_url":"/downloads/django.tar.gz"}`, hash)
		case "/bundles/nextjs-15.json":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"id":"nextjs-15","version":"1.0.0","hash":%q,"download_url":"/downloads/nextjs.tar.gz"}`, hash)
		case "/downloads/django.tar.gz":
			w.Header().Set("Content-Type", "application/gzip")
			w.Write(tarData)
		default:
			http.NotFound(w, r)
		}
	})

	client := registry.NewClient(baseURL)
	client.HTTPClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Result(), nil
	})

	// Test Scoped ID (peter/django-5-rules)
	manifest, err := client.Resolve("peter/django-5-rules")
	if err != nil {
		t.Fatalf("Resolve scoped failed: %v", err)
	}
	if manifest.ID != "peter/django-5-rules" || manifest.Hash != hash {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}

	targetDir := filepath.Join(t.TempDir(), "vendor", "peter", "django-5-rules")
	if err := client.DownloadAndExtract(manifest, targetDir); err != nil {
		t.Fatalf("DownloadAndExtract failed: %v", err)
	}

	// Verify extracted files
	idxContent, err := os.ReadFile(filepath.Join(targetDir, "index.md"))
	if err != nil || !bytes.Contains(idxContent, []byte("# Django Bundle")) {
		t.Errorf("failed to verify index.md: %v", err)
	}

	// Test Top-level ID (nextjs-15)
	manifestTop, err := client.Resolve("nextjs-15")
	if err != nil {
		t.Fatalf("Resolve top-level failed: %v", err)
	}
	if manifestTop.ID != "nextjs-15" {
		t.Fatalf("unexpected manifest ID: %s", manifestTop.ID)
	}

	// Test Git URL resolution with default branch
	gitManifest, err := client.Resolve("github.com/acme/my-bundle")
	if err != nil {
		t.Fatalf("Resolve git URL failed: %v", err)
	}
	if gitManifest.ID != "acme/my-bundle" || gitManifest.Version != "main" {
		t.Errorf("expected git ID acme/my-bundle with main, got %s / %s", gitManifest.ID, gitManifest.Version)
	}

	// Test Git URL resolution with immutable @tag
	gitTagManifest, err := client.Resolve("github.com/acme/my-bundle@v2.0.0")
	if err != nil {
		t.Fatalf("Resolve git URL with tag failed: %v", err)
	}
	if gitTagManifest.Version != "v2.0.0" || !strings.Contains(gitTagManifest.DownloadURL, "refs/tags/v2.0.0.tar.gz") {
		t.Errorf("expected git tag v2.0.0, got %+v", gitTagManifest)
	}

	// Test Registry bundle with @version
	manifestVersioned, err := client.Resolve("nextjs-15@1.0.0")
	if err != nil {
		t.Fatalf("Resolve versioned failed: %v", err)
	}
	if manifestVersioned.ID != "nextjs-15" || manifestVersioned.Version != "1.0.0" {
		t.Errorf("unexpected versioned manifest: %+v", manifestVersioned)
	}
}

