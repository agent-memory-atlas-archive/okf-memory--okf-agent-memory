---
type: Process
title: "Release Procedure & Distribution Runbook"
description: "Canonical procedure for preparing releases, quality gates, file inventory, signed tagging, and the human push boundary."
tags: [release, playbook, runbook, distribution, workflow, tagging]
generated: { by: agent/mcp, at: "2026-09-23T19:20:06Z" }
status: stable
sources:
  - resource: ../../docs/project/playbooks/RELEASE_PLAYBOOK.md
    id: release-playbook
    title: OKF Agent Memory Release Playbook
    last_modified: 2026-09-18
---

# Release Procedure & Distribution Runbook

This operational process governs the end-to-end workflow for preparing, verifying, and packaging releases of `okf-agent-memory`. It codifies `docs/project/playbooks/RELEASE_PLAYBOOK.md` directly into the agent's persistent memory.

---

## 1. Agent Invariant: Push Authority Boundary

> [!IMPORTANT]
> **Human-in-the-Loop Push Constraint:**
> AI agents MUST prepare all release assets, run verification checks, commit on `develop`, fast-forward `main`, and create the signed git tag.
> **AI agents MUST NEVER execute `git push` for releases.** Pushing to remote repositories (`origin develop`, `origin main`, `origin <tag>`) is strictly reserved for the human user.

---

## 2. Release Trigger: "Release" Action Pipeline

Whenever the user prompts with "release", "lass uns releasen", or specifies a target version `vX.Y.Z`, the agent must immediately execute the following steps in sequence:

### Phase 1: Quality & Verification Gates
1. Run formatting: `make fmt`
2. Sync embedded bootstrap assets (if `.agents/skills` changed): `make sync-assets`
3. Run comprehensive test suite: `make check` (vet, lint, unit tests, integration tests)
4. Validate knowledge bundle: `okf validate knowledge --strict --drift` (must report 0 errors, 0 warnings, 0 broken links)

### Phase 2: Master File Inventory & Release Documentation
Update or create all files in the canonical release inventory:
- [ ] `docs/releases/v<X.Y.Z>.md`: Comprehensive release notes with sections:
  - Overview & Highlights
  - Feature details & Breaking changes
  - Storage / Conformance hardening
  - Developer experience & CI updates
  - Community & Special Thanks (crediting human contributors by handle and issue/PR; never credit dedicated AI agent accounts or bots)
  - Pre-built Binaries link
  - `### Full Changelog`: Link comparison `https://github.com/okf-memory/okf-agent-memory/compare/v<PREV>...v<CURR>`
- [ ] `docs/releases/README.md`: Add row with version, date, and core highlights
- [ ] `knowledge/log.md`: Record release entry under today's date
- [ ] `knowledge/roadmap/milestones.md`: Update milestone deliverables and current phase status
- [ ] `CONTRIBUTORS.md`: Credit human contributors only (humans who author PRs using AI agents are credited normally; never credit dedicated bot/agent accounts like Jules or Claude Bot)
- [ ] `README.md`: Update supported version range (e.g. `(v0.1.0 – v0.4.3)`)

### Phase 3: Git Flow & Signed Tagging
1. Stage and commit all release preparation files on `develop`:
   ```bash
   git add docs/releases/ README.md knowledge/ CONTRIBUTORS.md
   git commit -m "chore(release): prepare release notes and knowledge for v<X.Y.Z>" # Signed with SSH key
   ```
2. Fast-forward merge `develop` into `main`:
   ```bash
   git checkout main
   git merge --ff-only develop
   ```
3. Create annotated, cryptographically signed tag on `main`:
   ```bash
   git tag -s v<X.Y.Z> -m "Release v<X.Y.Z>: <Title>"
   ```
4. Return to `develop`:
   ```bash
   git checkout develop
   ```

### Phase 4: Handover to User
Report completion to the user and supply the exact, final push command for them to execute:
```bash
git push origin develop && git push origin main && git push origin v<X.Y.Z>
```

---

# Related Concepts
- [Contributor Guidelines & PR Standards](contributing.md): GitFlow branch strategy and quality gates
- [Knowledge Lifecycle & Review Workflow](lifecycle.md): Read-Before-Write loop and audit logging
- [Project Roadmap & Development Milestones](../roadmap/milestones.md): Milestone tracking and phase progression
