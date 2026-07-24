#!/usr/bin/env bash

set -euo pipefail

pacman -Syu --noconfirm \
	git \
	go \
	libglvnd \
	libx11 \
	libxcursor \
	libxi \
	libxinerama \
	libxkbcommon \
	libxrandr \
	mesa \
	namcap \
	wayland

useradd --create-home builder
install -d -o builder -g builder /build
cp -a /workspace/packaging/. /build/
install -m644 \
	/build/casters-compendium.desktop \
	/build/casters-compendium.1 \
	/build/copyright \
	/build/arch/
chown -R builder:builder /build

source_fragment="${CASTERS_COMPENDIUM_SOURCE_FRAGMENT:-tag=v0.1.6}"

runuser -u builder -- env \
	HOME=/home/builder \
	CASTERS_COMPENDIUM_SOURCE_FRAGMENT="$source_fragment" \
	bash -lc '
  cd /build/arch
  makepkg --cleanbuild --noconfirm
'

package_path="$(runuser -u builder -- env \
	HOME=/home/builder \
	CASTERS_COMPENDIUM_SOURCE_FRAGMENT="$source_fragment" \
	bash -lc '
  cd /build/arch
  makepkg --packagelist
' | tail -n 1)"

namcap /build/arch/PKGBUILD "$package_path"
install -Dm644 "$package_path" "/workspace/dist/$(basename -- "$package_path")"
