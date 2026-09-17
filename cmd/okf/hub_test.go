package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/okf-memory/okf-agent-memory/pkg/sync"
	"github.com/okf-memory/okf-agent-memory/pkg/vault"
)

type localRoundTripper struct {
	handler http.Handler
}

func (l *localRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	l.handler.ServeHTTP(rec, req)
	return rec.Result(), nil
}

func newTestHubServer() *sync.Server {
	return sync.NewServer("")
}

func newTestClientFromHandler(handler http.Handler, token string) *sync.Client {
	c := sync.NewClient("http://hub.local", token)
	c.HTTPClient.Transport = &localRoundTripper{handler: handler}
	return c
}

func TestCmdHub_InitVault(t *testing.T) {
	var buf bytes.Buffer
	dir := t.TempDir()

	err := runHubInitVault(&buf, dir)
	if err != nil {
		t.Fatalf("runHubInitVault error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "EMERGENCY KIT") {
		t.Fatalf("output missing EMERGENCY KIT header:\n%s", out)
	}
	if !strings.Contains(out, "Vault ID:") {
		t.Fatalf("output missing Vault ID:\n%s", out)
	}
	if !strings.Contains(out, "Secret Key:") {
		t.Fatalf("output missing Secret Key:\n%s", out)
	}

	// Verify local config file was created
	cfgPath := filepath.Join(dir, ".okf-vault.json")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("expected config file %s to exist", cfgPath)
	}
}

func TestCmdHub_PushPullSyncWithServer(t *testing.T) {
	// Initialize a vault in dirA
	dirA := t.TempDir()
	_ = os.WriteFile(filepath.Join(dirA, "index.md"), []byte("# Knowledge Index\nokf_version: 0.2\n"), 0o644)

	var initBuf bytes.Buffer
	if err := runHubInitVault(&initBuf, dirA); err != nil {
		t.Fatalf("init error: %v", err)
	}

	// Read generated config
	cfgA, err := loadVaultConfig(dirA)
	if err != nil {
		t.Fatalf("load config error: %v", err)
	}

	secretKey, err := vault.GenerateSecretKey()
	if err != nil {
		t.Fatalf("GenerateSecretKey error: %v", err)
	}
	password := "master-pass-123"

	// Create test in-memory server handler
	server := newTestHubServer()
	clientA := newTestClientFromHandler(server.Handler(), "token-a")

	// 1. Push from A
	var pushBuf bytes.Buffer
	err = executeHubPush(&pushBuf, dirA, clientA, cfgA.VaultID, password, secretKey, "Initial push")
	if err != nil {
		t.Fatalf("executeHubPush error: %v", err)
	}
	if !strings.Contains(pushBuf.String(), "Push completed") {
		t.Fatalf("expected push completion message, got:\n%s", pushBuf.String())
	}

	// 2. Pull into dirB
	dirB := t.TempDir()
	clientB := newTestClientFromHandler(server.Handler(), "token-b")
	var pullBuf bytes.Buffer
	err = executeHubPull(&pullBuf, dirB, clientB, cfgA.VaultID, password, secretKey)
	if err != nil {
		t.Fatalf("executeHubPull error: %v", err)
	}

	pulledIndex, err := os.ReadFile(filepath.Join(dirB, "index.md"))
	if err != nil {
		t.Fatalf("failed to read pulled index.md: %v", err)
	}
	if string(pulledIndex) != "# Knowledge Index\nokf_version: 0.2\n" {
		t.Fatalf("unexpected content in pulled index.md: %s", string(pulledIndex))
	}
}
