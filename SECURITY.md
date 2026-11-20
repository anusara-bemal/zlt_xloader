# Security policy

## Supported versions

Only the latest tagged release and the `main` branch receive security fixes.

| Version | Supported |
| --- | --- |
| Latest tagged release | ✅ |
| `main` branch | ✅ |
| Older releases | ❌ |

## Reporting a vulnerability

Please **do not** open a public issue for a security vulnerability.

Use GitHub's private vulnerability reporting on this repository (the
**Security** tab → **Report a vulnerability**). If that is unavailable, contact
the maintainer through the contact details on the
[GitHub profile](https://github.com/anusara-bemal).

Include:

- affected version or commit,
- reproduction steps and a minimal proof of concept,
- impact assessment,
- suggested fix, if you have one.

You should receive an acknowledgement within **7 days** and a status update at
least every **14 days** until the issue is resolved.

## Scope

**In scope:** vulnerabilities in this tool itself — command injection in its own
argument handling, unsafe path handling, credential leakage in logs, insecure
temporary files, malicious profile files, and weaknesses in its local HTTP/TR-069
listeners.

**Out of scope:**

- vulnerabilities in the Tozed firmware itself (report those to the vendor),
- the default `admin/admin` credentials hardcoded in the target device profile
  (these are public firmware defaults, not a credential of this project),
- issues that require you to physically own or be authorised to test a device,
  where the "vulnerability" is the intended device-ownership functionality.

## Safe-harbour note

This project is a device-ownership and security-research toolkit. Good-faith
research on hardware you own, or that you are explicitly authorised in writing
to test, is welcomed. Do not use this tool against devices or networks you do
not own or are not authorised to assess. See the disclaimer in `README.md` and
the terms in `LICENSE`.

## Disclosure

When a fix is released, we will credit the reporter unless they prefer to
remain anonymous, and publish a short advisory describing the impact and the
remediation.
