# Release Process

This monorepo contains two independently versioned CLI tools: `gqlkit` and `gqlkit-sdl`. Each has its own GoReleaser config, GitHub Actions workflow, and tag prefix.

## Tag Conventions

| Tool       | Tag format          | Example             |
|------------|---------------------|---------------------|
| gqlkit     | `gqlkit/v*`         | `gqlkit/v0.2.0`    |
| gqlkit-sdl | `gqlkit-sdl/v*`     | `gqlkit-sdl/v0.1.0`|

## How It Works

1. You push a tag matching the prefix (e.g., `gqlkit/v0.2.0`)
2. GitHub Actions triggers the corresponding workflow:
   - `gqlkit/v*` → `.github/workflows/release-gqlkit.yml`
   - `gqlkit-sdl/v*` → `.github/workflows/release-gqlkit-sdl.yml`
3. GoReleaser builds binaries for linux/darwin/windows (amd64/arm64)
4. A GitHub Release is created with the archives and checksums

## Files Involved

```
.github/workflows/
  release-gqlkit.yml          Triggered by gqlkit/v* tags
  release-gqlkit-sdl.yml      Triggered by gqlkit-sdl/v* tags

gqlkit/
  .goreleaser.yml             Build config (main: ./cmd, ldflags for version)
  install.sh                  Auto-detect install script

gqlkit-sdl/
  .goreleaser.yml             Build config (ldflags for version)
  install.sh                  Auto-detect install script
```

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
go build -ldflags "-s -w -X main.version=v0.2.0" -o gqlkit ./cmd
./gqlkit version

# gqlkit-sdl
cd gqlkit-sdl
go build -ldflags "-s -w -X main.version=v0.1.0" -o gqlkit-sdl .
./gqlkit-sdl version
```

### Run without building

```bash
cd gqlkit && go run ./cmd version
cd gqlkit-sdl && go run . version
```

## Version Injection

GoReleaser injects the version at build time via ldflags:

| Tool       | Variable path          | Default |
|------------|------------------------|---------|
| gqlkit     | `main.version`         | `dev` |
| gqlkit-sdl | `main.version`           | `dev` |

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
git tag -d gqlkit/v0.2.0
git push origin :refs/tags/gqlkit/v0.2.0

# Also delete the GitHub release via CLI if it was created
gh release delete gqlkit/v0.2.0 --yes

# Re-tag and push
git tag gqlkit/v0.2.0
git push origin gqlkit/v0.2.0
```

## Monitor

Check workflow runs at: https://github.com/khanakia/gqlkit/actions
