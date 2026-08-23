#!/usr/bin/env node
/**
 * ══════════════════════════════════════════════════════════════════════
 * **دليلُ الشبكة — أيَّ نطاقٍ تُكلّم خريطةُ الويب فعلاً؟**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (إغلاقُ `TD-WEB-RASTER-SOURCE`، البند ٢٥.)
 *
 * **وفحصُ المصدر يُثبت أنّ النصَّ ليس فيها** — **وهذا يُثبت أنّ
 * المتصفّحَ لا يطلبها.** وهما شيئان: مكتبةٌ قد تحمل مصدراً افتراضيّاً
 * في شيفرتها لا في شيفرتنا.
 *
 *   node scripts/check-map-network.mjs http://localhost:3003
 */

import { chromium } from "playwright";

const BASE = (process.argv[2] || "http://localhost:3003").replace(/\/+$/, "");

/** **الصفحاتُ التي فيها خريطة** — كلُّها لا عيّنة. */
const PAGES = ["/cart", "/custom", "/join", "/contact", "/dashboard/settings"];

/** **ما لا يجوز أن يُطلَب.** */
const FORBIDDEN = [
  "tile.openstreetmap.",
  "basemaps.cartocdn.com",
  "demotiles.maplibre.org",
  "api.maptiler.com",
  "api.mapbox.com",
  "tiles.stadiamaps.com",
];

const browser = await chromium.launch();
const ctx = await browser.newContext({ locale: "ar" });
const page = await ctx.newPage();

const domains = new Map();
const violations = [];

page.on("request", (req) => {
  const url = req.url();
  if (url.startsWith("data:") || url.startsWith("blob:")) return;
  let host;
  try {
    host = new URL(url).host;
  } catch {
    return;
  }
  domains.set(host, (domains.get(host) ?? 0) + 1);
  for (const bad of FORBIDDEN) {
    if (url.includes(bad)) violations.push(url);
  }
});

for (const p of PAGES) {
  try {
    await page.goto(BASE + p, { waitUntil: "networkidle", timeout: 90_000 });
    // **ومهلةٌ للخريطة كي تحاول** — فالطلبُ الممنوعُ يقع بعد التحميل.
    await page.waitForTimeout(2500);
  } catch (e) {
    console.error(`  ! تعذّر تحميل ${p}: ${String(e).slice(0, 80)}`);
  }
}

await browser.close();

console.log("  **النطاقاتُ التي كُلِّمت**");
for (const [host, n] of [...domains].sort((a, b) => b[1] - a[1])) {
  console.log(`    ${String(n).padStart(4)} × ${host}`);
}

if (violations.length) {
  console.error("\nخريطةُ الويب طلبت مصدراً عموميّاً:");
  for (const v of [...new Set(violations)].slice(0, 10)) console.error("  ✗ " + v);
  process.exit(7);
}
console.log(`\nحارسُ شبكةِ الخريطة: صفرُ طلبٍ إلى مصدرٍ عموميّ  (${PAGES.length} صفحات)`);
