# Release Process

This monorepo contains two independently versioned CLI tools: `gqlkit` and `gqlkit-sdl`. Each has its own GoReleaser config, GitHub Actions workflow, and tag prefix.

## Tag Conventions

| Tool       | Tag format          | Example             |
|------------|---------------------|---------------------|
| gqlkit     | `gqlkit/v*`         | `gqlkit/v0.2.0`    |
| gqlkit-sdl | `gqlkit-sdl/v*`     | `gqlkit-sdl/v0.1.0`|

## How It Works

1. You push a prefixed tag (e.g., `gqlkit-sdl/v0.1.0`)
2. GitHub Actions matches the tag pattern and triggers the corresponding workflow:
   - `gqlkit/v*` → `.github/workflows/release-gqlkit.yml`
   - `gqlkit-sdl/v*` → `.github/workflows/release-gqlkit-sdl.yml`
3. The workflow strips the prefix to get clean semver (e.g., `gqlkit-sdl/v0.1.0` → `v0.1.0`)
4. It creates a local git tag with just the version (`v0.1.0`) and sets `GORELEASER_CURRENT_TAG` env var
5. GoReleaser picks up the clean semver tag, builds binaries, and creates a GitHub Release

## Under the Hood: Tag Prefix Stripping

GoReleaser (free/OSS) does not support monorepo tag prefixes — it expects plain semver tags like `v0.1.0`. The `monorepo.tag_prefix` config is a GoReleaser Pro feature.

To work around this, each workflow does the following before running GoReleaser:

```yaml
- name: Create semver tag
  run: |
    VERSION="${GITHUB_REF#refs/tags/gqlkit-sdl/}"   # strips prefix → "v0.1.0"
    git tag "$VERSION"                                # creates local tag
    echo "VERSION=$VERSION" >> "$GITHUB_ENV"          # exports for next step

- name: Run GoReleaser
  env:
    GORELEASER_CURRENT_TAG: ${{ env.VERSION }}        # tells GoReleaser which tag to use
```

This means:
- The **prefixed tag** (`gqlkit-sdl/v0.1.0`) is what triggers the workflow and lives on the remote
- The **plain tag** (`v0.1.0`) is only created locally in CI for GoReleaser to parse
- The GitHub Release is created under the **plain tag name** (`v0.1.0`)
- Both tools share the same version namespace on GitHub Releases (they have different asset names so no conflicts)

## Files Involved

```
.github/workflows/
  release-gqlkit.yml          Triggered by gqlkit/v* tags
  release-gqlkit-sdl.yml      Triggered by gqlkit-sdl/v* tags

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
- **checksum**: Generates `checksums.txt` for all archives
- **release**: Publishes to `khanakia/gqlkit` GitHub repo

## Releasing

### gqlkit

```bash
git tag gqlkit/v0.2.0
git push origin gqlkit/v0.2.0
```

### gqlkit-sdl

```bash
git tag gqlkit-sdl/v0.1.0
git push origin gqlkit-sdl/v0.1.0
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

| Tool       | Variable           | Location                  | Default |
|------------|--------------------|---------------------------|---------|
| gqlkit     | `main.version`     | `gqlkit/cmd/cli/root.go`  | `dev`   |
| gqlkit-sdl | `main.version`     | `gqlkit-sdl/main.go`      | `dev`   |

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
git tag -d gqlkit-sdl/v0.1.0
git push origin :refs/tags/gqlkit-sdl/v0.1.0

# Also delete the GitHub release via CLI if it was created
gh release delete v0.1.0 --repo khanakia/gqlkit --yes

# Re-tag and push
git tag gqlkit-sdl/v0.1.0
git push origin gqlkit-sdl/v0.1.0
```

Note: The GitHub Release is created under the **plain version** (`v0.1.0`), not the prefixed tag.

## Monitor

Check workflow runs at: https://github.com/khanakia/gqlkit/actions
