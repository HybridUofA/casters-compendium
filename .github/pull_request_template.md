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

Closes #
