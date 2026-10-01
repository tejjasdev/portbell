# Releasing

Maintainers only. A release is one tag.

1. Make sure `main` is green in CI and the [manual checklist](#manual-checklist) passes.
2. Move the "Unreleased" entries in `CHANGELOG.md` under a new version heading and date.
3. Tag and push:

   ```sh
   git tag -a v0.1.0 -m "v0.1.0"
   git push origin v0.1.0
   ```

4. The `Release` workflow runs GoReleaser: it builds binaries for Windows, macOS and Linux
   (amd64 and arm64), writes `checksums.txt`, and publishes a GitHub release.
5. Check the release page, and run the downloaded binary once.

## Package managers (from v0.2)

Create the repositories `tejjasdev/homebrew-tap` and `tejjasdev/scoop-bucket`, add a token
secret that can push to them, and enable the `brews` and `scoops` blocks in
`.goreleaser.yaml`.

## Manual checklist

Screen bugs hide in specific terminals. Before a release, check the screen in: tmux,
Windows Terminal, macOS Terminal or iTerm2, and the VS Code terminal. In each, resize the
window and make it very small.

## Versioning

[Semantic Versioning](https://semver.org). Before 1.0 a minor version may change behaviour,
always with a changelog note.
