package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/okf-memory/okf-agent-memory/pkg/okf"
	"github.com/okf-memory/okf-agent-memory/pkg/okf/aag"
)

func cmdAgents(args []string) {
	if len(args) < 1 {
		printAgentsUsage()
		os.Exit(1)
	}

	subcmd := args[0]
	subArgs := args[1:]

	switch subcmd {
	case "lint":
		if err := runAgentsLint(subArgs); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "init":
		if err := runAgentsInit(subArgs); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "link":
		if err := runAgentsLink(subArgs); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "check":
		if err := runAgentsCheck(subArgs); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown agents command '%s'\n\n", subcmd)
		printAgentsUsage()
		os.Exit(1)
	}
}

func printAgentsUsage() {
	fmt.Printf(`OKF Agents Management & AAG Suite

Usage:
  okf agents <command> [arguments] [flags]

Commands:
  lint [file]    Lint AGENTS.md against AAG-001 to AAG-005 and token budget
  init           Scaffold domain-specific AGENTS.md codex (--domain=software|research|legal|...)
  link           Create SSoT symlinks (CLAUDE.md, .cursorrules, .windsurfrules, Copilot)
  check          Run all-in-one CI validation (lint + symlink integrity)

Flags:
  --domain <name>  Domain profile (software, research, legal, coaching, books)
  --budget <int>   Token budget cap (default: 400 for file, 150 for micro-syntax)
  --strict         Treat warnings as fatal errors
  --force          Overwrite existing files or symlinks
  --check          Verify symlink integrity without mutating
  --json           Emit machine-readable JSON output
`)
}

func runAgentsLint(args []string) error {
	fs := flag.NewFlagSet("agents lint", flag.ContinueOnError)
	strict := fs.Bool("strict", false, "Treat warnings as errors")
	budget := fs.Int("budget", aag.DefaultBudgetLimit, "Max token budget")
	jsonOut := fs.Bool("json", false, "Output results as JSON")

	if err := fs.Parse(args); err != nil {
		return err
	}

	targetFile := "AGENTS.md"
	if len(fs.Args()) > 0 {
		targetFile = fs.Args()[0]
	}

	res, err := aag.LintFile(targetFile, aag.LinterOptions{
		BudgetLimit: *budget,
		Strict:      *strict,
	})
	if err != nil {
		return fmt.Errorf("failed to lint %s: %w", targetFile, err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(res)
	}

	fmt.Printf("AAG Linter: %s\n", targetFile)
	fmt.Printf("Token Stats: %d estimated tokens (Budget: %d tokens)\n\n",
		res.TokenStats.EstimatedTokens, res.TokenStats.BudgetLimit)

	if len(res.Findings) == 0 {
		fmt.Printf("✓ All AAG rules passed (0 errors, 0 warnings). 100%% conformant.\n")
		return nil
	}

	for _, f := range res.Findings {
		prefix := "WARN"
		if f.Severity == aag.SeverityError {
			prefix = "ERROR"
		}
		fmt.Printf("[%s] %s (line %d): %s\n", prefix, f.RuleID, f.Line, f.Message)
		if f.Snippet != "" {
			fmt.Printf("       > %s\n", f.Snippet)
		}
	}

	fmt.Printf("\nSummary: %d error(s), %d warning(s)\n", res.ErrorCount, res.WarnCount)
	if !res.Passed {
		return fmt.Errorf("AAG linting failed with %d error(s)", res.ErrorCount)
	}
	return nil
}

func runAgentsInit(args []string) error {
	fs := flag.NewFlagSet("agents init", flag.ContinueOnError)
	domain := fs.String("domain", "software", "Domain codex profile")
	name := fs.String("name", "", "Project name")
	root := fs.String("root", ".", "Target repository root directory")
	force := fs.Bool("force", false, "Overwrite existing AGENTS.md")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) > 0 && *root == "." {
		*root = fs.Args()[0]
	}

	targetPath := filepath.Join(*root, "AGENTS.md")
	if _, err := os.Stat(targetPath); err == nil && !*force {
		return fmt.Errorf("AGENTS.md already exists at %s (use --force to overwrite)", targetPath)
	}

	content, err := okf.GenerateAgentsMarkdown(*name, *domain)
	if err != nil {
		return fmt.Errorf("failed to generate AGENTS.md: %w", err)
	}

	if err := os.MkdirAll(*root, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", *root, err)
	}

	if err := os.WriteFile(targetPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", targetPath, err)
	}

	fmt.Printf("✓ Created %s with %q domain codex.\n", targetPath, *domain)
	return nil
}

func runAgentsLink(args []string) error {
	fs := flag.NewFlagSet("agents link", flag.ContinueOnError)
	root := fs.String("root", ".", "Target repository root directory")
	force := fs.Bool("force", false, "Overwrite existing non-symlink files")
	checkOnly := fs.Bool("check", false, "Check symlink status without modifying")
	jsonOut := fs.Bool("json", false, "Output results as JSON")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) > 0 && *root == "." {
		*root = fs.Args()[0]
	}

	if *checkOnly {
		statuses, err := okf.CheckToolSymlinks(*root)
		if err != nil {
			return err
		}

		if *jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(statuses)
		}

		hasErrors := false
		fmt.Printf("Checking SSoT tool symlinks in %s:\n\n", *root)
		for _, st := range statuses {
			if st.IsValid {
				fmt.Printf("✓ %-16s %s -> %s\n", st.ToolName, st.ToolPath, st.ActualTarget)
			} else {
				hasErrors = true
				fmt.Printf("✗ %-16s %s (error: %s)\n", st.ToolName, st.ToolPath, st.ErrorMessage)
			}
		}

		if hasErrors {
			return fmt.Errorf("one or more tool symlinks are missing or invalid (run 'okf agents link' to repair)")
		}
		fmt.Printf("\nAll SSoT tool symlinks are valid and drift-free.\n")
		return nil
	}

	results, err := okf.CreateToolSymlinks(*root, *force)
	if err != nil {
		return err
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	fmt.Printf("Created SSoT tool symlinks in %s:\n\n", *root)
	hasErrors := false
	for _, res := range results {
		if res.Error != "" {
			hasErrors = true
			fmt.Printf("✗ %-16s %s (error: %s)\n", res.ToolName, res.Path, res.Error)
		} else if res.Created {
			action := "created"
			if res.Overwritten {
				action = "overwritten"
			}
			fmt.Printf("✓ %-16s %s -> %s (%s)\n", res.ToolName, res.Path, res.Target, action)
		} else {
			fmt.Printf("✓ %-16s %s -> %s (already valid)\n", res.ToolName, res.Path, res.Target)
		}
	}

	if hasErrors {
		return fmt.Errorf("failed to create one or more symlinks")
	}
	return nil
}

func runAgentsCheck(args []string) error {
	fs := flag.NewFlagSet("agents check", flag.ContinueOnError)
	root := fs.String("root", ".", "Target repository root directory")
	strict := fs.Bool("strict", true, "Treat warnings as errors")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) > 0 && *root == "." {
		*root = fs.Args()[0]
	}

	agentsFile := filepath.Join(*root, "AGENTS.md")
	fmt.Printf("==> 1. Linting AAG rules in %s...\n", agentsFile)
	if err := runAgentsLint([]string{agentsFile, fmt.Sprintf("--strict=%t", *strict)}); err != nil {
		return err
	}

	fmt.Printf("\n==> 2. Verifying SSoT tool symlinks in %s...\n", *root)
	if err := runAgentsLink([]string{"--root", *root, "--check"}); err != nil {
		return err
	}

	fmt.Printf("\n✓ All agent governance and symlink integrity checks passed!\n")
	return nil
}
