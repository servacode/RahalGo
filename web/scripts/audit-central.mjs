/**
 * **جردُ ما هو خارجَ المركز — قسماً قسماً.**
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-٠٦: «صيدُ الألوان والأزرار والعناصر غير المركزيّة
 *  بالصفحات ونلاحقها… من أين نبدأ؟».)
 *
 * **ولا يُصلَح ما لا يُقاس**: الرقمُ لكلّ قسمٍ هو من يقرّر الترتيب، لا الظنّ.
 * **وهذا جردٌ لا حارس** — الحارسُ `check-central.mjs` يُسقط البناء، وهذا
 * يعدّ ليُقرَّر. **وكلُّ صنفٍ يُنظَّف ينتقل من هنا إلى هناك.**
 */
import { readdirSync, statSync, readFileSync } from "node:fs";
import { join, relative, sep } from "node:path";

const ROOT = process.cwd();
const SKIP = new Set(["node_modules", ".next", "dist", ".turbo", "coverage"]);
const walk = (d, out = []) => {
  for (const e of readdirSync(d)) {
    if (SKIP.has(e)) continue;
    const p = join(d, e);
    if (statSync(p).isDirectory()) walk(p, out);
    else if (e.endsWith(".tsx")) out.push(p);
  }
  return out;
};

/** **المركزُ نفسُه معفى** — فيه تُكتب اللغةُ لا تُخالَف. */
const CENTER = [
  "theme", "tokens", "icons", "components", "layout", "feedback", "overlay",
  "navigation", "topbar", "DashboardChrome", "platform", "MobileNav", "dataview",
].map((n) => `packages/ui/src/${n}.tsx`);

const CATS = [
  ["شفافيّة", /\b(?:bg|text|border|ring|divide|from|to)-[a-z-]+\/[0-9]{1,3}\b/g],
  ["شارة", /rounded-(?:badge|full)[^"'`]*\btext-(?:2xs|xs)\b|\btext-(?:2xs|xs)\b[^"'`]*rounded-(?:badge|full)/g],
  ["عنوان", /className="[^"]*\btext-(?:lg|xl|2xl|3xl)\b[^"]*\bfont-(?:bold|semibold)\b/g],
  ["زرّ", /<button[^>]*className="[^"]*\b(?:bg-(?:primary|accent|danger|success)|border-line)\b/g],
  ["حقل", /<(?:input|textarea|select)[^>]*className="/g],
  // **والحلقةُ موضعٌ لا صنف**: `ring-2 ring-accent ring-offset-2
  // ring-offset-shell` **أربعةٌ في مكانٍ واحد** — فكان الجردُ يقول تسعاً وهي
  // ثلاثة. **وعددٌ منتفخٌ يوجّه العملَ إلى غير موضعه.**
  ["حلقة", /ring-(?:[0-9]|[a-z])[a-z0-9-]*(?:\s+ring-[a-z0-9-]+)*/g],
  ["مقاسٌ حرفيّ", /text-\[[0-9.]+(?:px|rem)\]/g],
  ["ظلّ", /\bshadow-(?:sm|md|lg|xl|2xl|inner)\b/g],
  ["حشوةٌ عجيبة", /\b[pm][xytblrse]?-\[[^\]]+\]/g],
  ["قطرٌ يدويّ", /\brounded-(?:sm|md|lg|xl|2xl|3xl)\b/g],
];

const files = walk(ROOT).map((p) => relative(ROOT, p).split(sep).join("/"));
const bySection = new Map();
const byCat = new Map();
/** **التعليقاتُ بالعربية أسلوبُ المشروع** — والمقصودُ نصٌّ يبلغ الشاشة. */
function stripComments(src) {
  let out = "";
  for (let i = 0; i < src.length; ) {
    if (src.startsWith("/*", i)) {
      const j = src.indexOf("*/", i + 2);
      const end = j < 0 ? src.length : j + 2;
      // **الأسطرُ تبقى ويُمحى ما فيها.**
      //
      // **وحذفُها يُزيح كلَّ رقمٍ بعده**: يشير الحارسُ إلى سطرٍ بريء، **فيُقرأ
      // كاذباً ويُطفأ** — وحارسٌ لا يُصدَّق أسوأُ من لا حارس.
      out += src.slice(i, end).replace(/[^\n]/g, " ");
      i = end;
    } else if (src.startsWith("//", i)) {
      const j = src.indexOf("\n", i);
      const end = j < 0 ? src.length : j;
      out += " ".repeat(end - i);
      i = end;
    } else {
      out += src[i++];
    }
  }
  return out;
}

const samples = new Map();
const worst = new Map();

for (const f of files) {
  if (CENTER.includes(f)) continue;
  // **ولا يُعدّ ما في تعليق** — كان الجردُ يعدّ شرحاً يقول «كانت
  // `focus:ring-2`» **فيُحصي ما حُذف.**
  const src = stripComments(readFileSync(join(ROOT, f), "utf8"));
  const sec = f.startsWith("apps/") ? f.split("/")[1] : "packages/" + f.split("/")[1];
  let n = 0;
  for (const [name, re] of CATS) {
    const hits = src.match(re);
    if (!hits) continue;
    n += hits.length;
    const k = sec + "|" + name;
    bySection.set(k, (bySection.get(k) ?? 0) + hits.length);
    byCat.set(name, (byCat.get(name) ?? 0) + hits.length);
    if (!samples.has(name)) samples.set(name, `${f} → ${hits[0].slice(0, 44)}`);
  }
  if (n) worst.set(f, n);
}

const cats = CATS.map(([n]) => n).filter((n) => byCat.get(n));
const secs = [...new Set([...bySection.keys()].map((k) => k.split("|")[0]))];
const total = (s) => cats.reduce((a, c) => a + (bySection.get(s + "|" + c) ?? 0), 0);
secs.sort((a, b) => total(b) - total(a));

const pad = (s, w) => String(s).padStart(w);
const W = Math.max(...cats.map((c) => c.length)) + 3;
console.log("القسم".padEnd(14) + cats.map((c) => pad(c, W)).join("") + pad("المجموع", 10));
for (const s of secs) {
  console.log(s.padEnd(14) + cats.map((c) => pad(bySection.get(s + "|" + c) ?? "·", W)).join("") + pad(total(s), 10));
}
console.log("─".repeat(14 + cats.length * W + 10));
console.log("المجموع".padEnd(14) + cats.map((c) => pad(byCat.get(c), W)).join("") + pad([...byCat.values()].reduce((a, b) => a + b, 0), 10));

console.log("\nأثقلُ عشرة ملفّات:");
for (const [f, n] of [...worst].sort((a, b) => b[1] - a[1]).slice(0, 10)) console.log(`  ${pad(n, 4)}  ${f}`);
console.log("\nعيّنةٌ من كلّ صنف:");
for (const c of cats) console.log(`  ${c.padEnd(13)} ${samples.get(c)}`);
