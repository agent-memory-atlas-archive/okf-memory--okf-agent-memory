package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/okf-memory/okf-agent-memory/pkg/lock"
)

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
			fmt.Println("Usage: okf vendor remove <bundle-id>")
			os.Exit(1)
		}
		bundleID := args[1]
		vendorDir := filepath.Join(".okf", "vendor", filepath.FromSlash(bundleID))
		_ = os.RemoveAll(vendorDir)

		lf, err := lock.ReadLockfile("okf.lock")
		if err == nil {
			if lf.Remove(bundleID) {
				_ = lock.WriteLockfile("okf.lock", lf)
			}
		}
		fmt.Printf("Removed vendor bundle %s\n", bundleID)

	default:
		printVendorUsage()
	}
}
