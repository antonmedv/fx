# Release

1. Bumpversion in [version.go](version.go).
2. Bump version in [snapcraft.yaml](snap/snapcraft.yaml).
3. Bump version in [package.json](npm/package.json).
4. Publish npm package.
5. Create a new release on [GitHub](https://github.com/antonmedv/fx/releases/new).
6. Run [build.mjs](scripts/build.mjs) to upload binaries to the release.
   ```sh
   npx zx scripts/build.js 
   ```
7. Bump version in [install.sh](https://github.com/antonmedv/fx.wtf/blob/master/public/install.sh) and upload it
   to [fx.wtf](https://fx.wtf).
