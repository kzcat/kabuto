# Packaging

PKGBUILD (AUR) and winget manifests are generated automatically by GoReleaser on release.
No manual templates are committed here.

- **AUR:** `kabuto-bin` — generated via `.goreleaser.yaml` `aurs:` block
- **winget:** `kzcat.kabuto` — generated via `.goreleaser.yaml` `winget:` block
- **Nix:** `flake.nix` at repo root
