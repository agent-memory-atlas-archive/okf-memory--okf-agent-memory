package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func captureOutput(f func()) string {
	r, w, _ := os.Pipe()
	stdout := os.Stdout
	os.Stdout = w
	defer func() {
		os.Stdout = stdout
	}()

	f()
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestSearch_VendorLayeringAndShadowing(t *testing.T) {
	workDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// 1. Create primary bundle in ./knowledge
	projDir := filepath.Join(workDir, "knowledge")
	if err := os.MkdirAll(filepath.Join(projDir, "decisions"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "index.md"), []byte("# Project Index\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	shadowedProj := `---
type: Decision
title: Local Shadowed Decision
governance: constraint
---
Local project decision overrides vendor
`
	if err := os.WriteFile(filepath.Join(projDir, "decisions", "shadowed.md"), []byte(shadowedProj), 0o644); err != nil {
		t.Fatal(err)
	}

	// 2. Create vendor bundle in .okf/vendor/peter/django-5-rules
	vendorDir := filepath.Join(workDir, ".okf", "vendor", "peter", "django-5-rules")
	if err := os.MkdirAll(filepath.Join(vendorDir, "decisions"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendorDir, "index.md"), []byte("# Vendor Index\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	shadowedVendor := `---
type: Decision
title: Vendor Shadowed Decision
---
Vendor rule should be shadowed
`
	if err := os.WriteFile(filepath.Join(vendorDir, "decisions", "shadowed.md"), []byte(shadowedVendor), 0o644); err != nil {
		t.Fatal(err)
	}
	vendorAuth := `---
type: Decision
title: Vendor Django Auth
---
Use session authentication in Django 5
`
	if err := os.WriteFile(filepath.Join(vendorDir, "decisions", "auth.md"), []byte(vendorAuth), 0o644); err != nil {
		t.Fatal(err)
	}

	// 3. Search for "shadowed" -> only project decision returned (vendor shadowed)
	searchCmd, ok := FindCommand("search")
	if !ok {
		t.Fatalf("search command not found")
	}

	out := captureOutput(func() {
		searchCmd.Run([]string{"shadowed", "--json"})
	})

	if !strings.Contains(out, "Local Shadowed Decision") {
		t.Errorf("expected local decision in search output, got: %s", out)
	}
	if strings.Contains(out, "Vendor Shadowed Decision") {
		t.Errorf("expected vendor decision to be shadowed, but was present: %s", out)
	}

	// 4. Search for "django session" -> vendor concept returned
	outVendor := captureOutput(func() {
		searchCmd.Run([]string{"django", "--json"})
	})
	if !strings.Contains(outVendor, "okf://vendor/peter/django-5-rules/decisions/auth") {
		t.Errorf("expected vendor concept URI in search output, got: %s", outVendor)
	}

	// 5. Test show okf://vendor/...
	showCmd, ok := FindCommand("show")
	if !ok {
		t.Fatalf("show command not found")
	}
	outShow := captureOutput(func() {
		showCmd.Run([]string{"okf://vendor/peter/django-5-rules/decisions/auth"})
	})
	if !strings.Contains(outShow, "Vendor Django Auth") {
		t.Errorf("expected vendor title in show output, got: %s", outShow)
	}

	// 6. Test show @vendor/... shorthand
	outShowShort := captureOutput(func() {
		showCmd.Run([]string{"@vendor/peter/django-5-rules/decisions/auth.md"})
	})
	if !strings.Contains(outShowShort, "Vendor Django Auth") {
		t.Errorf("expected vendor title with shorthand show, got: %s", outShowShort)
	}
}
