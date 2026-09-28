package okf_test

import (
	"testing"

	"github.com/okf-memory/okf-agent-memory/pkg/okf"
)

func TestScope_ResolveURI(t *testing.T) {
	// okf://vendor/peter/django-5-rules/decisions/auth
	uri := "okf://vendor/peter/django-5-rules/decisions/auth"
	scope, bundleID, conceptID, err := okf.ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI failed: %v", err)
	}
	if scope != okf.ScopeVendor {
		t.Errorf("expected ScopeVendor, got %s", scope)
	}
	if bundleID != "peter/django-5-rules" {
		t.Errorf("expected peter/django-5-rules, got %s", bundleID)
	}
	if conceptID != "decisions/auth" {
		t.Errorf("expected decisions/auth, got %s", conceptID)
	}

	// Top-level vendor bundle: okf://vendor/nextjs-15/decisions/routing
	uriTop := "okf://vendor/nextjs-15/decisions/routing"
	scopeTop, bundleIDTop, conceptIDTop, err := okf.ParseURI(uriTop)
	if err != nil {
		t.Fatalf("ParseURI top-level failed: %v", err)
	}
	if scopeTop != okf.ScopeVendor {
		t.Errorf("expected ScopeVendor, got %s", scopeTop)
	}
	if bundleIDTop != "nextjs-15" {
		t.Errorf("expected nextjs-15, got %s", bundleIDTop)
	}
	if conceptIDTop != "decisions/routing" {
		t.Errorf("expected decisions/routing, got %s", conceptIDTop)
	}

	// User scope: okf://user/preferences
	uriUser := "okf://user/preferences"
	scopeUser, _, conceptIDUser, err := okf.ParseURI(uriUser)
	if err != nil {
		t.Fatalf("ParseURI user failed: %v", err)
	}
	if scopeUser != okf.ScopeUser || conceptIDUser != "preferences" {
		t.Errorf("unexpected user scope parse result")
	}

	// @vendor/ shorthand alias
	shorthand := "@vendor/peter/django-5-rules/decisions/auth.md"
	normURI := okf.NormalizeVendorLink(shorthand)
	if normURI != "okf://vendor/peter/django-5-rules/decisions/auth" {
		t.Errorf("expected normalized URI, got %s", normURI)
	}
}
