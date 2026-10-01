package okf_test

import (
	"testing"

	"github.com/okf-memory/okf-agent-memory/pkg/okf"
)

func TestScope_ResolveURI(t *testing.T) {
	// 1. Scoped bundle with okf://@: okf://@peter/django-5-rules/decisions/auth
	uri := "okf://@peter/django-5-rules/decisions/auth"
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

	// 2. Direct @-reference: @peter/django-5-rules/decisions/auth
	scopeBare, bundleIDBare, conceptIDBare, err := okf.ParseURI("@peter/django-5-rules/decisions/auth")
	if err != nil {
		t.Fatalf("ParseURI bare @ failed: %v", err)
	}
	if scopeBare != okf.ScopeVendor || bundleIDBare != "peter/django-5-rules" || conceptIDBare != "decisions/auth" {
		t.Errorf("unexpected bare @ parse result: %s, %s, %s", scopeBare, bundleIDBare, conceptIDBare)
	}

	// 3. Top-level vendor bundle with okf://@: okf://@nextjs-15/decisions/routing
	uriTop := "okf://@nextjs-15/decisions/routing"
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

	// 4. Direct @-reference top-level: @nextjs-15/decisions/routing
	scopeTopBare, bundleIDTopBare, conceptIDTopBare, err := okf.ParseURI("@nextjs-15/decisions/routing")
	if err != nil {
		t.Fatalf("ParseURI bare @ top failed: %v", err)
	}
	if scopeTopBare != okf.ScopeVendor || bundleIDTopBare != "nextjs-15" || conceptIDTopBare != "decisions/routing" {
		t.Errorf("unexpected bare @ top parse result: %s, %s, %s", scopeTopBare, bundleIDTopBare, conceptIDTopBare)
	}

	// 5. Unscoped target without @ (e.g. nextjs-15/decisions/routing) must reject ParseURI
	// so caller treats it as a local bundle concept!
	if _, _, _, err := okf.ParseURI("nextjs-15/decisions/routing"); err == nil {
		t.Errorf("expected error for unscoped target without @, but succeeded")
	}

	// 6. User scope: okf://user/preferences
	uriUser := "okf://user/preferences"
	scopeUser, _, conceptIDUser, err := okf.ParseURI(uriUser)
	if err != nil {
		t.Fatalf("ParseURI user failed: %v", err)
	}
	if scopeUser != okf.ScopeUser || conceptIDUser != "preferences" {
		t.Errorf("unexpected user scope parse result")
	}

	// 7. Markdown link normalization: @org/repo/path.md -> okf://@org/repo/path
	shorthand := "@peter/django-5-rules/decisions/auth.md"
	normURI := okf.NormalizeVendorLink(shorthand)
	if normURI != "okf://@peter/django-5-rules/decisions/auth" {
		t.Errorf("expected normalized URI, got %s", normURI)
	}

	shorthandTop := "@nextjs-15/decisions/routing.md"
	normURITop := okf.NormalizeVendorLink(shorthandTop)
	if normURITop != "okf://@nextjs-15/decisions/routing" {
		t.Errorf("expected normalized top URI, got %s", normURITop)
	}
}
