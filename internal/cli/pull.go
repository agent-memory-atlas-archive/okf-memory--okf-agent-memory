package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/okf-memory/okf-agent-memory/pkg/lock"
	"github.com/okf-memory/okf-agent-memory/pkg/registry"
)

var newRegistryClient = func(baseURL string) *registry.Client {
	return registry.NewClient(baseURL)
}

func printPullUsage() {
	fmt.Println(`Usage:
  okf pull <bundle-id|git-url> [flags]

Flags:
  --registry <url>   Override canonical registry endpoint (default: https://registry.okf-memory.dev)
  --force            Overwrite existing vendor installation if present

Examples:
  okf pull nextjs-15
  okf pull peter/django-5-rules
  okf pull github.com/acme/agent-rules`)
}

func cmdPull(args []string) {
	fs := flag.NewFlagSet("pull", flag.ExitOnError)
	registryURL := fs.String("registry", registry.DefaultRegistryURL, "Registry endpoint")
	force := fs.Bool("force", false, "Overwrite existing vendor installation")
	fs.Usage = printPullUsage

	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	if fs.NArg() < 1 {
		printPullUsage()
		os.Exit(1)
	}

	target := fs.Arg(0)
	client := newRegistryClient(*registryURL)

	fmt.Printf("Resolving %s from %s...\n", target, client.BaseURL)
	manifest, err := client.Resolve(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	vendorDir := filepath.Join(".okf", "vendor", filepath.FromSlash(manifest.ID))
	if _, err := os.Stat(vendorDir); err == nil && !*force {
		fmt.Printf("Bundle %q is already installed in %s. Use --force to reinstall.\n", manifest.ID, vendorDir)
		return
	}

	fmt.Printf("Downloading %s (version: %s)...\n", manifest.ID, manifest.Version)
	if err := client.DownloadAndExtract(manifest, vendorDir); err != nil {
		fmt.Fprintf(os.Stderr, "Download error: %v\n", err)
		os.Exit(1)
	}

	lockPath := "okf.lock"
	lf, err := lock.ReadLockfile(lockPath)
	if err != nil {
		lf = &lock.Lockfile{Version: 1}
	}
	lf.Upsert(lock.BundleEntry{
		ID:          manifest.ID,
		Source:      target,
		Version:     manifest.Version,
		Hash:        manifest.Hash,
		InstalledAt: time.Now().UTC(),
	})
	if err := lock.WriteLockfile(lockPath, lf); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to update okf.lock: %v\n", err)
	}

	fmt.Printf("Successfully installed %s into %s\n", manifest.ID, vendorDir)
}
