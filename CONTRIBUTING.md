# Contributing

Thanks for your interest in improving `zlt-unlocker`. Please read this file and
the disclaimer in [README.md](README.md) before you start.

## Ground rules

- Only work with devices you own or are explicitly authorised to test.
- Never commit captured configuration data, IMEIs, serial numbers, phone
  numbers, or other personal identifiers.
- Keep device-specific constants (endpoints, `goformId` values, AT commands,
  keys) in `profiles/*.json`, not in Go source.
- Mutating behaviour must remain dry-run by default. Destructive operations
  must require explicit confirmation (`--yes` or interactive prompt).
- Add unit tests for new parsers, RPC builders and decryption routines.

## Development setup

Requirements: Go 1.22 or newer, `make`, and `gofmt`.

```bash
git clone https://github.com/anusara-bemal/zlt_xloader.git
cd zlt_xloader
go mod download
make check        # gofmt check + go vet + go test ./...
make build        # host binary
```

Useful targets: `make test`, `make cover`, `make fmt`, `make vet`,
`make build-all`, `make ci`, `make snapshot` (GoReleaser, local only).

## Branching and commits

- Branch from `main`: `fix/<short-description>` or `feat/<short-description>`.
- Write imperative commit subjects, e.g. `Add PLMN unlock payload to profile`.
- Use conventional prefixes where practical: `feat:`, `fix:`, `docs:`, `test:`,
  `chore:`, `ci:`. They are used to build the changelog.

## Pull requests

1. Keep each PR focused on one change.
2. Fill in the pull-request template and link the related issue.
3. Make sure `make check` passes locally — CI runs the same checks plus a full
   cross-compile matrix.
4. Update `README.md` and `CHANGELOG.md` when behaviour or usage changes.
5. Rebase on `main` before requesting review; avoid merge commits.

## Adding a device profile

1. Copy `profiles/zlt_s20_1.12.7.json` to a new file named after the device and
   firmware version.
2. Populate endpoints, login defaults, fingerprint keys and payloads from your
   own packet capture. Anonymise anything device-specific.
3. Verify with `./zlt-unlocker --profile ./profiles/<new>.json --target <ip> scan`
   in dry-run mode.
4. Document the firmware version and provenance in the profile `name` field.

## Reporting bugs and requesting features

Use the issue templates. For vulnerabilities, follow [SECURITY.md](SECURITY.md)
and report privately.

## Licensing of contributions

This is a proprietary, all-rights-reserved project. By submitting a pull
request you confirm that you are authorised to contribute the changes and that
the copyright holder may use, modify and redistribute them under the terms of
the project's [LICENSE](LICENSE). If you cannot agree to this, please do not
submit the contribution.
