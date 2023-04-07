# Release

1. Bump version in [version.go](version.go).
2. Bump version in [snapcraft.yaml](snap/snapcraft.yaml).
3. Create a new release on [GitHub](https://github.com/antonmedv/fx/releases/new).
4. Run [build.mjs](scripts/build.mjs) to upload binaries to the release.
   ```sh
   npx zx scripts/build.js 
   ```
5. Bump version [install.sh](install.sh) and upload it to [fx.wtf](https://fx.wtf).
