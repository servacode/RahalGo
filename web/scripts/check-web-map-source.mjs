#!/usr/bin/env node
/**
 * ══════════════════════════════════════════════════════════════════════
 * **حارسُ مصدر خريطة الويب — `TD-WEB-RASTER-SOURCE`**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٢١، البندان ٢٦ و٢٧.)
 *
 * # ما يمنعه
 *
 * **كانت الخريطةُ ترتدّ إلى `tile.openstreetmap.org` بعد أربعةِ أخطاءِ
 * بلاطة** — **ومع ضبط مصدرنا الخاصّ.** فالارتدادُ لم يكن غفلةً بل
 * تصميماً، **ومن أعاده لن يلاحظه أحدٌ حتّى تنقطع الخدمة.**
 *
 * **وهذا الحارسُ يمنع عودتَه** — نصّاً أو نطاقاً أو مرآةً.
 */

import { readFileSync, readdirSync, statSync } from "node:fs";
import { join, relative } from "node:path";

const ROOT = new URL("..", import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, "$1");

/** **ما لا يجوز أن يُطلَب وقتَ التشغيل أبداً.** */
const FORBIDDEN = [
  "tile.openstreetmap.org",
  "tile.openstreetmap.de",
  "basemaps.cartocdn.com",
  "demotiles.maplibre.org",
  "api.maptiler.com",
  "api.mapbox.com",
  "tiles.stadiamaps.com",
  "tile.opentopomap.org",
];

/** **ومرايا OSM بأسماءِ نطاقاتٍ فرعيّة** — `a.tile.` و`{s}.tile.`. */
const MIRROR = /(?:[{[]s[}\]]|\b[a-c])\.tile\.openstreetmap\./i;

const SKIP = new Set(["node_modules", ".next", ".turbo", "dist", "build", ".git"]);
const EXT = /\.(tsx?|jsx?|mjs|cjs|json|css)$/;

function walk(dir, out = []) {
  for (const name of readdirSync(dir)) {
    if (SKIP.has(name)) continue;
    const p = join(dir, name);
    const st = statSync(p);
    if (st.isDirectory()) walk(p, out);
    else if (EXT.test(name)) out.push(p);
  }
  return out;
}

const problems = [];
const notes = [];

// ══════════════════════════════════════════════════════════════════════
// **١ · لا مصدرَ عموميٍّ في مصدرِ الويب**
// ══════════════════════════════════════════════════════════════════════

const files = [
  ...walk(join(ROOT, "apps")),
  ...walk(join(ROOT, "packages")),
];
let scanned = 0;
for (const f of files) {
  const rel = relative(ROOT, f).replace(/\\/g, "/");
  // **والحارسُ نفسُه يذكرها ليمنعها** — فيُستثنى هو وقائمتُه.
  if (rel.endsWith("scripts/check-web-map-source.mjs")) continue;
  if (rel.endsWith("packages/ui/src/mapconfig.ts")) continue;
  const src = readFileSync(f, "utf8");
  scanned++;
  for (const host of FORBIDDEN) {
    if (src.includes(host)) {
      problems.push(`${rel}: مصدرُ خرائطَ عموميٌّ — ${host}`);
    }
  }
  if (MIRROR.test(src)) {
    problems.push(`${rel}: مرآةُ OSM عموميّة`);
  }
}
notes.push(`فُحص ${scanned} ملفّاً في الويب`);

// ══════════════════════════════════════════════════════════════════════
// **٢ · ولا مكتبةَ راستر باقية**
// ══════════════════════════════════════════════════════════════════════

for (const pkg of ["apps/rahalgo/package.json", "packages/ui/package.json"]) {
  const j = JSON.parse(readFileSync(join(ROOT, pkg), "utf8"));
  const deps = { ...j.dependencies, ...j.devDependencies };
  for (const bad of ["leaflet", "react-leaflet", "@types/leaflet"]) {
    if (deps[bad]) problems.push(`${pkg}: ما زالت ${bad} معتمدة`);
  }
  if (pkg.endsWith("ui/package.json")) {
    for (const need of ["maplibre-gl", "pmtiles"]) {
      if (!deps[need]) problems.push(`${pkg}: تنقص ${need}`);
    }
  }
}

// ══════════════════════════════════════════════════════════════════════
// **٣ · ولا ارتدادَ في المكوّن**
// ══════════════════════════════════════════════════════════════════════

const mapSrc = readFileSync(join(ROOT, "packages/ui/src/map.tsx"), "utf8");
if (/TILE_SOURCES|FallbackTileLayer|srcIndex/.test(mapSrc)) {
  problems.push("packages/ui/src/map.tsx: سلسلةُ ارتدادٍ للبلاطات ما زالت قائمة");
}
if (!mapSrc.includes("resolveMapSource")) {
  problems.push("packages/ui/src/map.tsx: لا يحسم المصدرَ مركزيّاً");
}

const cfg = readFileSync(join(ROOT, "packages/ui/src/mapconfig.ts"), "utf8");
if (!cfg.includes("NEXT_PUBLIC_MAP_STYLE_URL")) {
  problems.push("mapconfig: لا يقرأ NEXT_PUBLIC_MAP_STYLE_URL");
}
if (cfg.includes("NEXT_PUBLIC_TILE_URL")) {
  problems.push("mapconfig: ما زال يقرأ الاسمَ المضلّل NEXT_PUBLIC_TILE_URL");
}
// **وغيابُ الإعداد لا يرتدّ** — البند ١٩.
if (!/reason: "missing"/.test(cfg)) {
  problems.push("mapconfig: غيابُ الإعداد لا يُنتج حالَ عطلٍ صريحة");
}

// ══════════════════════════════════════════════════════════════════════
// **٤ · والنمطُ مصدرٌ واحدٌ مع أندرويد**
// ══════════════════════════════════════════════════════════════════════

const canonical = join(ROOT, "..", "maps", "style", "rahalgo.style.json");
try {
  const st = JSON.parse(readFileSync(canonical, "utf8"));
  notes.push(`النمطُ المرجعيّ: ${st.layers.length} طبقة · مصدرٌ واحدٌ للويب وأندرويد`);
} catch {
  problems.push("لم يُقرأ النمطُ المرجعيّ maps/style/rahalgo.style.json");
}
const webStyles = files.filter((f) => /web-style\.json|map-style\.json/.test(f));
if (webStyles.length) {
  problems.push(`نمطٌ ثانٍ للويب ينحرف عن المرجعيّ: ${webStyles.length} ملفّاً`);
}

// ══════════════════════════════════════════════════════════════════════

if (problems.length) {
  console.error("خريطةُ الويب — عيوب:");
  for (const p of problems) console.error("  ✗ " + p);
  process.exit(7);
}
for (const n of notes) console.log("  · " + n);
console.log("خريطةُ الويب من مصدرٍ نملكه — ولا ارتدادَ عموميّ.");
