# Releasing a new version

This library is published as a Go module. New versions are released by pushing a
git tag to GitHub. The Go module proxy picks up the tag automatically.

## Prerequisites

- Push access to `github.com/dash0hq/dash0-api-client-go`
- All changes merged to `main`
- CI passing on `main`

## Steps

1. **Decide the next version number.**
   Follow [semver](https://semver.org/):
   - **patch** (v1.1.1) — bug fixes, doc changes
   - **minor** (v1.2.0) — new features, backward-compatible changes
   - **major** (v2.0.0) — breaking changes (requires a `/v2` module path)

2. **Update `CHANGELOG.md`.**
   Add a section for the new version at the top of the file with a summary of
   changes since the last release. Commit directly to `main` or via a PR:
   ```bash
   git checkout main
   git pull origin main
   # edit CHANGELOG.md
   git add CHANGELOG.md
   git commit -m "docs: add changelog for vX.Y.Z"
   git push origin main
   ```

3. **Create and push the tag.**
   ```bash
   git tag vX.Y.Z
   git push origin vX.Y.Z
   ```

4. **Verify the release.**
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
- There is no automated release workflow; tags are created manually.
- If a tag is pushed for a version that breaks CI, delete the tag and fix the
  issue before re-tagging:
  ```bash
  git tag -d vX.Y.Z
  git push origin --delete vX.Y.Z
  ```
- For a **v2+** major release, the module path in `go.mod` must be updated to
  include `/v2` (or `/vN`) and all internal imports must be updated accordingly.
  See https://go.dev/blog/v2-go-modules for details.
