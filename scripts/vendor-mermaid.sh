#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# Vendors the mermaid ESM build into web/static/vendor/mermaid, WITHOUT the
# ELK layout chunk (elkjs is EPL-2.0, see docs/adr/002-dependency-policy.md)
# and without source maps. Collects the licenses of all bundled packages.
set -euo pipefail

VERSION=12.2.0
INTEGRITY='sha512-w7nr/SFrLO7BaVTs3dQTvYSV6nYSBj0Yy9DjBhsIJtAcHqRzXSq4Pg32ujHZ/AdWsAMNEvvhbm5dt9E2tOiHzw=='

root=$(cd "$(dirname "$0")/.." && pwd)
out="$root/web/static/vendor/mermaid"
licenses="$root/web/static/licenses/mermaid.txt"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

cd "$work"
tgz=$(npm pack --silent "mermaid@$VERSION")
got="sha512-$(openssl dgst -sha512 -binary "$tgz" | openssl base64 -A)"
[[ "$got" == "$INTEGRITY" ]] || { echo "integrity mismatch for mermaid@$VERSION" >&2; exit 1; }
tar xzf "$tgz"

rm -rf "$out"
mkdir -p "$out/chunks/mermaid.esm.min"
cp package/dist/mermaid.esm.min.mjs "$out/"
for f in package/dist/chunks/mermaid.esm.min/*.mjs; do
  case "$(basename "$f")" in elk-*) continue ;; esac
  cp "$f" "$out/chunks/mermaid.esm.min/"
done
if grep -rl "org.eclipse.elk" "$out" >/dev/null; then
  echo "ELK code found outside the elk chunk; refusing to vendor" >&2; exit 1
fi
printf 'mermaid %s (ESM, without ELK layout)\n' "$VERSION" > "$out/VERSION"

# Licenses of mermaid and every package bundled into it (elkjs excluded).
npm init -y >/dev/null
npm install --silent --ignore-scripts --no-audit --no-fund "mermaid@$VERSION"
node - "$licenses" <<'JS'
const fs = require('fs'), path = require('path');
const out = [], seen = new Set();
function visit(dir) {
  for (const name of fs.readdirSync(dir)) {
    if (name.startsWith('.')) continue;
    const full = path.join(dir, name);
    if (name.startsWith('@')) { visit(full); continue; }
    const pkgFile = path.join(full, 'package.json');
    if (!fs.existsSync(pkgFile)) continue;
    const pkg = JSON.parse(fs.readFileSync(pkgFile, 'utf8'));
    const id = `${pkg.name}@${pkg.version}`;
    if (pkg.name === 'elkjs' || pkg.name.startsWith('@types/') || seen.has(id)) continue;
    seen.add(id);
    const file = fs.readdirSync(full).find((f) => /^(licen[cs]e|copying)/i.test(f));
    const license = typeof pkg.license === 'object' ? pkg.license.type : pkg.license;
    const text = file ? fs.readFileSync(path.join(full, file), 'utf8').trim() : `(no license file; package.json declares ${license})`;
    out.push({ id, license, text });
    const nested = path.join(full, 'node_modules');
    if (fs.existsSync(nested)) visit(nested);
  }
}
visit('node_modules');
out.sort((a, b) => a.id.localeCompare(b.id));
const header = 'mermaid and the packages bundled into its ESM build, as shipped by Kanso.\n' +
  'DOMPurify is used under Apache-2.0 (of MPL-2.0 OR Apache-2.0). elkjs (EPL-2.0) is not shipped.\n';
fs.writeFileSync(process.argv[2], header + out.map((p) =>
  `\n${'='.repeat(72)}\n${p.id} — ${p.license}\n${'='.repeat(72)}\n${p.text}\n`).join(''));
console.log(`collected ${out.length} licenses`);
JS
echo "vendored mermaid $VERSION into $out"
