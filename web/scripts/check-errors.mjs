/**
 * **حارسُ رسائل الخطأ** — يمنع أن يُرمى سببُ الخادم.
 *
 * # لماذا حارسٌ لهذا
 *
 * **`catch { setError(m.errors.internal) }` يمسك خطأَ الخادم ويرميه** ويكتب
 * مكانَه رسالةً واحدةً لكلّ العلل. **فيُقال للمستخدم «الخطأُ عندنا» والعلّةُ
 * في مُدخَله** — رقمٌ بلا واتساب، صورةٌ فوق الحدّ، قسمٌ فيه أصناف.
 *
 * **ووقع في عشرة مواضع** (٢٠٢٦-٠٨-١١): شهد المالكُ اثنين — توثيقَ الواتساب
 * وإثباتَ التسليم. **وثمانيةٌ لم يبلغها بعد**، وكانت تنتظره.
 *
 * **والخادمُ يقول السببَ في كلّ مرّة** — والشاشةُ كانت تُسكته.
 *
 * # وما يُقبل
 *
 * **`errorText(err)` من المعجم** — تقرأ مفتاحَ الخادم وتردّ نصَّه العربيّ،
 * **وتسقط إلى العامّة لما لا تعرفه وحدَه.**
 */

import { readFileSync, readdirSync, statSync } from "node:fs";
import { join, relative } from "node:path";

const ROOT = new URL("..", import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, "$1");
const SKIP = new Set(["node_modules", ".next", ".turbo", "dist", "build", ".git"]);

function walk(dir, out = []) {
  for (const name of readdirSync(dir)) {
    if (SKIP.has(name)) continue;
    const p = join(dir, name);
    if (statSync(p).isDirectory()) walk(p, out);
    else if (p.endsWith(".tsx") || p.endsWith(".ts")) out.push(p);
  }
  return out;
}

/** **رسالةٌ عامّةٌ تُكتب مكانَ خطأ الخادم** — وهي ما نمنعه. */
const GENERIC = /setError\(\s*m\.errors\.(internal|validation)\s*\)/;

/* ══════════════════════════════════════════════════════════════════════
   **والسهمُ يُمسَك كما تُمسَك الكتلة**
   ══════════════════════════════════════════════════════════════════════

   (كشفه فحصُ المالك ٢٠٢٦-٠٨-١٦ في «الأهداف والمكافآت».)

   **كان الحارسُ يفحص `catch {` وحدَها** — **و`.catch(() => …)` سهمٌ يمرّ
   منه سالماً.** فمرّت شاشةُ الأهداف بـ`setError(m.errors.internal)` في
   سهمٍ ومعها `setRows([])`: **«لا أحدَ بلغ الهدفَ» تُقرأ على قراءةٍ
   فشلت**، وهي استنتاجٌ يُبنى عليه قرارُ صرفِ مال.

   **وحارسٌ يمسك شكلاً واحداً من شكلين يُطمئن ولا يحرس** — ومن قرأ
   «سليم» ظنّ الشاشاتِ كلَّها تقول أسبابَها. */
const ARROW_CATCH = /\.catch\(\s*\(\s*\)\s*=>/;

const problems = [];
for (const file of walk(ROOT)) {
  const rel = relative(ROOT, file).split("\\").join("/");
  if (rel.startsWith("scripts/")) continue;
  /* ══════════════════════════════════════════════════════════════════
     **والتعليقاتُ تُبيَّض قبل الفحص — لا تُقفز بالشكل**
     ══════════════════════════════════════════════════════════════════

     **الشرحُ يذكر النمطَ الممنوعَ ليقول إنّه ممنوع** — وسطرٌ في وسط تعليقٍ
     طويلٍ لا يبدأ بعلامةٍ تدلّ عليه. **فقفزُ ما يبدأ بـ`*` وحدَه لا يكفي**،
     وقد أنذر الحارسُ على شرحِ نفسِه.

     **وحارسٌ يُنذر كذباً يُطفأ** — ومن أطفأه فقد ما يحرسه أيضاً.

     **فتُبيَّض الأسطرُ داخل `/* … *​/` كلُّها** مع إبقاء عددِها — فرقمُ
     السطر المُبلَّغُ هو رقمُه في الملفّ. */
  const raw = readFileSync(file, "utf8");
  let inBlock = false;
  const lines = raw.split("\n").map((l) => {
    let out = l;
    if (inBlock) {
      const end = l.indexOf("*/");
      out = end < 0 ? "" : l.slice(end + 2);
      inBlock = end < 0;
      return out;
    }
    const start = out.indexOf("/*");
    if (start >= 0 && out.indexOf("*/", start) < 0) {
      inBlock = true;
      out = out.slice(0, start);
    }
    return out.replace(/\/\/.*$/, "");
  });
  lines.forEach((line, i) => {
    // **`catch {` بلا متغيّرٍ يرمي الخطأ** — والرسالةُ في سطره أو الذي يليه.
    if (/catch\s*\{\s*$/.test(line) && GENERIC.test(lines[i + 1] ?? "")) {
      problems.push({ file: rel, line: i + 2 });
    } else if (/catch\s*\{/.test(line) && !/catch\s*\(/.test(line) && GENERIC.test(line)) {
      problems.push({ file: rel, line: i + 1 });
    }
    // **والسهمُ بلا وسيطٍ يرمي الخطأ كذلك** — والرسالةُ في نافذته.
    if (ARROW_CATCH.test(line) && GENERIC.test(lines.slice(i, i + 5).join("\n"))) {
      problems.push({ file: rel, line: i + 1 });
    }
  });
}

if (problems.length === 0) {
  console.log("رسائلُ الخطأ سليمة — لا شاشةَ تُسكت سببَ الخادم.");
  process.exit(0);
}
console.log(`رسائلُ الخطأ: ${problems.length} موضعاً يرمي سببَ الخادم\n`);
for (const p of problems) {
  console.log(`   ${p.file}:${p.line}`);
  console.log("      الإصلاح: catch (err) { setError(errorText(err)) }  — من @rahalgo/i18n");
  console.log("      وللسهم: .catch((err) => setError(errorText(err)))");
}
process.exit(1);
