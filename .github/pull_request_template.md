## Summary

Describe what changed and why.

## User impact

Explain the visible behavior, compatibility impact, or maintainer impact.

## Validation

- [ ] `gofmt` reports no unformatted Go files.
- [ ] `go vet -tags migrated_fynedo ./...`
- [ ] `go test ./...`
- [ ] `go test -tags migrated_fynedo ./...`
- [ ] Relevant manual behavior was tested, or this change does not require it.

## Project checks

- [ ] The change is focused and does not include unrelated files.
- [ ] User-facing behavior and release notes are documented when appropriate.
- [ ] New logic has proportional automated test coverage.
- [ ] Third-party artwork, data, or code has documented permission and attribution.
- [ ] AI assistance, if material, follows `AI_STATEMENT.md`.

## Release impact

- [ ] This does not change a packaged release.
- [ ] This changes a packaged release and its version metadata passes
      `bash scripts/check-version-consistency.sh`.
- [ ] Visual or platform-specific changes have explicit manual acceptance steps
      in the release candidate checklist.

Closes #
