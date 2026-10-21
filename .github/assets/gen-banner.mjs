/**
 * Generates the README banners from logo.svg, a copy of the window's own logo
 * in ui/src/assets. The pommel reads on white and on GitHub's dark background
 * alike, so both banners embed it unchanged.
 *
 *   pommelporter-banner.svg/.png       light banner: logo, name and claim
 *   pommelporter-banner-dark.svg/.png  dark banner
 *
 * Text is converted to paths with opentype.js so the SVG needs no font: Bree
 * Serif for the name, Lato for the claim.
 *
 * Deps (global): opentype.js, @resvg/resvg-js. Fonts are fetched to the OS
 * temp dir. Run: node .github/assets/gen-banner.mjs
 */
import { readFileSync, writeFileSync, existsSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { tmpdir } from "node:os";
import { createRequire } from "node:module";
import { execSync } from "node:child_process";

const require = createRequire(import.meta.url);
const groot = execSync("npm root -g").toString().trim();
const opentype = require(`${groot}/opentype.js`);
const { Resvg } = require(`${groot}/@resvg/resvg-js`);

const __dir = dirname(fileURLToPath(import.meta.url));

const NAME = "PommelPorter";
const CLAIM = "Carries your app in. No tip required.";
const W = 1600, H = 500;
const LOGO = 360;
const nameSize = 132, claimSize = 44, gap = 70, lineGap = 8;

const THEMES = [
  { suffix: "", bg: "#ffffff", name: "#1f2328", claim: "#5a5d5e" },
  { suffix: "-dark", bg: "#0d1117", name: "#e6edf3", claim: "#9aa4ad" },
];

async function loadFont(file, url) {
  const path = join(tmpdir(), file);
  if (!existsSync(path)) {
    const res = await fetch(url);
    if (!res.ok) throw new Error(`font fetch ${res.status}: ${url}`);
    writeFileSync(path, Buffer.from(await res.arrayBuffer()));
  }
  return opentype.parse(readFileSync(path));
}

const font = await loadFont("PommelPorter-BreeSerif-Regular.ttf", "https://github.com/google/fonts/raw/main/ofl/breeserif/BreeSerif-Regular.ttf");
const claimFont = await loadFont("PommelPorter-Lato-Regular.ttf", "https://github.com/google/fonts/raw/main/ofl/lato/Lato-Regular.ttf");

const nameW = font.getAdvanceWidth(NAME, nameSize);
const claimW = claimFont.getAdvanceWidth(CLAIM, claimSize);
const startX = Math.round((W - (LOGO + gap + Math.max(nameW, claimW))) / 2);
const textX = startX + LOGO + gap;

const nameAsc = font.ascender * (nameSize / font.unitsPerEm);
const nameDesc = -font.descender * (nameSize / font.unitsPerEm);
const claimAsc = claimFont.ascender * (claimSize / claimFont.unitsPerEm);
// The claim's descender counts toward the block height, or a claim with a y or
// p sits lower than the logo's centre.
const claimDesc = -claimFont.descender * (claimSize / claimFont.unitsPerEm);
const blockH = nameAsc + nameDesc + lineGap + claimAsc + claimDesc;
const nameBaseline = H / 2 - blockH / 2 + nameAsc;
const claimBaseline = nameBaseline + nameDesc + lineGap + claimAsc;

// One <g> per glyph, each path computed at x=0 and placed by its transform:
// opentype.js emits NaN for some glyphs once the x it is given grows past a few
// hundred units.
function textGroups(fnt, text, fontSize, x0, y0) {
  const scale = fontSize / fnt.unitsPerEm;
  let cx = x0;
  const parts = [];
  for (let i = 0; i < text.length; i++) {
    const glyph = fnt.charToGlyph(text[i]);
    const d = glyph.getPath(0, 0, fontSize).toPathData(2);
    parts.push(`<g transform="translate(${cx.toFixed(2)},${y0.toFixed(2)})"><path d="${d}"/></g>`);
    cx += glyph.advanceWidth * scale;
    if (i < text.length - 1) cx += fnt.getKerningValue(glyph, fnt.charToGlyph(text[i + 1])) * scale;
  }
  return parts.join("");
}

const logo = readFileSync(join(__dir, "logo.svg"), "utf8").replace(
  /<svg\b[^>]*>/,
  `<svg x="${startX}" y="${(H - LOGO) / 2}" width="${LOGO}" height="${LOGO}" viewBox="0 0 256 256" xmlns="http://www.w3.org/2000/svg">`,
);

for (const t of THEMES) {
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${W}" height="${H}" viewBox="0 0 ${W} ${H}">
  <rect width="${W}" height="${H}" fill="${t.bg}"/>
  ${logo}
  <g fill="${t.name}">${textGroups(font, NAME, nameSize, textX, nameBaseline)}</g>
  <g fill="${t.claim}">${textGroups(claimFont, CLAIM, claimSize, textX, claimBaseline)}</g>
</svg>
`;
  const name = `pommelporter-banner${t.suffix}`;
  writeFileSync(join(__dir, `${name}.svg`), svg);
  writeFileSync(join(__dir, `${name}.png`), new Resvg(svg, { background: t.bg, fitTo: { mode: "width", value: W } }).render().asPng());
  console.log(`wrote ${name}.svg + .png`);
}
