# Release

1. Update [version.go](version.go).
2. Create a new release on [GitHub](https://github.com/antonmedv/fx/releases/new).
3. Run [build.mjs](scripts/build.mjs) to upload binaries to the release.
   ```sh
   npx zx scripts/build.js 
   ```
4. Update [install.sh](install.sh) and upload it to [fx.wtf](https://fx.wtf).
