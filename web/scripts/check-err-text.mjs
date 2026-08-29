// ══════════════════════════════════════════════════════════════════════
//  **مترجمُ الخطأ واحدٌ — ولا يُكتب في شاشة**
// ══════════════════════════════════════════════════════════════════════
//
// **قِيس ٢٠٢٦-٠٨-٢٧**: `errText` كانت مكتوبةً في **واحدٍ وعشرين ملفّاً**
// **بثمانِ صيغٍ مختلفة** — فالخطأُ نفسُه يُعرض بنصٍّ مختلفٍ بحسب الشاشة
// التي وقع فيها.
//
// # وواحدةٌ منها كانت تبتلع السبب
//
// `app/dashboard/reports/page.tsx`:
//
//     return err instanceof ApiError ? m.errors.internal : m.errors.internal;
//
// **الفرعان متطابقان** — فسببُ الخادم يُرمى دائماً، **ويرى المستخدمُ
// «خطأ داخليّ» مهما قال المحرّك.** ولا يُكتشف هذا في بناءٍ ولا في
// مراجعةٍ سريعة: **الشيفرةُ تُقرأ سليمةً حتّى تُقارَن بأختها.**
//
// # والمركزيّةُ في `@rahalgo/i18n`
//
// `errorText(err, messages)` تقرأ المفتاحَ كاملاً ثمّ بآخر جزئه —
// **فتعرف `errors.x` و`auth.otpInvalid` معاً**، وهو ما كانت كلُّ صيغةٍ
// محلّيّةٍ تعرف نصفَه.
//
// # ولماذا حارسٌ لا مجرّدُ إصلاح
//
// **لأنّ الحادثة تتكرّر بطبعها**: من كتب شاشةً جديدةً احتاج نصَّ خطأٍ،
// **فنسخ الدالّةَ من الشاشة المجاورة** — وهي أقربُ من استيرادٍ يبحث
// عنه. **والحارسُ يجعل النسخَ يسقط البناءَ لا يمرّ.**

import { readFileSync, readdirSync, statSync } from "node:fs";
import { join, relative } from "node:path";
import { fileURLToPath } from "node:url";
import { dirname } from "node:path";

const here = dirname(fileURLToPath(import.meta.url));
const repo = join(here, "..");

/** **المصدَّرةُ الوحيدةُ المسموحة** — غلافٌ رقيقٌ فوق `errorText`. */
const ALLOWED = [
  // **الأصلُ نفسُه** — وهو ما يستوردُه الجميع.
  "packages/i18n/src/index.ts",
  // **وغلافٌ رقيقٌ فوقه** يحمل رسائلَ الحزمة، تستورده تطبيقاتٌ خارجيّة.
  "packages/auth/src/LoginCard.tsx",
];

const ROOTS = ["apps/rahalgo/src", "packages"];
const SKIP = new Set(["node_modules", ".next", ".turbo", "dist", "build"]);

function walk(dir, out = []) {
  for (const name of readdirSync(dir)) {
    if (SKIP.has(name)) continue;
    const p = join(dir, name);
    if (statSync(p).isDirectory()) walk(p, out);
    else if (p.endsWith(".ts") || p.endsWith(".tsx")) out.push(p);
  }
  return out;
}

const bad = [];
for (const root of ROOTS) {
  for (const p of walk(join(repo, root))) {
    const rel = relative(repo, p).split("\\").join("/");
    if (ALLOWED.includes(rel)) continue;
    const src = readFileSync(p, "utf8");
    const lines = src.split(String.fromCharCode(10));
    lines.forEach((l, i) => {
      if (/^\s*(?:export\s+)?(?:function|const)\s+err(?:Text|orText)\s*[(=]/.test(l)) {
        bad.push([rel, i + 1, l.trim().slice(0, 70)]);
      }
    });
  }
}

if (bad.length) {
  console.error("");
  console.error("مترجمُ الخطأ مكتوبٌ خارجَ مكانه:");
  for (const [f, n, l] of bad) console.error(`  ${f}:${n}  ${l}`);
  console.error("");
  console.error("  الإصلاح:  import { errorText } from \"@rahalgo/i18n\"");
  console.error("            وهي تقرأ المفتاحَ كاملاً وبآخر جزئه معاً.");
  process.exit(1);
}

console.log("مترجمُ الخطأ واحدٌ — ولا نسخةَ تنحرف عنه.");
