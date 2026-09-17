package sync_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/okf-memory/okf-agent-memory/pkg/sync"
	"github.com/okf-memory/okf-agent-memory/pkg/vault"
)

func TestEngine_PushAndPullRoundtrip(t *testing.T) {
	server := sync.NewServer("")
	ctx := context.Background()
	vaultID := "test-vault-roundtrip"
	vaultKey := make([]byte, 32)
	for i := range vaultKey {
		vaultKey[i] = byte(i + 1)
	}

	// 1. Setup local Bundle A
	dirA := t.TempDir()
	file1Path := filepath.Join(dirA, "index.md")
	file2Path := filepath.Join(dirA, "architecture", "sync.md")
	_ = os.MkdirAll(filepath.Dir(file2Path), 0o755)

	_ = os.WriteFile(file1Path, []byte("# Root Index\nokf_version: 0.2\n"), 0o644)
	_ = os.WriteFile(file2Path, []byte("# Sync Architecture\nZero-knowledge details\n"), 0o644)

	clientA := newTestClient(server.Handler(), "token-a")
	engineA := sync.NewEngine(clientA, vaultID, vaultKey, dirA)

	authorA := vault.CommitAuthor{ClientID: "client-a", Agent: "agent-a"}
	pushResA, err := engineA.Push(ctx, authorA, "Initial commit from A")
	if err != nil {
		t.Fatalf("EngineA Push error: %v", err)
	}

	if pushResA.CommitHash == "" {
		t.Fatalf("expected non-empty commit hash")
	}

	// Verify server head is now at pushResA.CommitHash
	head, err := clientA.GetHead(ctx, vaultID)
	if err != nil {
		t.Fatalf("GetHead error: %v", err)
	}
	if head.HeadCommit != pushResA.CommitHash {
		t.Fatalf("expected remote head %s, got %s", pushResA.CommitHash, head.HeadCommit)
	}

	// 2. Setup Client B (empty folder)
	dirB := t.TempDir()
	clientB := newTestClient(server.Handler(), "token-b")
	engineB := sync.NewEngine(clientB, vaultID, vaultKey, dirB)

	pullResB, err := engineB.Pull(ctx)
	if err != nil {
		t.Fatalf("EngineB Pull error: %v", err)
	}

	if pullResB.CommitHash != pushResA.CommitHash {
		t.Fatalf("EngineB expected commit %s, got %s", pushResA.CommitHash, pullResB.CommitHash)
	}

	// Verify files exist in Dir B with identical plaintext content
	bFile1, err := os.ReadFile(filepath.Join(dirB, "index.md"))
	if err != nil {
		t.Fatalf("failed to read pulled index.md in dirB: %v", err)
	}
	if string(bFile1) != "# Root Index\nokf_version: 0.2\n" {
		t.Fatalf("unexpected content in bFile1: %s", string(bFile1))
	}

	bFile2, err := os.ReadFile(filepath.Join(dirB, "architecture", "sync.md"))
	if err != nil {
		t.Fatalf("failed to read pulled sync.md in dirB: %v", err)
	}
	if string(bFile2) != "# Sync Architecture\nZero-knowledge details\n" {
		t.Fatalf("unexpected content in bFile2: %s", string(bFile2))
	}
}

func TestEngine_Sync_DisjointMerge(t *testing.T) {
	server := sync.NewServer("")
	ctx := context.Background()
	vaultID := "test-vault-disjoint"
	vaultKey := make([]byte, 32)
	vaultKey[0] = 42

	// Setup Base
	dirA := t.TempDir()
	_ = os.WriteFile(filepath.Join(dirA, "base.md"), []byte("Base content"), 0o644)

	clientA := newTestClient(server.Handler(), "token-a")
	engineA := sync.NewEngine(clientA, vaultID, vaultKey, dirA)
	authorA := vault.CommitAuthor{ClientID: "client-a", Agent: "agent"}

	_, err := engineA.Push(ctx, authorA, "Initial commit")
	if err != nil {
		t.Fatalf("initial push error: %v", err)
	}

	// Client B pulls base
	dirB := t.TempDir()
	clientB := newTestClient(server.Handler(), "token-b")
	engineB := sync.NewEngine(clientB, vaultID, vaultKey, dirB)
	authorB := vault.CommitAuthor{ClientID: "client-b", Agent: "agent"}

	_, err = engineB.Pull(ctx)
	if err != nil {
		t.Fatalf("engineB pull error: %v", err)
	}

	// Client B adds note_b.md and pushes to Hub
	_ = os.WriteFile(filepath.Join(dirB, "note_b.md"), []byte("Note B content"), 0o644)
	_, err = engineB.Push(ctx, authorB, "Add note B")
	if err != nil {
		t.Fatalf("engineB push error: %v", err)
	}

	// Meanwhile, Client A adds note_a.md locally (without having pulled note_b.md)
	_ = os.WriteFile(filepath.Join(dirA, "note_a.md"), []byte("Note A content"), 0o644)

	// Client A runs Sync!
	syncResA, err := engineA.Sync(ctx, authorA, "Add note A and sync")
	if err != nil {
		t.Fatalf("engineA Sync error: %v", err)
	}

	if len(syncResA.Conflicts) != 0 {
		t.Fatalf("expected 0 conflicts in disjoint sync, got: %v", syncResA.Conflicts)
	}

	// Verify Client A now has both note_a.md and note_b.md
	if _, err := os.Stat(filepath.Join(dirA, "note_a.md")); err != nil {
		t.Fatalf("note_a.md missing in dirA")
	}
	if _, err := os.Stat(filepath.Join(dirA, "note_b.md")); err != nil {
		t.Fatalf("note_b.md should have been pulled to dirA during sync")
	}
}

func TestEngine_Sync_CollisionForking(t *testing.T) {
	server := sync.NewServer("")
	ctx := context.Background()
	vaultID := "test-vault-collision"
	vaultKey := make([]byte, 32)
	vaultKey[1] = 99

	// Setup Base
	dirA := t.TempDir()
	_ = os.WriteFile(filepath.Join(dirA, "shared.md"), []byte("Initial shared note"), 0o644)

	clientA := newTestClient(server.Handler(), "token-a")
	engineA := sync.NewEngine(clientA, vaultID, vaultKey, dirA)
	authorA := vault.CommitAuthor{ClientID: "client-a", Agent: "agent"}

	_, err := engineA.Push(ctx, authorA, "Initial commit")
	if err != nil {
		t.Fatalf("initial push error: %v", err)
	}

	// Client B pulls base
	dirB := t.TempDir()
	clientB := newTestClient(server.Handler(), "token-b")
	engineB := sync.NewEngine(clientB, vaultID, vaultKey, dirB)
	authorB := vault.CommitAuthor{ClientID: "client-b", Agent: "agent"}

	_, err = engineB.Pull(ctx)
	if err != nil {
		t.Fatalf("engineB pull error: %v", err)
	}

	// Client B modifies shared.md and pushes
	_ = os.WriteFile(filepath.Join(dirB, "shared.md"), []byte("Remote edits by B"), 0o644)
	_, err = engineB.Push(ctx, authorB, "Client B modified shared.md")
	if err != nil {
		t.Fatalf("engineB push error: %v", err)
	}

	// Client A modifies same shared.md differently
	_ = os.WriteFile(filepath.Join(dirA, "shared.md"), []byte("Local edits by A"), 0o644)

	// Client A runs Sync
	syncResA, err := engineA.Sync(ctx, authorA, "Client A modified shared.md and syncs")
	if err != nil {
		t.Fatalf("engineA Sync error: %v", err)
	}

	if len(syncResA.Conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(syncResA.Conflicts))
	}

	// Verify collision failsafe on disk:
	// shared.md contains Remote edits ("Remote edits by B")
	// shared.conflict-local.md contains Local edits ("Local edits by A")
	sharedContent, err := os.ReadFile(filepath.Join(dirA, "shared.md"))
	if err != nil {
		t.Fatalf("failed to read shared.md in dirA: %v", err)
	}
	if string(sharedContent) != "Remote edits by B" {
		t.Fatalf("shared.md expected 'Remote edits by B', got %q", string(sharedContent))
	}

	forkedContent, err := os.ReadFile(filepath.Join(dirA, "shared.conflict-local.md"))
	if err != nil {
		t.Fatalf("failed to read shared.conflict-local.md in dirA: %v", err)
	}
	if string(forkedContent) != "Local edits by A" {
		t.Fatalf("shared.conflict-local.md expected 'Local edits by A', got %q", string(forkedContent))
	}
}
