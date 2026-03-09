# Releasing xweb

## Overview

Releases are triggered by pushing a `v*` tag to the repository. GitHub Actions will then run tests
across Linux, macOS, and Windows and create a GitHub release.

## Patch or Minor Release (no breaking changes)

1. **Merge the PR** into `main`.

2. **Pull the latest `main` locally:**
   ```bash
   git checkout main
   git pull
   ```

3. **Determine the next version** by checking the most recent tag:
   ```bash
   git tag --sort=-v:refname | head -5
   ```
   Increment:
   - **patch** (`v3.0.x`) — bug fixes and backward-compatible additions
   - **minor** (`v3.x.0`) — new backward-compatible features

4. **Create and push the tag:**
   ```bash
   git tag v3.0.4
   git push origin v3.0.4
   ```

5. **Verify the release workflow** ran successfully at:
   `https://github.com/openziti/xweb/actions`

## Major Release (breaking changes — e.g. v3 → v4)

A major version bump requires updating the Go module path everywhere the old major version appears.

1. **Update `go.mod`** — change the module path:
   ```
   module github.com/openziti/xweb/v4
   ```

2. **Update all internal imports** — replace the old major version suffix in every `.go` file:
   ```bash
   find . -name '*.go' | xargs sed -i 's|github.com/openziti/xweb/v3|github.com/openziti/xweb/v4|g'
   ```

3. **Verify the build still passes:**
   ```bash
   go build ./...
   go test ./...
   ```

4. **Commit the module path changes**, merge the PR into `main`, then pull:
   ```bash
   git checkout main
   git pull
   ```

5. **Create and push the tag:**
   ```bash
   git tag v4.0.0
   git push origin v4.0.0
   ```

6. **Update all downstream consumers** (e.g. `ziti`, `transport`, etc.) — in each repo:
   - Update `go.mod` imports and `go get`:
     ```bash
     find . -name '*.go' | xargs sed -i 's|github.com/openziti/xweb/v3|github.com/openziti/xweb/v4|g'
     go get github.com/openziti/xweb/v4@v4.0.0
     go mod tidy
     ```

## Consuming a New Release in Downstream Modules

Once tagged, downstream Go modules can update to the new version:

```bash
go get github.com/openziti/xweb/v3@v3.0.4
go mod tidy
```
