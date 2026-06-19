---
name: rforge-release
description: Commit pending changes, push, wait for CI, tag and publish the next rforge patch release, trigger go.dev indexing, and reinstall the local binary. Use when the user says "release patch", "release next patch", "cut a release", or "ship rforge".
---

# rforge-release

Full release pipeline for ResearchForge patch versions. Runs in order — stops and reports if any step fails.

## Step 1 — Commit and push pending changes

Check for uncommitted changes:

```sh
git status --short
git diff --stat
```

If there are changes: stage relevant files, write a concise commit message summarising what changed, commit, and push to main. If the working tree is clean, skip to Step 2.

Do not use `git add -A` blindly — inspect what's changed and stage intentionally. Do not commit `.env`, secrets, or binaries.

## Step 2 — Verify CI is green

```sh
gh run list --branch main --limit 5 --json status,conclusion,name,headSha
```

- **All passing** — continue.
- **Any failing** — show the failing job names and stop. Do not tag a failing commit.
- **Pending/in-progress** — wait up to 3 minutes, rechecking every 30 seconds. Stop if still pending after 3 minutes and ask the user whether to proceed.

## Step 3 — Compute next patch version

```sh
git describe --tags --abbrev=0   # current latest tag, e.g. v0.1.1
```

Increment the patch segment only: `v0.1.1` → `v0.1.2`. Never bump minor or major here.

List what will be included in the release:

```sh
git log <current-tag>..HEAD --oneline
```

## Step 4 — Tag and push

```sh
git tag vX.Y.Z
git push origin vX.Y.Z
```

## Step 5 — GitHub release

Build release binaries and checksums before creating the release:

```sh
rm -rf dist
make build-release checksums
```

Then attach every release binary plus checksums:

```sh
gh release create vX.Y.Z \
  dist/rforge-* dist/checksums.txt \
  --repo TrebuchetDynamics/research-forge \
  --title "vX.Y.Z — <one-line summary of changes>" \
  --notes "<bullet list of commits since prior tag>"
```

Include the release URL in your output.

## Step 6 — Trigger go.dev indexing

```sh
GOPROXY=https://proxy.golang.org GO111MODULE=on \
  go list -m github.com/TrebuchetDynamics/research-forge@vX.Y.Z
```

Confirm the module version is echoed back.

## Step 7 — Reinstall rforge

```sh
make install
rforge version
```

Confirm the version string matches the new tag.

## Red lines

- Do not tag if CI is failing — stop and report.
- Do not bump minor or major version.
- Do not force-push tags.
- Do not commit secrets, `.env` files, or build artifacts.
- Do not skip the CI check even if the user says "just release it" — ask explicitly if they want to override.
