/**
 * **شبكةٌ بلا عمودٍ أساسيّ — وهي تفيض عن الجوّال.**
 *
 * (شكوى المالك ٢٠٢٦-٠٨-٠٧: «التجاوبُ على الموبايل وكلّ الجوّالات والشاشات
 *  أبداً مو مضبوط».)
 *
 * **`grid md:grid-cols-2` بلا `grid-cols-1`**: تحت `md` لا جدولَ أعمدةٍ
 * أصلاً، **فيُنشئ المتصفّحُ عموداً ضمنيّاً مقاسُه `max-content`** — وهو لا
 * يتقيّد بعرض الحاوية.
 *
 * **وقِيس على `‎/orders` بجوّال ٣٦٠**: الحاويةُ ٣٣٦ والمسارُ ٣٩٩٫٢٣،
 * **والمستندُ ٤٢٢** — فتنزلق الصفحةُ أفقيّاً ويُقصّ الشريطُ العلويّ.
 *
 * **ولا يمسكها الحارسُ القديم**: هو يفحص `grid-cols-[3-9]` بلا سابقةٍ —
 * **وهذه بلا رقمٍ أصلاً.**
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

const hits = [];
for (const abs of walk(ROOT)) {
  const f = relative(ROOT, abs).split(sep).join("/");
  const src = readFileSync(abs, "utf8");
  src.split("\n").forEach((line, i) => {
    // كلُّ سلسلةِ أصنافٍ فيها `grid`
    for (const m of line.matchAll(/(?:class[nN]ame=)?["'`]([^"'`]*\bgrid\b[^"'`]*)["'`]/g)) {
      const cls = m[1];
      if (!/(?:^|\s)grid(?:\s|$)/.test(cls)) continue;
      const responsive = /\b(?:sm|md|lg|xl|2xl):grid-cols-\d/.test(cls);
      const base = /(?:^|\s)grid-cols-\d/.test(cls) || /grid-cols-\[/.test(cls);
      const flow = /\bgrid-flow-col\b|\bauto-cols-/.test(cls);
      if (responsive && !base && !flow) hits.push([f, i + 1, cls.slice(0, 62)]);
    }
  });
}

const bySec = new Map();
for (const [f] of hits) {
  const sec = f.startsWith("apps/") ? f.split("/")[1] : "packages/" + f.split("/")[1];
  bySec.set(sec, (bySec.get(sec) ?? 0) + 1);
}
console.log("شبكاتٌ بلا عمودٍ أساسيّ:", hits.length, "\n");
for (const [s, n] of [...bySec].sort((a, b) => b[1] - a[1])) console.log("  " + String(n).padStart(3) + "  " + s);
console.log("");
for (const [f, ln, cls] of hits) console.log(`  ${f}:${ln}\n      ${cls}`);
