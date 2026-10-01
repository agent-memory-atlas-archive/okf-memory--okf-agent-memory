package okf

import (
	"fmt"
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

func NormalizeVendorLink(link string) string {
	if strings.HasPrefix(link, "@vendor/") {
		rel := strings.TrimPrefix(link, "@vendor/")
		rel = strings.TrimSuffix(rel, ".md")
		return "okf://vendor/" + rel
	}
	return link
}

func ParseURI(rawURI string) (Scope, string, string, error) {
	if !strings.HasPrefix(rawURI, "okf://") {
		return "", "", "", fmt.Errorf("not an okf:// URI")
	}
	stripped := strings.TrimPrefix(rawURI, "okf://")
	parts := strings.Split(stripped, "/")
	if len(parts) < 2 {
		return "", "", "", fmt.Errorf("malformed okf:// URI: %s", rawURI)
	}

	scope := Scope(parts[0])
	switch scope {
	case ScopeVendor:
		// Format: okf://vendor/<bundle-id>/<concept-id>
		// Can be: okf://vendor/nextjs-15/decisions/routing (len 4)
		// Or: okf://vendor/peter/django-5-rules/decisions/auth (len 5)
		// Or with leading @: okf://vendor/@peter/django-5-rules/decisions/auth
		if len(parts) >= 5 {
			bundleID := strings.TrimPrefix(parts[1], "@") + "/" + parts[2]
			conceptID := strings.Join(parts[3:], "/")
			return scope, bundleID, conceptID, nil
		}
		if len(parts) >= 4 && strings.Contains(parts[2], "-") {
			bundleID := strings.TrimPrefix(parts[1], "@") + "/" + parts[2]
			conceptID := strings.Join(parts[3:], "/")
			return scope, bundleID, conceptID, nil
		}
		bundleID := strings.TrimPrefix(parts[1], "@")
		conceptID := strings.Join(parts[2:], "/")
		return scope, bundleID, conceptID, nil

	case ScopeUser, ScopeProject, ScopeSystem:
		conceptID := strings.Join(parts[1:], "/")
		return scope, "", conceptID, nil

	default:
		return "", "", "", fmt.Errorf("unknown scope: %s", scope)
	}
}
