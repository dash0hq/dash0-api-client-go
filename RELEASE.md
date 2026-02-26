# Releasing a new version

This library is published as a Go module. New versions are released by pushing a
git tag to GitHub. The Go module proxy picks up the tag automatically.

## Prerequisites

- Push access to `github.com/dash0hq/dash0-api-client-go`
- All changes merged to `main`
- CI passing on `main`

## Automated Release (Recommended)

The release process is automated via two GitHub Actions workflows:

1. **Prepare Release** — updates `version.go`, generates the changelog entry, commits, tags, and pushes.
2. **Release** — triggered by the tag push; creates a GitHub Release with generated release notes and triggers Go module proxy indexing.

### Steps

1. Go to **Actions → Prepare Release** in the GitHub repository.
2. Click **Run workflow**.
3. Enter the version number **without** the `v` prefix (e.g., `1.6.0`).
4. Click **Run workflow** to start.

The workflow will:
- Validate the version format and check the tag doesn't already exist
- Update `version.go` with the new version
- Generate a changelog section from conventional commits using git-cliff
- Prepend the new section to `CHANGELOG.md`
- Run `go build` and `go test` to verify nothing is broken
- Commit (`chore: prepare release vX.Y.Z`), create a lightweight tag, and push

Once the tag is pushed, the **Release** workflow automatically:
- Creates a GitHub Release with the generated release notes
- Triggers Go module proxy indexing

### Version Guidelines

Follow [semver](https://semver.org/):
- **patch** (v1.1.1) — bug fixes, doc changes
- **minor** (v1.2.0) — new features, backward-compatible changes
- **major** (v2.0.0) — breaking changes (requires a `/v2` module path)

## Manual Release (Fallback)

If the automated workflow is unavailable, you can release manually:

1. **Update `version.go`** with the new version string.

2. **Update `CHANGELOG.md`.**
   Add a section for the new version at the top of the file with a summary of
   changes since the last release.

3. **Commit and push.**
   ```bash
   git checkout main
   git pull origin main
   git add version.go CHANGELOG.md
   git commit -m "chore: prepare release vX.Y.Z"
   git push origin main
   ```

4. **Create and push the tag.**
   ```bash
   git tag vX.Y.Z
   git push origin vX.Y.Z
   ```

5. **Verify the release.**
   After a few minutes, confirm the new version appears on the Go module proxy:
   ```
   https://pkg.go.dev/github.com/dash0hq/dash0-api-client-go@vX.Y.Z
   ```
   You can also force the proxy to fetch it:
   ```bash
   GOPROXY=https://proxy.golang.org go list -m github.com/dash0hq/dash0-api-client-go@vX.Y.Z
   ```

## Notes

- Tags are lightweight (not annotated). This is sufficient for Go module versioning.
- If a tag is pushed for a version that breaks CI, delete the tag and fix the
  issue before re-tagging:
  ```bash
  git tag -d vX.Y.Z
  git push origin --delete vX.Y.Z
  ```
- For a **v2+** major release, the module path in `go.mod` must be updated to
  include `/v2` (or `/vN`) and all internal imports must be updated accordingly.
  See https://go.dev/blog/v2-go-modules for details.
- If `main` has branch protection rules requiring PRs, the bot push in the
  prepare-release workflow will fail. Workaround: use a GitHub App token or PAT
  with bypass permissions, or modify the workflow to open a PR instead of pushing
  directly.
