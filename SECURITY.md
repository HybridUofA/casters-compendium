# Security policy

## Supported versions

Caster's Compendium is an early-stage application. Security fixes are provided
for the latest published release only. Users should update to the newest
version before reporting a problem that may already have been corrected.

## Reporting a vulnerability

Do not open a public issue for a suspected vulnerability. Use the repository's
[private security advisory form](https://github.com/HybridUofA/casters-compendium/security/advisories/new).

Include:

- The affected version and operating system.
- Reproduction steps or a minimal proof of concept.
- The expected security impact.
- Any known mitigations.

Please avoid accessing other users' data, disrupting hosted services, or
publishing exploit details before a fix can be evaluated. The maintainer will
acknowledge a complete report when practical and coordinate disclosure based on
severity and available development time.

## Security boundaries

The application downloads public card data and artwork from the
publisher-authorized hosted catalog and verifies published database size and
SHA-256 metadata before installation. Release binaries are built through GitHub
Actions.

Deck files and diagnostic information may contain user-selected names or local
filesystem paths. Review them before posting publicly. The application does not
need credentials for normal use; never place Cloudflare, GitHub, Discord, or
other service credentials in an issue or deck file.

Card-rules corrections, missing artwork, and ordinary application bugs are not
security vulnerabilities and should use the normal issue forms.
