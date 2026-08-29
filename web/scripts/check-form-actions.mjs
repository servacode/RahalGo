// ══════════════════════════════════════════════════════════════════════
//  **ذيلُ النافذة مركزيّ — وموضعُ «حفظ» لا يتنقّل**
// ══════════════════════════════════════════════════════════════════════
//
// **قِيس ٢٠٢٦-٠٨-٢٧**: اثنان وثلاثون ذيلاً مكتوبةً بيدٍ في اللوحة
// والموقع، **وفيها عيبان يراهما المستخدمُ ولا يراهما بناء:**
//
// **١ · زرُّ الإلغاء بصيغتين** — `ghost` في خمسة و`secondary` في خمسةَ
// عشر. **فالزرُّ نفسُه زرٌّ في شاشةٍ ونصفُ زرٍّ في أخرى.**
//
// **٢ · وترتيبُهما ينقلب** — «إلغاء ثمّ حفظ» في ستّ نوافذَ و«حفظ ثمّ
// إلغاء» في ثلاث. **فموضعُ «حفظ» يتنقّل بين نافذةٍ وأخرى.**
//
// **ومن حفظ موضعَ زرٍّ ضغطه بلا نظر** — وهو ما يفعله من يعمل على اللوحة
// يومَه كلَّه. **فيقع على «إلغاء» وقد ملأ نموذجاً كاملاً.**
//
// # وهذا لا يُمسَك في مراجعة
//
// **كلُّ ملفٍّ يُقرأ سليماً وحدَه** — العيبُ في الفرق بينه وبين أخيه،
// **ولا أحدَ يفتح اثنين وثلاثين ملفّاً ليقارن.**
//
// # والإصلاح
//
//     import { FormActions } from "@rahalgo/ui";
//     <FormActions onSave={submit} onCancel={onClose} busy={busy} />
//
// **وتقبل `submit` و`saveLabel` و`tone="danger"`** — فما احتاج نصّاً
// آخرَ أو لهجةَ حذفٍ يجدها، **ولا يعود أحدٌ يكتبها بيده.**

import { readFileSync, readdirSync, statSync } from "node:fs";
import { join, relative, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const repo = join(here, "..");

const ROOTS = ["apps/rahalgo/src", "packages"];
const SKIP = new Set(["node_modules", ".next", ".turbo", "dist", "build"]);

/** **الذيلُ المركزيُّ نفسُه** — هو من يكتب الزرّين. */
const ALLOWED = ["packages/ui/src/FormActions.tsx"];

function walk(dir, out = []) {
  for (const name of readdirSync(dir)) {
    if (SKIP.has(name)) continue;
    const p = join(dir, name);
    if (statSync(p).isDirectory()) walk(p, out);
    else if (p.endsWith(".tsx")) out.push(p);
  }
  return out;
}

const NL = String.fromCharCode(10);
const bad = [];
let central = 0;

for (const root of ROOTS) {
  for (const p of walk(join(repo, root))) {
    const rel = relative(repo, p).split("\\").join("/");
    const src = readFileSync(p, "utf8");
    central += (src.match(/<FormActions/g) ?? []).length;
    if (ALLOWED.includes(rel)) continue;

    // **زرٌّ نصُّه «إلغاء» ومعه زرٌّ آخرُ في الحاوية نفسِها** — وهو الذيل.
    const lines = src.split(NL);
    lines.forEach((l, i) => {
      if (!/\{m\.common\.cancel\}/.test(l)) return;
      // **يُنظر حولَه**: أهو داخل صفٍّ فيه زرٌّ ثانٍ؟
      const around = lines.slice(Math.max(0, i - 8), i + 9).join(NL);
      const buttons = (around.match(/<Button\b/g) ?? []).length;
      if (buttons >= 2) bad.push([rel, i + 1, l.trim().slice(0, 60)]);
    });
  }
}

if (bad.length) {
  console.error("");
  console.error("ذيلُ نافذةٍ مكتوبٌ بيد:");
  for (const [f, n, l] of bad) console.error(`  ${f}:${n}  ${l}`);
  console.error("");
  console.error("  الإصلاح:  <FormActions onSave={…} onCancel={…} busy={…} />");
  console.error("            من \"@rahalgo/ui\" — وموضعُ «حفظ» لا يتنقّل.");
  process.exit(1);
}

console.log(`ذيلُ النوافذ مركزيٌّ — ${central} موضعاً، ولا زرَّ يتنقّل.`);
