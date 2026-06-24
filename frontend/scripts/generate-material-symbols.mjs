// Downloads a Material Symbols Outlined subset containing only the icons the app
// renders, via the Google Fonts `icon_names` API. The returned font is still a
// variable font (FILL 0..1, wght 300..400, opsz/GRAD pinned) so the filled/
// outline/light states keep working, but it ships only our glyphs — the full
// font is ~3.8 MB, the subset is a few dozen KB.
// Regenerate with: pnpm generate:material-symbols
//
// Icon names are discovered by scanning the source (templates + inline
// component templates + TS), so there is no list to keep in sync. The Google
// Fonts API ignores unknown names, so occasional over-matches are harmless.
import { mkdirSync, readdirSync, readFileSync, statSync, writeFileSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const srcDir = join(frontendRoot, 'src');

function walkSource(dir) {
  const files = [];
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry);
    if (statSync(full).isDirectory()) files.push(...walkSource(full));
    else if (entry.endsWith('.html') || entry.endsWith('.ts')) files.push(full);
  }
  return files;
}

// Material Symbols ligatures are lowercase snake_case (lsicon names use hyphens,
// so they never match these patterns).
const ICON_NAME = '[a-z][a-z0-9_]*';
// `<mat-icon ...>...</mat-icon>` blocks (covers .html and inline .ts templates).
// The closing tag tolerates whitespace before `>` (the formatter wraps long tags).
const MAT_ICON_BLOCK_RE = /<mat-icon\b[^>]*>([\s\S]*?)<\/mat-icon\s*>/g;
// A bare ligature, e.g. `<mat-icon>delete</mat-icon>`.
const LITERAL_RE = new RegExp(`^${ICON_NAME}$`);
// Quoted names inside interpolations, e.g. `{{ x ? 'visibility_off' : 'visibility' }}`.
const QUOTED_RE = new RegExp(`['"](${ICON_NAME})['"]`, 'g');
// TS: `icon: 'folder'` object properties.
const ICON_PROP_RE = new RegExp(`\\bicon\\s*:\\s*['"](${ICON_NAME})['"]`, 'g');
// TS: `*_ICON* = 'draft'` constants (e.g. FILE_ICON_FALLBACK, FOLDER_ICON).
const ICON_CONST_RE = new RegExp(`_ICON\\w*\\s*=\\s*['"](${ICON_NAME})['"]`, 'g');

const icons = new Set();
for (const file of walkSource(srcDir)) {
  const source = readFileSync(file, 'utf8');
  for (const [, inner] of source.matchAll(MAT_ICON_BLOCK_RE)) {
    const trimmed = inner.trim();
    if (LITERAL_RE.test(trimmed)) icons.add(trimmed);
    else for (const [, name] of trimmed.matchAll(QUOTED_RE)) icons.add(name);
  }
  if (file.endsWith('.ts')) {
    for (const [, name] of source.matchAll(ICON_PROP_RE)) icons.add(name);
    for (const [, name] of source.matchAll(ICON_CONST_RE)) icons.add(name);
  }
}
const names = [...icons].sort();
if (names.length === 0) throw new Error('No Material Symbols icons discovered in source.');

// Axis order must be lowercase-alpha then uppercase-alpha: opsz, wght, FILL, GRAD.
const family = 'Material+Symbols+Outlined:opsz,wght,FILL,GRAD@24,300..400,0..1,0';
const cssUrl =
  `https://fonts.googleapis.com/css2?family=${family}` +
  `&icon_names=${names.join(',')}&display=block`;

// A modern UA makes Google Fonts serve woff2 (with variations) rather than ttf.
const userAgent =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 ' +
  '(KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

const cssResponse = await fetch(cssUrl, { headers: { 'User-Agent': userAgent } });
if (!cssResponse.ok) {
  throw new Error(
    `Google Fonts CSS request failed (${cssResponse.status}): ${await cssResponse.text()}`,
  );
}
const css = await cssResponse.text();
const urlMatch = css.match(/url\((https:\/\/[^)]+)\)\s*format\('woff2'\)/);
if (!urlMatch) throw new Error(`No woff2 URL found in Google Fonts CSS:\n${css}`);

const fontResponse = await fetch(urlMatch[1]);
if (!fontResponse.ok) throw new Error(`Font download failed (${fontResponse.status})`);
const buffer = Buffer.from(await fontResponse.arrayBuffer());

const outDir = join(frontendRoot, 'public', 'fonts');
mkdirSync(outDir, { recursive: true });
const outPath = join(outDir, 'material-symbols-outlined-subset.woff2');
writeFileSync(outPath, buffer);

console.log(`Subset ${names.length} icons: ${names.join(', ')}`);
console.log(`Wrote ${(buffer.length / 1024).toFixed(1)} KB to ${outPath}`);
