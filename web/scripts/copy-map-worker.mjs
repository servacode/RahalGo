#!/usr/bin/env node
/**
 * ══════════════════════════════════════════════════════════════════════
 * **نسخُ عامل MapLibre إلى الأصول العامّة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (إغلاقُ `TD-WEB-RASTER-SOURCE`، ٢٠٢٦-٠٨-٢١.)
 *
 * **MapLibre يترك بناءَ عاملِه للحُزَم، وTurbopack لا يبنيه** —
 * فلا يُنشَأ العاملُ **ولا يُرمى خطأ**، والخريطةُ تبقى فارغة.
 *
 * **فيُنسخ الملفّان ويُعطى المحرّكُ مساراً صريحاً** (`setWorkerUrl`).
 *
 * **وملفّان لا ملفّ**: الشِبهُ يستورد المشتركَ بمسارٍ نسبيّ.
 *
 * **ويُنادى قبل `dev` و`build`** — فلا يُنسى عند ترقية.
 */

import { copyFileSync, mkdirSync, statSync } from "node:fs";
import { createRequire } from "node:module";
import { dirname, join } from "node:path";

const require = createRequire(import.meta.url);
const ROOT = new URL("..", import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, "$1");
const PUBLIC = join(ROOT, "apps", "rahalgo", "public");

/** **ما يلزم العاملَ ليعمل.** */
const FILES = ["maplibre-gl-worker.mjs", "maplibre-gl-shared.mjs"];

// **ويُحلّ عبرَ `package.json`** — فحزمةُ MapLibre لا تُصدّر مدخلاً
// صالحاً لـ`require.resolve`.
const pkg = require.resolve("maplibre-gl/package.json", {
  paths: [join(ROOT, "packages", "ui")],
});
const dist = join(dirname(pkg), "dist");

mkdirSync(PUBLIC, { recursive: true });
for (const f of FILES) {
  const from = join(dist, f);
  const to = join(PUBLIC, f);
  copyFileSync(from, to);
  console.log(`  · ${f} — ${(statSync(to).size / 1024).toFixed(0)} KB`);
}
console.log("عاملُ الخريطة في الأصول العامّة.");
