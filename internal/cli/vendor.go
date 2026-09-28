package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/okf-memory/okf-agent-memory/pkg/lock"
)

var exitFunc = os.Exit

func printVendorUsage() {
	fmt.Println(`Usage:
  okf vendor <command>

Commands:
  list     List installed vendor bundles
  remove   Uninstall a vendor bundle (okf vendor remove <bundle-id>)`)
}

func cmdVendor(args []string) {
	if len(args) == 0 {
		printVendorUsage()
		return
	}

	switch args[0] {
	case "list":
		lf, err := lock.ReadLockfile("okf.lock")
		if err != nil || len(lf.Bundles) == 0 {
			fmt.Println("No vendor bundles installed.")
			return
		}
		fmt.Printf("Installed Vendor Bundles (%d):\n", len(lf.Bundles))
		for _, b := range lf.Bundles {
			fmt.Printf("  - %-24s %-10s (%s)\n", b.ID, b.Version, b.Source)
		}

	case "remove":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: okf vendor remove <bundle-id>")
			exitFunc(1)
			return
		}
		bundleID := args[1]
		vendorDir := filepath.Join(".okf", "vendor", filepath.FromSlash(bundleID))

		dirExists := false
		if info, err := os.Stat(vendorDir); err == nil && info.IsDir() {
			dirExists = true
		}

		lf, err := lock.ReadLockfile("okf.lock")
		lockHasBundle := false
		if err == nil {
			for _, b := range lf.Bundles {
				if b.ID == bundleID {
					lockHasBundle = true
					break
				}
			}
		}

		if !dirExists && !lockHasBundle {
			fmt.Fprintf(os.Stderr, "Error: vendor bundle %q is not installed\n", bundleID)
			exitFunc(1)
			return
		}

		if dirExists {
			_ = os.RemoveAll(vendorDir)
			if strings.Contains(bundleID, "/") {
				parentDir := filepath.Dir(vendorDir)
				if entries, err := os.ReadDir(parentDir); err == nil && len(entries) == 0 {
					_ = os.Remove(parentDir)
				}
			}
		}

		if lockHasBundle && lf != nil {
			if lf.Remove(bundleID) {
				_ = lock.WriteLockfile("okf.lock", lf)
			}
		}
		fmt.Printf("Removed vendor bundle %s\n", bundleID)

	default:
		printVendorUsage()
	}
}
