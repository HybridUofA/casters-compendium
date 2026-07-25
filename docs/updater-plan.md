# Application update and installation plan

The updater must make releases easier to discover without silently replacing a
working installation or bypassing operating-system package ownership.

## Phase 1: version visibility and changelog

- Show the public version and build identifier on the main menu and diagnostics.
- Add an in-app changelog sourced from bundled, reviewed release notes.
- Keep the changelog available offline.

This phase is low complexity and does not modify the user's installation.

## Phase 2: update discovery

- Read a small HTTPS release manifest from the Caster's Compendium service.
- Compare stable, prerelease, and ignored versions using semantic-version rules.
- Support automatic checks at a respectful interval and an explicit
  **Check for updates** action.
- Display release notes, package type, size, checksum, and a direct download
  action before making changes.
- Allow users to disable automatic checks and decline or postpone a release.

The manifest should be signed or otherwise anchored to trusted release metadata.
Downloaded files must be verified against published SHA-256 checksums.

## Phase 3: platform-aware installation

- Windows: download and launch a dedicated installer after explicit consent.
- macOS: download the correct architecture bundle and guide replacement of the
  application; do not imply notarization until signing is available.
- Debian/Ubuntu and Arch: detect package-managed installations and hand the
  verified native package to the system package manager with user confirmation.
- Portable Linux: download the archive and provide explicit replacement
  instructions or a carefully designed opt-in helper.

The application must not overwrite package-manager-owned files itself. Failed
updates must leave the installed version usable, and restart or privilege
escalation must always be visible to the user.

## Phase 4: installer polish

- Add resumable downloads and clear progress/error states.
- Preserve settings and deck data across upgrades and rollbacks.
- Add end-to-end tests for manifest parsing, version selection, checksum
  rejection, cancellation, and platform dispatch.
- Test actual release-candidate packages on every supported platform.

## Recommended issue split

1. Main-menu version/build label and offline changelog.
2. Release-manifest schema and semantic-version comparison.
3. Update preferences, scheduling, and user interface.
4. Verified download service.
5. Windows installer integration.
6. macOS bundle update guidance.
7. Debian/Ubuntu and Arch package-manager handoff.
8. Portable Linux update flow and recovery testing.
