/**
 * Renders the README screenshots in the style of the house's store pictures: a
 * captured installer page, tilted, on a dark ground with the pommel as a relief,
 * a heading and a line of text beside it. 1920x1000 each.
 *
 * The captures in captures/ are 760x560 at twice the pixel density, taken from
 * the UI's mock (`npm run dev` in ui/) in dark mode, with the clear edge around
 * the card left transparent, since the card is the whole window.
 *
 * Needs playwright-core installed globally and Chrome.
 * Run: node .github/assets/screenshots/render.mjs
 */
import { existsSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { tmpdir } from "node:os";
import { fileURLToPath } from "node:url";
import { createRequire } from "node:module";
import { execSync } from "node:child_process";

const require = createRequire(import.meta.url);
const { chromium } = require(`${execSync("npm root -g").toString().trim()}/playwright-core`);

const here = dirname(fileURLToPath(import.meta.url));

const SHOTS = [
  { file: "folder.png", capture: "folder.png", caption: "Any folder,<br><em>on any drive</em>", sub: "One installer for KnightLoader, ArrowLoop and TrickWork on Windows and Linux" },
  { file: "progress.png", capture: "progress.png", caption: "Every step <em>in plain sight</em>", sub: "It adds a Start menu entry, a desktop shortcut, an entry under Apps and a daily update check" },
  { file: "uninstall.png", capture: "uninstall.png", caption: "Out again,<br><em>your data stays</em>", sub: "The uninstaller lists what it removes and what it keeps" },
];

async function cached(file, url) {
  const path = join(tmpdir(), `pommelporter-${file}`);
  if (!existsSync(path)) {
    const res = await fetch(url);
    if (!res.ok) throw new Error(`${file}: fetch ${res.status}`);
    writeFileSync(path, Buffer.from(await res.arrayBuffer()));
  }
  return readFileSync(path);
}

const dataUrl = (buf, type) => `data:${type};base64,${buf.toString("base64")}`;
const fonts = "https://github.com/google/fonts/raw/main/ofl";
const bree = dataUrl(await cached("BreeSerif-Regular.ttf", `${fonts}/breeserif/BreeSerif-Regular.ttf`), "font/ttf");
const lato = dataUrl(await cached("Lato-Regular.ttf", `${fonts}/lato/Lato-Regular.ttf`), "font/ttf");
const logo = dataUrl(readFileSync(join(here, "..", "logo.svg")), "image/svg+xml");

const STYLE = `
@font-face { font-family: "Bree Serif"; src: url(${bree}); }
@font-face { font-family: Lato; src: url(${lato}); }
* { box-sizing: border-box; margin: 0; }
body { position: relative; width: 1920px; height: 1000px; overflow: hidden; font-family: Lato, sans-serif; background: #0c0c0b; }
.wall { position: absolute; inset: 0; background: radial-gradient(55% 60% at 62% 35%, #26231d, #0f0e0c 72%); }
.relief { position: absolute; width: 52%; left: -10%; top: 6%; opacity: .32;
  filter: grayscale(1) brightness(.36) contrast(1.2) drop-shadow(-2px -2px 0 rgba(255,255,255,.16)) drop-shadow(12px 18px 26px rgba(0,0,0,.85)); }
.vignette { position: absolute; inset: 0; box-shadow: inset 0 0 200px rgba(0,0,0,.6); }
.copy { position: absolute; left: 84px; top: 0; bottom: 0; width: 520px; display: flex; flex-direction: column; justify-content: center; gap: 28px; }
.copy img { width: 132px; }
h1 { font: 400 64px/1.12 "Bree Serif", serif; color: #f4f4f4; }
h1 em { font-style: normal; color: #FCC419; }
.sub { font-size: 28px; line-height: 1.35; color: #9d9481; }
.stage { position: absolute; left: 700px; top: 110px; width: 1100px; perspective: 2400px; }
.floor { position: absolute; left: 12%; right: 12%; bottom: 0; height: 8%; border-radius: 50%; background: rgba(0,0,0,.8); filter: blur(30px); }
.win { position: relative; transform: rotateY(-8deg) rotateX(2deg); }
.win > img { display: block; width: 1100px; height: 811px; filter: drop-shadow(0 18px 36px rgba(0,0,0,.45)) drop-shadow(0 52px 100px rgba(0,0,0,.55)); }
`;

const page = ({ capture, caption, sub }) => `<!doctype html><html><head><meta charset="utf-8"><style>${STYLE}</style></head><body>
<div class="wall"></div><img class="relief" src="${logo}"><div class="vignette"></div>
<div class="copy"><img src="${logo}"><h1>${caption}</h1><p class="sub">${sub}</p></div>
<div class="stage"><div class="floor"></div><div class="win">
  <img src="${dataUrl(readFileSync(join(here, "captures", capture)), "image/png")}">
</div></div>
</body></html>`;

// Rendered at twice the size and scaled down, which keeps the type crisp.
const browser = await chromium.launch({ channel: "chrome" });
for (const shot of SHOTS) {
  const big = await browser.newPage({ viewport: { width: 1920, height: 1000 }, deviceScaleFactor: 2 });
  await big.setContent(page(shot));
  await big.evaluate(() => document.fonts.ready);
  const png = await big.screenshot();
  await big.close();
  const small = await browser.newPage({ viewport: { width: 1920, height: 1000 } });
  await small.setContent(`<body style="margin:0"><img src="${dataUrl(png, "image/png")}" style="display:block;width:1920px;height:1000px">`);
  await small.locator("img").evaluate((img) => img.decode());
  await small.screenshot({ path: join(here, shot.file) });
  await small.close();
  console.log(`${shot.file} written`);
}
await browser.close();
