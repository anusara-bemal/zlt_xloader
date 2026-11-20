# Firmware reference dumps

These files are minified JavaScript assets extracted from the web UI of a
Tozed ZLT S20 router running firmware `S20_1.12.7`.

They are **reference material only**. They are not compiled into the
`zlt_xloader` binary and are not used at runtime. They are kept in the
repository so that the API endpoints, `goformId` values, token flow and hidden
UI flags implemented in this tool can be verified against the original
firmware source.

Notes:

- All rights in these assets remain with the original firmware vendor. They are
  included for interoperability research and documentation purposes only.
- The files may contain firmware defaults (for example `admin`/`admin`) and
  carrier identifiers such as the ACS URL. They do **not** contain personal
  device data.
- Do not place captured configuration files, IMEIs, serial numbers or subscriber
  numbers in this directory. See [SECURITY.md](../SECURITY.md) and the
  [license](../LICENSE).
- When adding an asset, record its source firmware version and checksum.
