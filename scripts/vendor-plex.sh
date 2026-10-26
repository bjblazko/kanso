#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# Vendors IBM Plex Sans/Mono WOFF2 (Latin-1 subsets) into web/static/fonts.
# Versions and npm integrity hashes are pinned; a mismatch aborts.
set -euo pipefail

SANS_VERSION=1.1.0
SANS_INTEGRITY='sha512-WPgvO6Yfj2w5YbhyAr1tv95RUz4LRJlqN+CmYvBglabXteufP1D1E9BABMde+ZIKdRbFJDoKF5eQzfhpnbgZcQ=='
MONO_VERSION=2.5.0
MONO_INTEGRITY='sha512-STBJIPxPomOYPmBMO7z5TKPJUotAF9u3gAUumTqVgwgrAO+K4FRNh0MlhsoJjKhJKsMbBJR10/bk4inkj/wc1w=='

root=$(cd "$(dirname "$0")/.." && pwd)
out="$root/web/static/fonts"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

fetch() { # package version integrity → extracted dir
  local pkg=$1 version=$2 want=$3 tgz got
  tgz=$(cd "$work" && npm pack --silent "$pkg@$version")
  got="sha512-$(openssl dgst -sha512 -binary "$work/$tgz" | openssl base64 -A)"
  [[ "$got" == "$want" ]] || { echo "integrity mismatch for $pkg@$version" >&2; exit 1; }
  mkdir -p "$work/${pkg##*/}"
  tar xzf "$work/$tgz" -C "$work/${pkg##*/}"
  echo "$work/${pkg##*/}/package"
}

sans=$(fetch @ibm/plex-sans "$SANS_VERSION" "$SANS_INTEGRITY")
mono=$(fetch @ibm/plex-mono "$MONO_VERSION" "$MONO_INTEGRITY")

mkdir -p "$out"
for w in Regular Medium SemiBold; do
  cp "$sans/fonts/split/woff2/IBMPlexSans-$w-Latin1.woff2" "$out/"
done
cp "$mono/fonts/split/woff2/IBMPlexMono-Regular-Latin1.woff2" "$out/"
cp "$sans/LICENSE.txt" "$root/web/static/licenses/ibm-plex.txt"
printf 'IBM Plex Sans %s, IBM Plex Mono %s\n' "$SANS_VERSION" "$MONO_VERSION" > "$out/VERSION"
echo "vendored IBM Plex into $out"
