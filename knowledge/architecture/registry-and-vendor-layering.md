---
type: Decision
title: "OKF Registry Client, Dependency Locking, and Multi-Scope Vendor Layering"
description: "Decentralized package registry client, zero-dependency okf.lock manifest, multi-scope priority layering, and hermetic okf:// cross-scope URI routing."
tags: [registry, vendor, lockfile, multi-scope, layering, pull]
generated: { by: agent/mcp, at: "2026-10-01T08:24:47Z" }
status: stable
---

## Context & Motivation

As autonomous agents and human developers scale memory across projects, standard domain rules, framework best practices (e.g. Next.js, Django), and organizational standards must be shared without manual copy-pasting or repository pollution.

This architectural decision codifies the decentralized OKF Registry distribution model, the zero-dependency `okf.lock` manifest format, multi-scope prioritization semantics, and hermetic `okf://` cross-scope linking.

---

## 1. Multi-Scope Hierarchy & Priority Weighting

OKF Agent Memory evaluates concepts across multiple discrete scopes, prioritizing local project context while enabling seamless upstream dependency inheritance:

```mermaid
flowchart TD
    subgraph S1 ["Scope: Project (Priority 100)"]
        P["./knowledge/ (Local Project Memory)"]
    end
    subgraph S2 ["Scope: Vendor (Priority 70)"]
        V[".okf/vendor/<bundle-id>/ (External Dependencies)"]
    end
    subgraph S3 ["Scope: User (Priority 50)"]
        U["~/.okf/ (Personal Machine Memory)"]
    end
    subgraph S4 ["Scope: System (Priority 10)"]
        SYS["/etc/okf/ (Global Enterprise Standards)"]
    end

    P -->|Shadows identical IDs| V
    V -->|Shadows identical IDs| U
    U -->|Shadows identical IDs| SYS
```

* **Shadowing Invariant**: Local project concepts (`priority: 100`) strictly shadow vendor concepts (`priority: 70`) of the same relative ID.
* **Composite BM25 Indexing**: Searches across scopes combine document frequencies while respecting scope precedence.

---

## 2. Registry Client & Package Resolution

The client communicates with the canonical OKF Registry (`https://registry.okf-memory.dev`) or custom private registries specified via `--registry <url>` or `OKF_REGISTRY_URL`:

* **Identifier Resolution**:
  * Scoped Packages: `@org/bundle` (e.g. `@acme/security-rules`)
  * Top-Level Packages: `bundle` (e.g. `nextjs-15`)
  * Version Pinning: `@vX.Y.Z` or `@tag` (e.g. `nextjs-15@1.2.0`)
  * Direct Git URLs: `github.com/org/repo@vX.Y.Z`
* **Knowledge Directory Promotion**: For DMAA repository archives that package memory under `knowledge/`, the extractor automatically stream-filters and promotes `knowledge/` contents directly to `.okf/vendor/<bundle-id>/` without root directory pollution or disk churn.
* **Integrity & Rollback**: Archives are verified against SHA-256 integrity hashes. Downloaded bundles are strictly validated (`okf validate`); any validation failure triggers an immediate, atomic rollback.

---

## 3. Zero-Dependency Lockfile (`okf.lock`)

Dependency versions and cryptographic hashes are locked in `okf.lock` using a deterministic, human-readable YAML structure parsed and emitted without external dependencies:

```yaml
version: 1
bundles:
  - id: nextjs-15
    source: https://registry.okf-memory.dev/tarballs/nextjs-15-1.0.0.tar.gz
    version: 1.0.0
    hash: 5f4dcc3b5aa765d61d8327deb882cf99...
    installed_at: 2026-10-01T10:00:00Z
```

* **Deterministic CLI Operations**:
  * `okf pull [<bundle-id>]`: Resolves, downloads, extracts, validates, and locks the dependency.
  * `okf restore`: Restores and verifies all dependencies in `okf.lock` in clean environments.
  * `okf vendor list`: Lists installed vendor bundles.
  * `okf vendor remove <bundle-id>`: Uninstalls the vendor bundle and cleans `okf.lock`.

---

## 4. Hermetic Cross-Scope Linking (`okf://`)

To prevent broken link warnings in isolated environments, concepts reference external vendor bundles using the `okf://` URI scheme:

* `okf://vendor/<bundle-id>/<concept-id>`
* Shorthand: `@vendor/<bundle-id>/<concept-id>.md`

Bundle validation (`okf validate --strict`) treats `okf://` and `@vendor/` URIs as hermetic external references, ensuring CI pipelines pass without requiring vendor dependencies to be co-located during isolated unit validation.

---

## Related Concepts

- [5-Layer System Architecture](layers.md): Layered separation of concerns
- [Bundle Isolation and Mutation Security Boundaries](security-boundaries.md): Defensive containment and path traversal protection
- [Go Single-Binary CLI & MCP Architecture Decision](tooling-decision.md): Zero-dependency tooling architecture
