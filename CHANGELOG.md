# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Professional repository structure: `SECURITY.md`, `CONTRIBUTING.md`,
  `CODE_OF_CONDUCT.md`, issue forms, pull-request template and changelog.
- Tag-triggered release workflow with GoReleaser, producing checksummed
  archives for Linux, Windows and macOS.
- Centralised build metadata in `internal/buildinfo`, injected by the Makefile
  and GoReleaser; `zlt_xloader version` now reports version, commit and date.
- `make check`, `make ci`, `make cover`, `make snapshot` and `make release`
  targets.
- Prominent educational-use-only / do-not-misuse warning in the README, CLI
  help, LICENSE and `WARNING.md`.
- SSH client transport (`internal/sshclient`) and `shell --ssh`, including
  best-effort dropbear enablement.
- Device profiles for the MTN `S20_1.13.6` and ORG `S20_1.02.7` variants, with
  variant-specific unlock parameters driven by the new profile `unlock` section.
- `tr069-exploit --set-acs` pushes the local ACS URL via `setTR069Config`.
- Additional goformIds: `SET_DEVICE_MODE`, `TZ_SET_USB_STATUS`, `DMZ_SETTING`,
  `setTR069Config` and `REBOOT_DEVICE`.
- Creative README hero layout with locally generated logo, device render and
  CLI preview images (`docs/assets/`, rebuild with `generate_assets.py`).

### Changed

- Canonical Go module is now `github.com/anusara-bemal/zlt_xloader`, matching
  the GitHub repository name.
- License changed from MIT to a proprietary **all rights reserved** license;
  added `NOTICE` with third-party attribution.

### Removed

- Accidental empty `-p/` directories and the duplicate `js_dump/js_dump/`
  copy of the firmware reference dumps.

## [1.0.0] - 2026-11-20

### Added

- Initial toolkit: `scan`, `api-exploit`, `telnet`, `unlock`, `persist`,
  `unhide`, `optimize`, `shell`, `status`, `power`, `sms`, `ussd`, `download`,
  `tr069-exploit`, `uart-at`, `config-decrypt` and `version` commands.
- Editable JSON device profile for the Tozed ZLT S20 (`S20_1.12.7`).
- Dry-run-by-default safety model with interactive confirmation for destructive
  operations.
- Unit tests across the API, crypto, profile, TR-069, UART, netutil and safety
  packages.
- Cross-platform Makefile build matrix and CI workflow.
