/**
 * **كلُّ نداءٍ للمحرّك يمرّ بعميل المنصة.**
 *
 * (كشفه فحصٌ يدويٌّ ٢٠٢٦-٠٨-٠٨: رفعُ إثبات التسليم كان `fetch` خامّاً بلا
 *  ترويسة استيثاق — **٤٠١ لكلّ سائقٍ في كلّ تسليم**، وصورةُ التسليم
 *  إلزاميّةٌ افتراضاً، **فلا يصير طلبٌ مسلَّماً أبداً.**)
 *
 * # لماذا حارسٌ لا انتباه
 *
 * **العميلُ يعرف ثلاثةَ أشياءَ لا يعرفها `fetch`**: أين المحرّك، وكيف
 * يُلحَق الرمز، ومتى تُترك ترويسةُ النوع للمتصفّح (`FormData`). ومن كتب
 * نداءً بيده نسي واحداً منها على الأقلّ.
 *
 * **ولا يُمسَك في بناءٍ ولا تحقّقِ أنواع**: الشيفرةُ سليمةٌ نحواً، والخطأُ
 * يظهر في الخادم وقتَ التشغيل — **ويُبتلع في `catch` فيُعرض «حدث خطأ».**
 *
 * # ما يُسمح
 *
 * عميلُ المنصة نفسُه (`packages/auth/src/client.ts`) هو من ينادي `fetch` —
 * **وهو المستثنى الوحيد.** وما عداه يستورد `api`.
 */
import { readFileSync } from "node:fs";
import { globSync } from "node:fs";
import { join, relative } from "node:path";

const ROOT = new URL("..", import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, "$1");

/** الملفّاتُ التي يحقّ لها أن تنادي المحرّكَ مباشرةً. */
const ALLOWED = [
  "packages/auth/src/client.ts",
  // فحوصٌ وأدواتٌ لا تُشحن إلى متصفّح
  "scripts/",
];

const files = globSync("{apps,packages}/**/*.{ts,tsx}", { cwd: ROOT })
  .filter((f) => !f.includes("node_modules") && !f.includes(".next"));

const hits = [];
for (const rel of files) {
  const posix = rel.split("\\").join("/");
  if (ALLOWED.some((a) => posix.startsWith(a) || posix === a)) continue;
  const src = readFileSync(join(ROOT, rel), "utf8");
  const lines = src.split("\n");
  lines.forEach((line, i) => {
    if (!/\bfetch\s*\(/.test(line)) return;
    // المسارُ قد يمتدّ سطرين في قالبٍ نصّيّ.
    const span = line + (lines[i + 1] ?? "") + (lines[i + 2] ?? "");
    if (!/\/api\/v1\//.test(span)) return;

    /* **ومسارُ `public` لا رمزَ له أصلاً** — يُقرأ من الخادم قبل أن يوجد
       مستخدم (`sitemap`, `platform-server`)، **ومَنعُه منعُ ما لا ضرر فيه.** */
    if (/\/api\/v1\/public\//.test(span)) return;

    /* ══════════════════════════════════════════════════════════════════
       **ولا استثناءَ للترويسة اليدويّة بعد اليوم**
       ══════════════════════════════════════════════════════════════════

       **كان يُستثنى من ألحق `Authorization` بيده** — والسببُ مكتوبٌ:
       «التنزيلُ إلى ملفٍّ يحتاج `blob()` فلا يمرّ بعميلٍ يردّ JSON».

       **وكان صحيحاً يومَه** — ثمّ سقط: صارت `apiFile` تردّ الاستجابةَ
       خاماً **وتجدّد التوكنَ عند ٤٠١ كما تفعل `api`.**

       **والاستثناءُ خبّأ عطبين** (كشفهما فحصُ المالك ٢٠٢٦-٠٨-١٦):
       تصديرُ التقارير **يفشل بانتهاء توكنٍ والصفحةُ حولَه تعمل**،
       وتصديرُ الحسابات **يُنزّل ملفّاً اسمُه `accounts.csv` وفيه رسالةُ
       خطأ** — ولا يفحص النجاحَ أصلاً.

       **وحارسٌ باستثناءٍ بطل سببُه يحرس أقلَّ ممّا يُظنّ به.** */

    hits.push(`${posix}:${i + 1}  ${line.trim().slice(0, 90)}`);
  });
}

if (hits.length) {
  console.error("✗ نداءٌ للمحرّك خارجَ عميل المنصة — ولا ترويسةَ استيثاق فيه:\n");
  hits.forEach((h) => console.error("   " + h));
  console.error(`\n   ${hits.length} موضعاً. استورد api من عميل بوّابتك:`);
  console.error("     await api(\"/api/v1/...\", { method: \"POST\", body: fd });");
  console.error("   وللتنزيل: apiFile — تردّ الاستجابةَ خاماً وتجدّد التوكنَ عند 401.");
  console.error("   وهي تعرف FormData فتترك ترويسةَ النوع للمتصفّح، وتُلحق الرمز.");
  process.exit(1);
}
console.log(`نداءاتُ المحرّك كلُّها تمرّ بالعميل — فُحص ${files.length} ملفّاً.`);
void relative;
