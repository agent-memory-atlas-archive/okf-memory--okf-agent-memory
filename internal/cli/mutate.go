package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/okf-memory/okf-agent-memory/pkg/okf"
)

// ============================================================================
// CREATE, UPDATE, RELATE (MUTATE)
// ============================================================================

func printCreateUsage() {
	fmt.Printf(`Create a new concept with automated frontmatter, timestamps, and log bookkeeping.

Usage:
  okf create <concept-id> [bundle] [flags]

Arguments:
  <concept-id>           Unique identifier for the concept (e.g. 'decisions/adr-001' or 'architecture/caching')
  [bundle]               Path to OKF bundle directory (default: 'knowledge' or '.')

Flags:
  --type <type>          Semantic concept type (e.g. 'Decision', 'Architecture', 'Fact', 'Requirement', 'Bug') [default: Fact]
  --title <title>        Human-readable title (defaults to concept basename)
  --desc <desc>          One-sentence summary of the concept
  --body <body>          Markdown body content
  --tags <tags>          Comma-separated list of searchable tags (e.g. 'auth,security,jwt')
  --actor <actor>        Author provenance identifier (default: 'agent/cli')
  --no-log               Skip appending an entry to knowledge/log.md
  --no-index             Skip updating immediate parent directory index.md
  --json                 Emit machine-readable JSON result

Examples:
  okf create decisions/adr-001 knowledge --type Decision --title "Database Architecture" --desc "Use PostgreSQL with connection pooling."
  okf create architecture/caching --type Architecture --desc "Redis cluster configuration." --json
`)
}

func printUpdateUsage() {
	fmt.Printf(`Update an existing concept's metadata, description, or body with automated bookkeeping.

Usage:
  okf update <concept-id> [bundle] [flags]

Arguments:
  <concept-id>           Unique identifier of existing concept to mutate
  [bundle]               Path to OKF bundle directory (default: 'knowledge' or '.')

Flags:
  --desc <desc>          Update the one-sentence description
  --title <title>        Update the title
  --body <body>          Update the markdown body content
  --type <type>          Update the concept type
  --status <status>      Set concept status (e.g. 'active', 'deprecated', 'draft')
  --tags <tags>          Replace tags with comma-separated list
  --actor <actor>        Author provenance identifier (default: 'agent/cli')
  --no-log               Skip appending to knowledge/log.md
  --no-index             Skip updating parent directory index.md
  --json                 Emit machine-readable JSON result

Examples:
  okf update architecture/database knowledge --desc "Migrated from SQLite to PostgreSQL 16 on RDS."
  okf update decisions/adr-001 --status deprecated --desc "Superseded by adr-008." --json
`)
}

func printRelateUsage() {
	fmt.Printf(`Connect two concepts with a relative link and semantic context.

Usage:
  okf relate <source-id> <target-id> [bundle] --desc <context> [flags]

Arguments:
  <source-id>            Source concept identifier
  <target-id>            Target concept identifier to link to
  [bundle]               Path to OKF bundle directory (default: 'knowledge' or '.')

Flags:
  --desc <prose>         Semantic description of the relationship (required)
  --actor <actor>        Author provenance identifier (default: 'agent/cli')
  --no-log               Skip appending to knowledge/log.md
  --json                 Emit machine-readable JSON result

Examples:
  okf relate decisions/adr-008 architecture/database knowledge --desc "implements connection pooling strategy"
`)
}

func cmdCreate(args []string) {
	if len(args) == 0 || hasHelpFlag(args) {
		printCreateUsage()
		if len(args) == 0 {
			os.Exit(1)
		}
		return
	}

	rawID := strings.TrimSpace(args[0])
	if err := okf.ValidateConceptID(rawID); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	conceptID := strings.TrimSuffix(rawID, ".md")
	var subArgs []string
	if len(args) > 1 {
		subArgs = args[1:]
	}

	fs := flag.NewFlagSet("create", flag.ExitOnError)
	cType := fs.String("type", "Fact", "Concept type (required)")
	title := fs.String("title", "", "Concept title")
	desc := fs.String("desc", "", "Concept description (one sentence)")
	body := fs.String("body", "", "Concept body content")
	tagsStr := fs.String("tags", "", "Comma-separated tags")
	actor := fs.String("actor", "agent/cli", "Author actor string")
	noLog := fs.Bool("no-log", false, "Skip appending to log.md")
	noIndex := fs.Bool("no-index", false, "Skip updating parent index.md")
	jsonOut := fs.Bool("json", false, "Emit JSON result")

	bundleDir, flagArgs := defaultBundle(subArgs)
	_ = fs.Parse(flagArgs)

	titleVal := *title
	if strings.TrimSpace(titleVal) == "" {
		titleVal = filepath.Base(conceptID)
	}

	var tags []string
	if *tagsStr != "" {
		for _, t := range strings.Split(*tagsStr, ",") {
			trimmed := strings.TrimSpace(t)
			if trimmed != "" {
				tags = append(tags, trimmed)
			}
		}
	}

	c := &okf.Concept{
		ID:          conceptID,
		Path:        conceptID + ".md",
		Type:        *cType,
		Title:       titleVal,
		Description: *desc,
		Tags:        tags,
		Body:        *body,
	}

	err := okf.SaveConcept(bundleDir, c, okf.SaveOptions{
		IsNew:     true,
		AutoLog:   !*noLog,
		AutoIndex: !*noIndex,
		Actor:     *actor,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *jsonOut {
		data, _ := json.Marshal(map[string]string{
			"status":     "success",
			"concept_id": conceptID,
			"path":       c.Path,
		})
		fmt.Println(string(data))
	} else {
		fmt.Printf("Created concept '%s' (%s) in '%s'\n", c.Path, c.Title, bundleDir)
	}
}

func cmdUpdate(args []string) {
	if len(args) == 0 || hasHelpFlag(args) {
		printUpdateUsage()
		if len(args) == 0 {
			os.Exit(1)
		}
		return
	}

	rawID := strings.TrimSpace(args[0])
	if err := okf.ValidateConceptID(rawID); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	conceptID := strings.TrimSuffix(rawID, ".md")
	var subArgs []string
	if len(args) > 1 {
		subArgs = args[1:]
	}

	fs := flag.NewFlagSet("update", flag.ExitOnError)
	title := fs.String("title", "", "Updated concept title")
	desc := fs.String("desc", "", "Updated description")
	body := fs.String("body", "", "Updated body content")
	actor := fs.String("actor", "agent/cli", "Author actor string")
	noLog := fs.Bool("no-log", false, "Skip appending to log.md")
	noIndex := fs.Bool("no-index", false, "Skip updating parent index.md")
	jsonOut := fs.Bool("json", false, "Emit JSON result")

	bundleDir, flagArgs := defaultBundle(subArgs)
	_ = fs.Parse(flagArgs)

	b, err := okf.LoadBundle(bundleDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading bundle: %v\n", err)
		os.Exit(2)
	}

	c, ok := b.Concepts[conceptID]
	if !ok {
		fmt.Fprintf(os.Stderr, "Concept '%s' not found\n", conceptID)
		os.Exit(1)
	}

	isPassed := func(name string) bool {
		found := false
		fs.Visit(func(f *flag.Flag) {
			if f.Name == name {
				found = true
			}
		})
		return found
	}

	if isPassed("title") {
		c.Title = *title
	}
	if isPassed("desc") {
		c.Description = *desc
	}
	if isPassed("body") {
		c.Body = *body
	}

	err = okf.SaveConcept(bundleDir, c, okf.SaveOptions{
		IsNew:     false,
		AutoLog:   !*noLog,
		AutoIndex: !*noIndex,
		Actor:     *actor,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *jsonOut {
		data, _ := json.Marshal(map[string]string{
			"status":     "success",
			"concept_id": conceptID,
			"path":       c.Path,
		})
		fmt.Println(string(data))
	} else {
		fmt.Printf("Updated concept '%s' in '%s'\n", c.Path, bundleDir)
	}
}

func cmdRelate(args []string) {
	if len(args) < 2 || hasHelpFlag(args) {
		printRelateUsage()
		if len(args) < 2 {
			os.Exit(1)
		}
		return
	}

	sourceID := args[0]
	targetID := args[1]

	var subArgs []string
	if len(args) > 2 {
		subArgs = args[2:]
	}

	fs := flag.NewFlagSet("relate", flag.ExitOnError)
	desc := fs.String("desc", "", "Description explaining the relationship")
	actor := fs.String("actor", "agent/cli", "Author actor string")
	jsonOut := fs.Bool("json", false, "Emit JSON result")

	bundleDir, flagArgs := defaultBundle(subArgs)
	_ = fs.Parse(flagArgs)

	err := okf.RelateConcepts(bundleDir, sourceID, targetID, *desc, *actor)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *jsonOut {
		data, _ := json.Marshal(map[string]string{
			"status": "success",
			"source": strings.TrimSpace(strings.TrimSuffix(sourceID, ".md")),
			"target": strings.TrimSpace(strings.TrimSuffix(targetID, ".md")),
		})
		fmt.Println(string(data))
	} else {
		fmt.Printf("Linked '%s' -> '%s' in '%s'\n", strings.TrimSpace(strings.TrimSuffix(sourceID, ".md")), strings.TrimSpace(strings.TrimSuffix(targetID, ".md")), bundleDir)
	}
}
