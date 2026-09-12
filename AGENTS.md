# AGENTS.md — ktayl-policy-service

Context bridge for any coding agent (Claude Code, Codex, …). Kept **tiny on purpose**: the code is
the source of truth; this file states only what the repo *can't* (policy, catches, pointers). No repo
overview/stack/tree here — it's derivable and rots (evidence: it makes agents worse).

## What this repo is
An **application source repo** — you **build + prove** the artifact here; you do **NOT** deploy it.
Deployment config lives in the GitOps repo: **minicloud-gitops/services/ktayl-policy-service/helm/ (GAP wrapper chart + kargo/)**.

## Source of truth / state
`git log` / `git status` + the product's **GitHub Project board**. There is no
CURRENT-STATE/HANDOFF doc — verify any handoff against the repo (the repo wins). A local `CLAUDE.md`,
if present, is **local only — never commit it**.

## Hard rules (the platform enforces these)
- **Build + prove only — Kargo owns promotion.** CI tests, scans, signs (cosign) + SBOMs the image and
  dual-pushes **Harbor (dev) + ghcr (prod SHA)**. Do **NOT** bump the gitops repo or edit deploy
  config from here — **Kargo** promotes dev→prod via a **CODEOWNERS-gated PR** in `minicloud-gitops`.
- **Deploy config is NOT in this repo** — image tag / replicas / ingress / secrets live in the GitOps
  repo (minicloud-gitops/services/ktayl-policy-service/helm/ (GAP wrapper chart + kargo/)). Changing how it *runs* = a PR there, not here.
- **Secrets:** CI uses **org-level** secrets; app secrets flow via **ESO→Vault**. Never hardcode or
  commit a secret. `MINICLOUD_CA_CERT` is **raw PEM** — never base64-decode it.
- **`main` is protected:** PR + **GPG-signed** commits, trunk-based (feature → PR → main),
  **Conventional Commits**. Don't rewrite history / force-push.
- **Smallest safe change** — don't rewrite working architecture, delete tests, or refactor unrelated code.

## Catches (what configs don't tell you)
- CI pushes to Harbor over **Tailscale**; a stale **repo-level** `HARBOR_USER`/`HARBOR_PASSWORD`
  **shadows** the org secret → `docker login` 401. Fix = delete the repo-level ones (`gh secret list --repo`).
- `gh pr create/merge` has glitched across this org (spurious GraphQL "sha blank"/perms) → fall back
  to `gh api .../pulls` + `/merge` (REST).
- If this repo has a `website/`, it auto-deploys to GitHub Pages on push to `main`.

## Start-of-task
`git status` → smallest safe change → run this repo's **tests / lint / build** → PR (CI + CODEOWNERS
do the rest; **don't touch the gitops repo** — Kargo promotes the new image). Pick a **BMAD delivery
path** by size (A small / B feature / C new-product / E hotfix) — a one-line fix needs no PRD. Full
model: `minicloud-gitops/.claude/rules/bmad*.md` + the docs-site **BMAD Operating Model** page.
