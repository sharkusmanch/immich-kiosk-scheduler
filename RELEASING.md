# Release Process

This project uses [Semantic Versioning](https://semver.org/) and automated releases via GitHub Actions and GoReleaser.

## Version Format

```
vMAJOR.MINOR.PATCH
```

- **MAJOR**: Breaking changes (config format changes, removed features)
- **MINOR**: New features (new config options, new endpoints)
- **PATCH**: Bug fixes, documentation updates

## Creating a Release

1. **Ensure main is stable**
   ```bash
   git checkout main
   git pull
   go test ./...
   ```

2. **Create and push a tag**
   ```bash
   # For a new feature release
   git tag v1.1.0
   git push origin v1.1.0

   # For a bug fix release
   git tag v1.0.1
   git push origin v1.0.1
   ```

3. **Monitor the release**
   - Go to [Actions](https://github.com/sharkusmanch/immich-kiosk-scheduler/actions)
   - Watch the "Release" workflow
   - Once complete, check [Releases](https://github.com/sharkusmanch/immich-kiosk-scheduler/releases)

## What Gets Released

### Binaries (via GoReleaser)
- `immich-kiosk-scheduler_VERSION_linux_amd64.tar.gz`
- `immich-kiosk-scheduler_VERSION_linux_arm64.tar.gz`
- `immich-kiosk-scheduler_VERSION_darwin_amd64.tar.gz`
- `immich-kiosk-scheduler_VERSION_darwin_arm64.tar.gz`
- `immich-kiosk-scheduler_VERSION_windows_amd64.zip`
- `checksums.txt`

### Docker Images (GHCR)
Multi-arch images (amd64 + arm64) are published to:
- `ghcr.io/sharkusmanch/immich-kiosk-scheduler:vX.Y.Z` (exact version)
- `ghcr.io/sharkusmanch/immich-kiosk-scheduler:vX.Y` (minor version)
- `ghcr.io/sharkusmanch/immich-kiosk-scheduler:vX` (major version)
- `ghcr.io/sharkusmanch/immich-kiosk-scheduler:latest`

## Pre-release Versions

For testing releases before making them official:
```bash
git tag v1.1.0-rc1
git push origin v1.1.0-rc1
```

Pre-release tags (`-rc`, `-beta`, `-alpha`) are automatically marked as pre-releases on GitHub.

## Recommended Workflow

1. Develop features on feature branches
2. Create PRs to main
3. CI runs on all PRs (lint, test, build)
4. Merge to main after review
5. When ready for release, tag main with the next version
6. Release workflow builds and publishes everything

## Docker Image Tags for Kubernetes

For production, pin to a specific version:
```yaml
image: ghcr.io/sharkusmanch/immich-kiosk-scheduler:v1.0.0
```

For auto-updates within a minor version (receives patch fixes):
```yaml
image: ghcr.io/sharkusmanch/immich-kiosk-scheduler:v1.0
```

For auto-updates within a major version (receives new features):
```yaml
image: ghcr.io/sharkusmanch/immich-kiosk-scheduler:v1
```
