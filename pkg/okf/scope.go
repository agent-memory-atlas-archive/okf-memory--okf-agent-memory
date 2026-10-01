package okf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Scope string

const (
	ScopeProject Scope = "project"
	ScopeVendor  Scope = "vendor"
	ScopeUser    Scope = "user"
	ScopeSystem  Scope = "system"
)

const (
	PriorityProject = 100
	PriorityVendor  = 70
	PriorityUser    = 50
	PrioritySystem  = 10
)

type LayeredSearchResult struct {
	ConceptID string  `json:"concept_id"`
	Scope     Scope   `json:"scope"`
	Priority  int     `json:"priority"`
	Score     float64 `json:"score"`
	Title     string  `json:"title"`
}

// NormalizeVendorLink normalizes a vendor link or markdown reference into an okf://@ URI.
// For example:
//
//	@peter/django-5-rules/decisions/auth.md -> okf://@peter/django-5-rules/decisions/auth
//	@nextjs-15/decisions/routing.md        -> okf://@nextjs-15/decisions/routing
func NormalizeVendorLink(link string) string {
	if strings.HasPrefix(link, "@") {
		rel := strings.TrimPrefix(link, "@")
		rel = strings.TrimSuffix(rel, ".md")
		return "okf://@" + rel
	}
	if strings.HasPrefix(link, "user:") {
		rel := strings.TrimPrefix(link, "user:")
		rel = strings.TrimPrefix(rel, "/")
		rel = strings.TrimSuffix(rel, ".md")
		return "okf://user/" + rel
	}
	if strings.HasPrefix(link, "system:") {
		rel := strings.TrimPrefix(link, "system:")
		rel = strings.TrimPrefix(rel, "/")
		rel = strings.TrimSuffix(rel, ".md")
		return "okf://system/" + rel
	}
	return link
}

// ParseVendorRef splits a bundle-relative target (e.g. "nextjs-15/decisions/routing" or
// "peter/django-5-rules/decisions/auth") into bundleID and conceptID.
// It checks vendorRoot (default .okf/vendor) for existing index.md files to disambiguate
// between 1-part and 2-part bundle names, with a deterministic fallback heuristic.
func ParseVendorRef(target, vendorRoot string) (string, string, error) {
	clean := strings.TrimPrefix(target, "@")
	clean = strings.Trim(clean, "/")
	parts := strings.Split(clean, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("malformed vendor reference (must contain bundle and concept): %s", target)
	}

	if vendorRoot == "" {
		vendorRoot = filepath.Join(".okf", "vendor")
	}

	// 1. Check disk: does 2-segment bundle exist (.okf/vendor/org/bundle/index.md)?
	if len(parts) >= 3 {
		if _, err := os.Stat(filepath.Join(vendorRoot, parts[0], parts[1], "index.md")); err == nil {
			return parts[0] + "/" + parts[1], strings.Join(parts[2:], "/"), nil
		}
	}

	// 2. Check disk: does 1-segment bundle exist (.okf/vendor/bundle/index.md)?
	if _, err := os.Stat(filepath.Join(vendorRoot, parts[0], "index.md")); err == nil {
		return parts[0], strings.Join(parts[1:], "/"), nil
	}

	// 3. Fallback heuristic: standard concept directories
	if len(parts) >= 3 && !isKnownConceptCategory(parts[1]) {
		return parts[0] + "/" + parts[1], strings.Join(parts[2:], "/"), nil
	}

	return parts[0], strings.Join(parts[1:], "/"), nil
}

func isKnownConceptCategory(s string) bool {
	switch strings.ToLower(s) {
	case "decisions", "architecture", "convention", "roadmap", "project",
		"requirements", "domain", "runbooks", "concepts", "facts", "entities",
		"guides", "playbooks", "specs", "adr", "rfc", "api":
		return true
	default:
		return false
	}
}

// ParseURI parses canonical okf:// URIs or @-prefixed vendor references.
// Examples:
//
//	okf://@peter/django-5-rules/decisions/auth -> (ScopeVendor, "peter/django-5-rules", "decisions/auth", nil)
//	okf://@nextjs-15/decisions/routing         -> (ScopeVendor, "nextjs-15", "decisions/routing", nil)
//	okf://user/preferences                     -> (ScopeUser, "", "preferences", nil)
//	okf://system/compliance                    -> (ScopeSystem, "", "compliance", nil)
//	@peter/django-5-rules/decisions/auth       -> (ScopeVendor, "peter/django-5-rules", "decisions/auth", nil)
func ParseURI(rawURI string) (Scope, string, string, error) {
	if strings.HasPrefix(rawURI, "@") {
		bundleID, conceptID, err := ParseVendorRef(rawURI, "")
		if err != nil {
			return "", "", "", err
		}
		return ScopeVendor, bundleID, conceptID, nil
	}

	if strings.HasPrefix(rawURI, "user:") {
		conceptID := strings.TrimPrefix(rawURI, "user:")
		conceptID = strings.TrimPrefix(conceptID, "/")
		conceptID = strings.TrimSuffix(conceptID, ".md")
		return ScopeUser, "", conceptID, nil
	}
	if strings.HasPrefix(rawURI, "system:") {
		conceptID := strings.TrimPrefix(rawURI, "system:")
		conceptID = strings.TrimPrefix(conceptID, "/")
		conceptID = strings.TrimSuffix(conceptID, ".md")
		return ScopeSystem, "", conceptID, nil
	}

	if !strings.HasPrefix(rawURI, "okf://") {
		return "", "", "", fmt.Errorf("not an okf:// URI or @vendor reference")
	}

	stripped := strings.TrimPrefix(rawURI, "okf://")
	if strings.HasPrefix(stripped, "@") {
		bundleID, conceptID, err := ParseVendorRef(stripped, "")
		if err != nil {
			return "", "", "", err
		}
		return ScopeVendor, bundleID, conceptID, nil
	}

	parts := strings.Split(stripped, "/")
	if len(parts) < 2 {
		return "", "", "", fmt.Errorf("malformed okf:// URI: %s", rawURI)
	}

	scope := Scope(parts[0])
	switch scope {
	case ScopeUser, ScopeProject, ScopeSystem:
		conceptID := strings.Join(parts[1:], "/")
		return scope, "", conceptID, nil
	default:
		return "", "", "", fmt.Errorf("unknown scope in URI: %s", scope)
	}
}

// IsExternalLink checks whether a link href points to an external or scoped target
// (vendor @bundle/..., user:..., system:..., or explicit scheme like okf://, https://, etc.)
// and therefore must never cause broken-link failures in local bundle validation.
func IsExternalLink(href string) bool {
	norm := strings.TrimSpace(href)
	if norm == "" {
		return false
	}
	return strings.Contains(norm, "://") ||
		strings.HasPrefix(norm, "@") ||
		strings.HasPrefix(norm, "user:") ||
		strings.HasPrefix(norm, "system:") ||
		strings.HasPrefix(norm, "mailto:")
}
