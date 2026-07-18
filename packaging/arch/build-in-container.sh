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
chown -R builder:builder /build

runuser -u builder -- env HOME=/home/builder bash -lc '
  cd /build/arch
  makepkg --cleanbuild --noconfirm
'

package_path="$(runuser -u builder -- env HOME=/home/builder bash -lc '
  cd /build/arch
  makepkg --packagelist
' | tail -n 1)"

namcap /build/arch/PKGBUILD "$package_path"
install -Dm644 "$package_path" "/workspace/dist/$(basename -- "$package_path")"
