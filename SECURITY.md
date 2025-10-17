# Security Policy

Status: hobby project; effectively unmaintained. Best‑effort only, no SLA.

## Supported Versions

- No guaranteed support window. Use the latest commit or the most recent tag.
- The code targets Go `>= 1.25`. Older versions may work but aren't evaluated.

## Report a Vulnerability (preferred)

- Use GitHub's private advisory flow:
  https://github.com/deichbewohner/swiftseer/security/advisories/new
- Please avoid opening a public issue for new vulnerabilities.

If the advisory flow isn't available for you, open a minimal public issue
without exploit details and we'll move it private if possible.

## Scope (what this policy covers)

- This repository and its code only.

## Refresh Token Storage

Swiftseer stores a **refresh token** in an OS-specific config file:

* **Linux:** `~/.config/swiftseer/config.yaml`
* **macOS:** `~/.config/swiftseer/config.yaml`
* **Windows:** `%AppData%\swiftseer\config.yaml`

## Token Considerations

* Treat the refresh token like a **password**.
* Restrict file access (e.g., `chmod 600` on Linux/macOS; user-only ACLs on Windows).
* **Never** commit or share the config file; add it to `.gitignore`.
* Avoid pasting tokens/paths in bug reports or screenshots.
* If you suspect exposure: **revoke/rotate** credentials in Future, delete the
  file, and re-authenticate.
