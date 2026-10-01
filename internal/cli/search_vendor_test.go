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

	// 4. Search for "django session" -> vendor concept returned as @peter/django-5-rules/decisions/auth
	outVendor := captureOutput(func() {
		searchCmd.Run([]string{"django", "--json"})
	})
	if !strings.Contains(outVendor, "@peter/django-5-rules/decisions/auth") {
		t.Errorf("expected vendor concept @ID in search output, got: %s", outVendor)
	}

	// 5. Test show @peter/django-5-rules/decisions/auth
	showCmd, ok := FindCommand("show")
	if !ok {
		t.Fatalf("show command not found")
	}
	outShow := captureOutput(func() {
		showCmd.Run([]string{"@peter/django-5-rules/decisions/auth"})
	})
	if !strings.Contains(outShow, "Vendor Django Auth") {
		t.Errorf("expected vendor title in show output, got: %s", outShow)
	}

	// 6. Test show @peter/... shorthand with .md
	outShowShort := captureOutput(func() {
		showCmd.Run([]string{"@peter/django-5-rules/decisions/auth.md"})
	})
	if !strings.Contains(outShowShort, "Vendor Django Auth") {
		t.Errorf("expected vendor title with shorthand show, got: %s", outShowShort)
	}

	// 7. Test show okf://@peter/django-5-rules/decisions/auth
	outShowURI := captureOutput(func() {
		showCmd.Run([]string{"okf://@peter/django-5-rules/decisions/auth"})
	})
	if !strings.Contains(outShowURI, "Vendor Django Auth") {
		t.Errorf("expected vendor title with URI show, got: %s", outShowURI)
	}

	// 8. Test top-level vendor bundle @nextjs-15/decisions/routing
	nextjsDir := filepath.Join(workDir, ".okf", "vendor", "nextjs-15")
	if err := os.MkdirAll(filepath.Join(nextjsDir, "decisions"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nextjsDir, "index.md"), []byte("# Next.js Index\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nextjsRouting := "---\ntype: Decision\ntitle: Next.js App Router\n---\nUse app router"
	if err := os.WriteFile(filepath.Join(nextjsDir, "decisions", "routing.md"), []byte(nextjsRouting), 0o644); err != nil {
		t.Fatal(err)
	}

	outNextjs := captureOutput(func() {
		showCmd.Run([]string{"@nextjs-15/decisions/routing"})
	})
	if !strings.Contains(outNextjs, "Next.js App Router") {
		t.Errorf("expected top-level vendor title with @nextjs-15, got: %s", outNextjs)
	}

	// 9. Verify that 'okf show nextjs-15/decisions/routing' WITHOUT @ does NOT look in vendor,
	// but searches the local 'knowledge' bundle (where it does not exist)
	origExit := exitFunc
	exitCalled := false
	exitFunc = func(code int) {
		exitCalled = true
	}
	defer func() { exitFunc = origExit }()

	// We capture stderr as well
	rErr, wErr, _ := os.Pipe()
	stderr := os.Stderr
	os.Stderr = wErr
	defer func() { os.Stderr = stderr }()

	showCmd.Run([]string{"nextjs-15/decisions/routing"})
	_ = wErr.Close()
	var errBuf bytes.Buffer
	_, _ = io.Copy(&errBuf, rErr)

	if !strings.Contains(errBuf.String(), "not found in 'knowledge'") {
		t.Errorf("expected target without @ to search in 'knowledge' bundle, got stderr: %s", errBuf.String())
	}
	if !exitCalled {
		t.Errorf("expected exitFunc to be called for missing concept")
	}

	// 10. Verify --scope flag behavior
	// --scope project should NOT return vendor concepts
	outScopeProject := captureOutput(func() {
		searchCmd.Run([]string{"django", "--scope", "project", "--json"})
	})
	if strings.Contains(outScopeProject, "@peter/django-5-rules") {
		t.Errorf("expected --scope project to exclude vendor concepts, got: %s", outScopeProject)
	}

	// --scope vendor should return vendor concepts even when searched specifically
	outScopeVendor := captureOutput(func() {
		searchCmd.Run([]string{"django", "--scope", "vendor", "--json"})
	})
	if !strings.Contains(outScopeVendor, "@peter/django-5-rules/decisions/auth") {
		t.Errorf("expected --scope vendor to include vendor concepts, got: %s", outScopeVendor)
	}

	// --scope vendor for "shadowed" returns vendor concept because local layer is excluded!
	outVendorShadowed := captureOutput(func() {
		searchCmd.Run([]string{"shadowed", "--scope", "vendor", "--json"})
	})
	if !strings.Contains(outVendorShadowed, "Vendor Shadowed Decision") {
		t.Errorf("expected --scope vendor to find vendor shadowed decision, got: %s", outVendorShadowed)
	}
}
