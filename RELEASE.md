# Release

1. Bump version in [version.go](version.go).
2. Bump version in [snapcraft.yaml](snap/snapcraft.yaml).
3. Bump version in [package.json](npm/package.json).
4. Create a new release on [GitHub](https://github.com/antonmedv/fx/releases/new).
5. Run [build.mjs](scripts/build.mjs) to upload binaries to the release.
   ```sh
   npx zx scripts/build.js 
   ```
6. Bump version in [install.sh](https://github.com/antonmedv/fx.wtf/blob/master/public/install.sh) and upload it
   to [fx.wtf](https://fx.wtf).
