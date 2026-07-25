# Release process

Caster's Compendium separates automated engineering validation from manual
product acceptance. A green CI run means the source builds and its automated
checks pass; it does not mean every packaged interface has been visually
approved.

## Release channels

- Development builds are workflow artifacts produced from active branches or
  `main`. They are not public releases.
- Release candidates use tags such as `v0.1.7-rc.1`. They run the complete
  packaging workflow and are published as GitHub prereleases for installation
  and testing.
- Stable releases use tags such as `v0.1.7`. They may be published only from a
  commit whose packaged release candidate completed the release checklist.
- Hotfixes use tags such as `v0.1.7-hotfix.1`. They are narrowly scoped
  prereleases unless the maintainer explicitly promotes their fixes into a new
  stable version.

## Version source

[`VERSION`](../VERSION) is the authoritative public release version. Some tools
require transformed versions:

- Fyne application bundles use only the numeric `major.minor.patch` portion.
- Debian packages use the complete public version.
- Arch packages replace prerelease hyphens with underscores.
- Git tags add a leading `v`.

Run `scripts/check-version-consistency.sh` after changing release metadata.
Continuous integration runs the same check and rejects metadata drift.

## Candidate-to-stable flow

1. Merge the intended release changes and confirm CI is green.
2. Update `VERSION`, packaging metadata, documentation, and candidate release
   notes together.
3. Run the version-consistency check and create an `-rc.N` tag.
4. Let GitHub build the actual packages users would receive.
5. Install those packages and complete
   [the release checklist](release-checklist.md). Record tester names, platforms,
   package names, and results in the release issue or pull request.
6. Correct failures with a new candidate. Do not reuse or move an existing tag.
7. After acceptance, create the stable tag from the exact approved source
   commit and verify its checksums, SBOM, provenance, website links, and
   announcement.

Manual checklist entries must be completed by a person who actually performed
the test. Automation and AI assistance must not mark visual or platform checks
as passed.

## Hotfix follow-up

When a defect escapes a public release, use the
[hotfix retrospective template](hotfix-retrospective-template.md). The purpose
is to preserve the technical lesson and improve prevention, not to assign
blame.

## Repository controls

The recommended `main` branch ruleset requires:

- A pull request before merging.
- The `Format, vet, test, build, and report coverage` and
  `Vulnerability scan` checks.
- The branch to be current before merging.
- Resolution of review conversations.

Repository rules and protected release environments are GitHub settings, so
their activation remains a deliberate maintainer action. A future protected
`production-release` environment can add manual approval before stable
publication after the candidate process is established.

Planned in-application release visibility and platform-aware installation are
tracked separately in the [updater plan](updater-plan.md).
