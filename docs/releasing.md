# Release Process

This monorepo contains two independently versioned CLI tools: `gqlkit` and `gqlkit-sdl`. Each has its own GoReleaser config, GitHub Actions workflow, and tag prefix.

## Tag Conventions

| Tool       | Tag format      | Example              | GitHub Release tag   | Release title        |
|------------|-----------------|----------------------|----------------------|----------------------|
| gqlkit     | `gqlkit@v*`     | `gqlkit@v0.2.0`     | `gqlkit@v0.2.0`     | gqlkit v0.2.0       |
| gqlkit-sdl | `gqlkit-sdl@v*` | `gqlkit-sdl@v0.1.0` | `gqlkit-sdl@v0.1.0` | gqlkit-sdl v0.1.0   |

Both tools can use the same version number (e.g., both at `v0.1.0`) without conflicts because each GitHub Release is tied to its own prefixed tag.

## How It Works — Step by Step

Taking `gqlkit-sdl@v0.1.0` as an example:

### 1. You push a prefixed tag

```bash
git tag gqlkit-sdl@v0.1.0
git push origin gqlkit-sdl@v0.1.0
```

### 2. GitHub Actions matches the tag pattern

The tag `gqlkit-sdl@v0.1.0` matches `gqlkit-sdl@v*` in `.github/workflows/release-gqlkit-sdl.yml`, which triggers the workflow.

### 3. The workflow extracts the version and creates a local semver tag

```yaml
- name: Extract version and create local semver tag
  id: version
  run: |
    FULL_TAG="${GITHUB_REF#refs/tags/}"          # "gqlkit-sdl@v0.1.0"
    VERSION="${FULL_TAG#gqlkit-sdl@}"             # "v0.1.0"
    git tag -f "$VERSION"                         # create local tag pointing to HEAD
    echo "version=$VERSION" >> "$GITHUB_OUTPUT"   # pass to later steps
    echo "full_tag=$FULL_TAG" >> "$GITHUB_OUTPUT"
```

**Why this is needed:** GoReleaser (free/OSS) only accepts plain semver tags like `v0.1.0`. It does not support monorepo prefixed tags — that's a GoReleaser Pro feature (`monorepo.tag_prefix`). So we create a local `v0.1.0` tag for GoReleaser to find.

**Why `git tag -f`:** Both tools might release the same version number (e.g., both `v0.1.0`). If gqlkit-sdl already created a local `v0.1.0` tag in a previous run, `git tag` would fail. The `-f` flag force-creates the tag, overwriting any existing one.

**Why `$GITHUB_OUTPUT`:** Unlike `$GITHUB_ENV` (which sets env vars for all steps), `$GITHUB_OUTPUT` sets named outputs scoped to this step, accessible via `${{ steps.version.outputs.version }}`. We use this to pass values to later steps cleanly.

### 4. GoReleaser builds binaries (but does NOT publish)

```yaml
- name: Build with GoReleaser
  uses: goreleaser/goreleaser-action@v6
  with:
    args: release --clean --skip=publish
    workdir: gqlkit-sdl
  env:
    GORELEASER_CURRENT_TAG: ${{ steps.version.outputs.version }}
```

**`--skip=publish`:** GoReleaser only builds and archives — it does NOT create a GitHub Release. If we let GoReleaser publish, it would create the release under the plain `v0.1.0` tag, and both tools would collide on the same tag.

**`GORELEASER_CURRENT_TAG`:** Tells GoReleaser to use `v0.1.0` as the current tag. Without this, GoReleaser auto-detects tags and would find the prefixed `gqlkit-sdl@v0.1.0` which it can't parse as semver.

**`workdir: gqlkit-sdl`:** Runs GoReleaser from inside the module directory where `.goreleaser.yml` lives.

**What GoReleaser does here:**
- Reads `.goreleaser.yml` config
- Cross-compiles for linux/darwin/windows (amd64/arm64) with `CGO_ENABLED=0`
- Injects version via ldflags: `-X main.version=0.1.0`
- Creates archives in `gqlkit-sdl/dist/`: `.tar.gz` for linux/macOS, `.zip` for Windows
- Generates `gqlkit-sdl_checksums.txt`

### 5. `gh release create` publishes under the prefixed tag

```yaml
- name: Create GitHub Release
  run: |
    TAG="${{ steps.version.outputs.full_tag }}"       # "gqlkit-sdl@v0.1.0"
    VERSION="${{ steps.version.outputs.version }}"     # "v0.1.0"
    gh release create "$TAG" \
      --repo khanakia/gqlkit \
      --title "gqlkit-sdl $VERSION" \
      --generate-notes \
      gqlkit-sdl/dist/*.tar.gz gqlkit-sdl/dist/*.zip gqlkit-sdl/dist/gqlkit-sdl_checksums.txt
```

**Why `gh release create` instead of GoReleaser publish:** This is the key to making monorepo releases work. `gh release create` can create a release under ANY tag name — including `gqlkit-sdl@v0.1.0`. GoReleaser would only create it under the plain semver tag `v0.1.0`, causing collisions.

**Result:** Each tool gets its own GitHub Release page tied to its own prefixed tag. No conflicts.

## Problems Solved by This Approach

| Problem | Solution |
|---------|----------|
| GoReleaser rejects prefixed tags as invalid semver | Create local plain semver tag + `GORELEASER_CURRENT_TAG` env var |
| Both tools releasing same version would collide on `v0.1.0` | GoReleaser only builds (`--skip=publish`), `gh release create` publishes under prefixed tag |
| `git tag v0.1.0` fails if tag already exists from other tool | Use `git tag -f` to force-create |
| `checksums.txt` name collision between tools | Each tool uses `{project}_checksums.txt` naming |

## Files Involved

```
.github/workflows/
  release-gqlkit.yml          Triggered by gqlkit@v* tags
  release-gqlkit-sdl.yml      Triggered by gqlkit-sdl@v* tags

gqlkit/
  .goreleaser.yml             Build config (main: ./cmd/cli, ldflags for version)
  install.sh                  Auto-detect install script

gqlkit-sdl/
  .goreleaser.yml             Build config (ldflags for version)
  install.sh                  Auto-detect install script
```

## GoReleaser Config

Each `.goreleaser.yml` configures:
- **builds**: Cross-compilation targets (linux/darwin/windows, amd64/arm64), CGO disabled, ldflags for version injection
- **archives**: `.tar.gz` for linux/macOS, `.zip` for Windows. Filenames exclude the version so `releases/latest/download/` URLs stay stable
- **checksum**: Named `{project}_checksums.txt` to avoid collisions
- **release**: Points to `khanakia/gqlkit` repo with a project-specific `name_template`

## Releasing

### gqlkit

```bash
git tag gqlkit@v0.2.0
git push origin gqlkit@v0.2.0
```

### gqlkit-sdl

```bash
git tag gqlkit-sdl@v0.1.0
git push origin gqlkit-sdl@v0.1.0
```

## Testing Locally

### GoReleaser dry run (builds binaries, skips publishing)

```bash
cd gqlkit
goreleaser release --snapshot --clean
ls dist/

cd gqlkit-sdl
goreleaser release --snapshot --clean
ls dist/
```

### Build with version ldflags

```bash
# gqlkit
cd gqlkit
go build -ldflags "-s -w -X main.version=v0.2.0" -o gqlkit ./cmd/cli
./gqlkit version

# gqlkit-sdl
cd gqlkit-sdl
go build -ldflags "-s -w -X main.version=v0.1.0" -o gqlkit-sdl .
./gqlkit-sdl version
```

### Run without building

```bash
cd gqlkit && go run ./cmd/cli version
cd gqlkit-sdl && go run . version
```

## Version Injection

GoReleaser injects the version at build time via ldflags (`-X main.version={{.Version}}`):

| Tool       | Variable       | Location                 | Default |
|------------|----------------|--------------------------|---------|
| gqlkit     | `main.version` | `gqlkit/cmd/cli/root.go` | `dev`   |
| gqlkit-sdl | `main.version` | `gqlkit-sdl/main.go`     | `dev`   |

## Archive Naming

Archives do NOT include the version in the filename, so `releases/latest/download/` URLs work:

```
gqlkit_darwin_arm64.tar.gz
gqlkit_linux_amd64.tar.gz
gqlkit-sdl_darwin_arm64.tar.gz
gqlkit-sdl_linux_amd64.tar.gz
```

## User Install

```bash
# gqlkit
curl -sL https://raw.githubusercontent.com/khanakia/gqlkit/main/gqlkit/install.sh | sh

# gqlkit-sdl
curl -sL https://raw.githubusercontent.com/khanakia/gqlkit/main/gqlkit-sdl/install.sh | sh
```

## Deleting and Re-tagging (if needed)

```bash
# Delete tag locally and remotely
git tag -d gqlkit-sdl@v0.1.0
git push origin :refs/tags/gqlkit-sdl@v0.1.0

# Delete the GitHub release (uses the prefixed tag name)
gh release delete gqlkit-sdl@v0.1.0 --repo khanakia/gqlkit --yes

# Re-tag and push
git tag gqlkit-sdl@v0.1.0
git push origin gqlkit-sdl@v0.1.0
```

## Monitor

Check workflow runs at: https://github.com/khanakia/gqlkit/actions
