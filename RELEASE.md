# Release

1. Bump version in [version.go](version.go).
2. Bump version in [snapcraft.yaml](snap/snapcraft.yaml).
3. Bump version in [package.json](npm/package.json).
4. Commit changes.
5. Publish npm package.
6. Trigger [GitHub Actions](https://github.com/antonmedv/fx/actions) (brew, snap, docker).
7. Create a new release on [GitHub](https://github.com/antonmedv/fx/releases/new).
8. Run [build.mjs](scripts/build.mjs) to upload binaries to the release.
   ```sh
   npx zx scripts/build.mjs
   ```
9. Bump version in [install.sh](https://github.com/antonmedv/fx.wtf/blob/master/public/install.sh) and upload it
   to [fx.wtf](https://fx.wtf).
