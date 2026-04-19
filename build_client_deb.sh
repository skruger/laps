#!/usr/bin/env bash
set -euo pipefail

# build_server_deb.sh - compile laps-server and produce a Debian package that depends on `inn2`
# Usage: ./build_server_deb.sh [version]  (version defaults to git describe or 0.1.0)

PKGNAME=laps-client
ARCH=${ARCH:-amd64}
VERSION=${1:-$(git describe --tags --always 2>/dev/null || echo "0.1.0")}
INSTALL_PATH=/usr/sbin
CONFIG_SRC=default/client.cfg
CONFIG_DEST=/etc/laps/client.cfg

echo "Packaging $PKGNAME version $VERSION for $ARCH"

WORKDIR=$(mktemp -d)
PKGDIR=$WORKDIR/${PKGNAME}_${VERSION}_${ARCH}

mkdir -p "$PKGDIR/DEBIAN"
mkdir -p "$PKGDIR${INSTALL_PATH}"
mkdir -p "$(dirname "$PKGDIR$CONFIG_DEST")"


cp lapsclient.sh "$PKGDIR${INSTALL_PATH}/${PKGNAME}"
chmod 0755 "$PKGDIR${INSTALL_PATH}/${PKGNAME}"

if [ -f "$CONFIG_SRC" ]; then
  echo "Including config $CONFIG_SRC -> $CONFIG_DEST"
  cp "$CONFIG_SRC" "$PKGDIR$CONFIG_DEST"
  chmod 0644 "$PKGDIR$CONFIG_DEST"
else
  echo "Warning: config $CONFIG_SRC not found; creating empty placeholder at $CONFIG_DEST"
  mkdir -p "$(dirname "$PKGDIR$CONFIG_DEST")"
  echo "# laps-client" > "$PKGDIR$CONFIG_DEST"
  chmod 0644 "$PKGDIR$CONFIG_DEST"
fi

mkdir -p "$PKGDIR/lib/systemd/system"
cp init/laps-client.service $PKGDIR/lib/systemd/system/laps-client.service
cp init/laps-client.timer $PKGDIR/lib/systemd/system/laps-client.timer


# Populate control file
MAINTAINER="$(git config user.name 2>/dev/null || echo "packager") <$(git config user.email 2>/dev/null || echo "packager@example.invalid")>"
cat > "$PKGDIR/DEBIAN/control" <<EOF
Package: $PKGNAME
Version: $VERSION
Section: utils
Priority: optional
Architecture: $ARCH
Maintainer: $MAINTAINER
Depends: curl
Description: laps-client - Client for updating DNS records with LAPS server.
EOF

cp package/DEBIAN-client/* "$PKGDIR/DEBIAN/" 2>/dev/null || true
chmod 0755 "$PKGDIR/DEBIAN/postinst" "$PKGDIR/DEBIAN/postrm" "$PKGDIR/DEBIAN/prerm"
chmod 0644 "$PKGDIR/DEBIAN/control"

echo "Building .deb package (you may be prompted for fakeroot if available)..."
if command -v fakeroot >/dev/null 2>&1; then
  fakeroot dpkg-deb --build "$PKGDIR"
else
  dpkg-deb --build "$PKGDIR"
fi

OUT_DEB="${PKGNAME}_${VERSION}_${ARCH}.deb"
mv "${PKGDIR}.deb" "$OUT_DEB" 2>/dev/null || mv "$WORKDIR/${PKGNAME}_${VERSION}_${ARCH}.deb" "$OUT_DEB" 2>/dev/null || true

if [ -f "$OUT_DEB" ]; then
  echo "Package created: $OUT_DEB"
else
  echo "Packaging failed: expected $OUT_DEB to be created" >&2
  ls -la "$WORKDIR"
  exit 1
fi

# Cleanup
rm -rf "$WORKDIR"

echo "Done."

