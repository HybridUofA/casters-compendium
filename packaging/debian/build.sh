#!/usr/bin/env bash

set -euo pipefail

version="${1:?usage: build.sh VERSION [OUTPUT_DIRECTORY]}"
output_directory="${2:-dist}"

if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
	echo "version must use x.y.z format" >&2
	exit 2
fi

repository_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
package_root="$(mktemp -d)"
trap 'rm -rf -- "$package_root"' EXIT

output_directory="$(mkdir -p -- "$output_directory" && cd -- "$output_directory" && pwd)"
package_path="$output_directory/casters-compendium_${version}_amd64.deb"

cd -- "$repository_root"
install -d -m755 "$package_root/usr/bin"
go build \
	-tags migrated_fynedo \
	-buildmode=pie \
	-trimpath \
	-ldflags="-s -w" \
	-o "$package_root/usr/bin/casters-compendium" \
	./cmd/gui
chmod 755 "$package_root/usr/bin/casters-compendium"

install -Dm644 \
	packaging/casters-compendium.desktop \
	"$package_root/usr/share/applications/casters-compendium.desktop"
install -Dm644 \
	data/images/MTD-back-ver01.png \
	"$package_root/usr/share/pixmaps/casters-compendium.png"
install -Dm644 \
	README.md \
	"$package_root/usr/share/doc/casters-compendium/README.md"
install -Dm644 \
	packaging/copyright \
	"$package_root/usr/share/doc/casters-compendium/copyright"
install -d -m755 "$package_root/usr/share/man/man1"
gzip -9n -c packaging/casters-compendium.1 > \
	"$package_root/usr/share/man/man1/casters-compendium.1.gz"
chmod 644 "$package_root/usr/share/man/man1/casters-compendium.1.gz"

changelog="casters-compendium (${version}) stable; urgency=medium

  * Package Caster's Compendium ${version} for Debian and Ubuntu.

 -- HybridUofA <hybriduofa@users.noreply.github.com>  Fri, 17 Jul 2026 00:00:00 +0000
"
printf '%s' "$changelog" | gzip -9n > \
	"$package_root/usr/share/doc/casters-compendium/changelog.gz"
chmod 644 "$package_root/usr/share/doc/casters-compendium/changelog.gz"

installed_size="$(du -sk "$package_root/usr" | cut -f1)"
install -d -m755 "$package_root/DEBIAN"
sed \
	-e "s/@VERSION@/$version/g" \
	-e "s/@INSTALLED_SIZE@/$installed_size/g" \
	packaging/debian/control > "$package_root/DEBIAN/control"
chmod 644 "$package_root/DEBIAN/control"

(
	cd -- "$package_root"
	find usr -type f -print0 | sort -z | xargs -0 md5sum > DEBIAN/md5sums
)

dpkg-deb --root-owner-group --build "$package_root" "$package_path"
echo "$package_path"
