---
type: Convention
title: "Dual-Memory Agent Architecture & Agent Action Grammar"
description: Two-layer memory model separating normative working memory (Push/AAG) from semantic domain memory (Pull/OKF).
resource: "https://github.com/okf-memory/okf-agent-memory"
tags: [architecture, dmaa, aag, memory-model, instructions, context-efficiency]
generated: { by: agent/cli, at: "2026-09-15T09:42:40Z" }
status: stable
sources:
  - resource: ../../docs/spec/DUAL_MEMORY_AGENT_ARCHITECTURE_RFC.md
    id: dmaa-rfc
    title: RFC Dual-Memory Agent Architecture
    last_modified: 2026-09-15
  - resource: ../../docs/guides/AGENT_INSTRUCTION_BEST_PRACTICES.md
    id: best-practices
    title: Best Practices for Agent Instructions in AGENTS.md
    last_modified: 2026-09-15
---

# Dual-Memory Agent Architecture (DMAA) & Agent Action Grammar (AAG)

The **Dual-Memory Agent Architecture (DMAA)** resolves the context-bloat and attention-drift dilemma in AI coding agents through a 2-layer cognitive model.[^dmaa-rfc]

```mermaid
flowchart TD
    subgraph PUSH["1. Normative Working Memory (Push Layer)"]
        direction TB
        C1["Canonical AGENTS.md (~100 tokens)"]
        C2["Agent Action Grammar (AAG) Syntax"]
        C3["Permanent Invariants: Tone, Formats, Hard Stops"]
    end

    subgraph PULL["2. Semantic Domain Memory (Pull Layer)"]
        direction TB
        O1["OKF v0.2 Bundle (knowledge/)"]
        O2["0 Tokens at baseline prompt"]
        O3["Queried on-demand via okf_search / okf_show"]
    end

    INPUT["User Request"] --> PUSH
    PUSH -->|Enforces Guardrails| AGENT["AI Agent (LLM)"]
    AGENT -->|Selective Retrieval| PULL
    PULL -->|Context Facts| AGENT
    AGENT --> OUTPUT["Deterministic Response"]
```

## Layer 1: Normative Working Memory (Push Layer)
* **Location:** `AGENTS.md` at repository root (symlinked to `CLAUDE.md`, `.cursorrules`, etc.).
* **Language:** **Agent Action Grammar (AAG)** — dense ASCII operators (`=>`, `ASSERT`, `!`, `MUST`) replacing natural language prose (~78% token reduction).[^best-practices]
* **Budget:** Strictly capped at **100–150 tokens**.
* **Responsibility:** Invariant formatting rules, communication style, and bootstrap triggers to search OKF memory.

## Layer 2: Semantic Domain Memory (Pull Layer)
* **Location:** `knowledge/` OKF v0.2 bundle.
* **Footprint:** **0 tokens** in initial system prompt.
* **Responsibility:** Architecture decisions, data models, domain knowledge, and subsystem governance rules retrieved on-demand via `okf_search`.

## Inter-Concept Connections
* Extends the core principles in [principles](principles.md) with compact agent instruction standards.
* Complements the 5-layer architecture in [architecture/layers](../architecture/layers.md).

[^dmaa-rfc]: RFC Dual-Memory Agent Architecture (DMAA)
[^best-practices]: Best Practices for Agent Instructions in AGENTS.md

# Related Concepts
- [Core Memory Principles & Agent Contract](principles.md): Specializes behavioral invariants into two cognitive memory layers
- [5-Layer System Architecture](../architecture/layers.md): Defines Layer 1 push working memory and Layer 2 pull domain memory
