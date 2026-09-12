# AGENTS.md — ktayl-policy-service

Context bridge for any coding agent (Claude Code, Codex, …). Kept **tiny on purpose**: the code is
the source of truth; this states only what the repo *can't* (policy, catches, pointers). No repo
overview/stack/tree here — it's derivable and rots.

## What this repo is
An **application source repo** — you **build + prove** the artifact here; you do **NOT** deploy it.
Deployment config lives in the GitOps repo: **minicloud-gitops/services/ktayl-policy-service/helm/** (GAP wrapper
chart) + **kargo/** (promotion pipeline).

## Source of truth / state
`git log` / `git status` + the product's **GitHub Project board**. No CURRENT-STATE/HANDOFF doc —
verify any handoff against the repo (the repo wins). A local `CLAUDE.md`, if present, is **never committed**.

## Hard rules (the platform enforces these)
- **Build + prove only — Kargo owns promotion.** CI tests/scans/signs (cosign) + SBOMs and dual-pushes
  **Harbor (dev) + ghcr (prod SHA)**. Do **NOT** bump the gitops repo or edit deploy config from here —
  **Kargo** promotes dev→prod via a **CODEOWNERS-gated PR** in `minicloud-gitops`.
- **Deploy config is NOT here** — image tag / replicas / ingress / secrets live in the GitOps repo.
- **Secrets:** CI org-level secrets; app secrets via **ESO→Vault**. Never hardcode/commit a secret.
  `MINICLOUD_CA_CERT` is raw PEM — never base64-decode it.
- **`main` is protected:** PR + **GPG-signed** commits; **Conventional Commits**, **all-lowercase
  subject**, no trailing period, ≤100 chars (commitlint/husky enforced).
- **Smallest safe change** — don't rewrite working architecture, delete tests, or refactor unrelated code.

## Catches
- CI pushes to Harbor over **Tailscale**; a stale **repo-level** `HARBOR_USER`/`HARBOR_PASSWORD`
  shadows the org secret → `docker login` 401 (delete the repo-level ones).
- `gh pr create/merge` has glitched org-wide → fall back to `gh api .../pulls` + `/merge` (REST).

## Start-of-task
`git status` → smallest safe change → run this repo's tests/lint/build → PR (CI + CODEOWNERS do the
rest; **don't touch the gitops repo** — Kargo promotes the new image). Pick a **BMAD delivery path**
by size (A/B/C/E) — a one-line fix needs no PRD. Full model: `minicloud-gitops/.claude/rules/bmad*.md`
+ the docs-site **BMAD Operating Model** page.
