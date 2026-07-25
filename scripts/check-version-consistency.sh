#!/usr/bin/env bash

set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repository_root"

failures=0

fail() {
	printf 'version consistency: %s\n' "$*" >&2
	failures=$((failures + 1))
}

require_literal() {
	local file="$1"
	local expected="$2"
	local description="$3"

	if ! grep -Fq -- "$expected" "$file"; then
		fail "$file does not contain the expected $description: $expected"
	fi
}

release_version="$(tr -d '[:space:]' < VERSION)"
if [[ ! "$release_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z][0-9A-Za-z.-]*)?$ ]]; then
	fail "VERSION is not a supported release version: $release_version"
fi

app_version="${release_version%%-*}"
arch_version="${release_version//-/_}"
tag="v${release_version}"

require_literal internal/deckbuilder/app/startup.go \
	"applicationVersion = \"$release_version\"" "application version"
require_literal cmd/deckbuilder/FyneApp.toml \
	"Version = \"$app_version\"" "numeric Fyne bundle version"
require_literal .github/workflows/package-desktop.yml \
	"APP_VERSION: $app_version" "numeric package version"
require_literal .github/workflows/package-desktop.yml \
	"RELEASE_VERSION: $release_version" "release version"
require_literal .github/workflows/package-desktop.yml \
	"artifact: casters-compendium_${release_version}_amd64" "Debian artifact name"
require_literal .github/workflows/package-desktop.yml \
	"artifact: casters-compendium-${arch_version}-1-x86_64" "Arch artifact name"
require_literal packaging/arch/PKGBUILD \
	"pkgver=$arch_version" "Arch package version"
require_literal packaging/arch/PKGBUILD \
	"tag=$tag" "Arch source tag"
require_literal packaging/arch/build-in-container.sh \
	"tag=$tag" "Arch container source tag"
require_literal packaging/casters-compendium.1 \
	"Caster's Compendium $release_version" "manual-page version"
require_literal README.md \
	"v$release_version" "documented release version"
require_literal docs/downloads.html \
	"releases/tag/$tag" "website release link"
require_literal docs/index.html \
	"Version $release_version" "website application version"

if ((failures > 0)); then
	printf 'version consistency: found %d problem(s)\n' "$failures" >&2
	exit 1
fi

printf 'version consistency: %s is consistent (bundle %s, Arch %s, tag %s)\n' \
	"$release_version" "$app_version" "$arch_version" "$tag"
