/**
 * **حارسُ اسمِ المنصة: نصٌّ فيه `{platform}` لا يُطبع خامّاً.**
 *
 * (قاعدةُ المالك: «لا أريد أن تكتب اسمَ المنصة بأيّ مكانٍ أبداً» — ثمّ
 *  ٢٠٢٦-٠٨-٠٨ بلقطةِ فاتورةٍ مطبوعة: «الملفُّ الذي يفتح عند الطباعة ما له
 *  علاقةٌ بالمحتوى المطلوب».)
 *
 * # ولماذا حارسٌ لا إصلاح
 *
 * **أُصلح هذا مرّةً** (`b711cad`: «اسمُ المنصة يُحقَن ولا يُكتب — ١٨ نصّاً
 * و٥ عناوين») **وبقيت الفاتورةُ وكشفُ الحساب.** فالفاتورةُ تُطبع وتُسلَّم
 * للزبون **وفيها `{platform}` حرفاً حرفاً** — وهي آخرُ ورقةٍ يُفترض أن
 * يخطئ فيها اسمُ المنصة.
 *
 * **وما فات مرّةً يفوت ثانية**: المعجمُ فيه اليوم أحدَ عشرَ نصّاً بالعلامة،
 * **ومن أضاف ثاني عشرَ لا شيءَ يذكّره أن يحقنَه.**
 *
 * # وكيف يُقاس
 *
 * تُجمع أسماءُ المفاتيح التي **كلُّ قيمِها في المعجم فيها `{platform}`** —
 * فاسمٌ كـ`footer` لا يُحرَس إلّا إذا كان كلُّ `footer` في المعجم موسوماً.
 * **ثمّ يُبحث في JSX عن `{شيء.ذاك_المفتاح}` بلا `withPlatform`.**
 *
 * **ولا يُحرَس ما لا يُطبع**: `title`/`aria-label`/`alt` وسوائرُها تمرّ إن
 * وُسمت `@platform-ok` فوقها بسبب.
 */
import { readFileSync, readdirSync, statSync } from "node:fs";
import { join, relative } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = join(fileURLToPath(new URL(".", import.meta.url)), "..");
const dict = JSON.parse(readFileSync(join(ROOT, "packages/i18n/src/locales/ar.json"), "utf8"));

/** كلُّ ورقةٍ في المعجم: اسمُ المفتاح ← أفيه العلامة؟ */
const marked = new Map();
(function walk(node) {
  for (const [k, v] of Object.entries(node)) {
    if (typeof v === "string") {
      const has = v.includes("{platform}");
      marked.set(k, (marked.get(k) ?? true) && has);
      if (!marked.has(k)) marked.set(k, has);
      if (!has) marked.set(k, false);
    } else if (v && typeof v === "object") walk(v);
  }
})(dict);

const KEYS = new Set([...marked].filter(([, all]) => all).map(([k]) => k));
if (KEYS.size === 0) {
  console.log("لا نصَّ فيه {platform} — لا شيءَ يُحرَس.");
  process.exit(0);
}

const files = [];
function collect(dir) {
  for (const e of readdirSync(dir)) {
    if (e === "node_modules" || e === ".next" || e === "dist" || e === ".turbo") continue;
    const p = join(dir, e);
    if (statSync(p).isDirectory()) collect(p);
    else if (p.endsWith(".tsx") || p.endsWith(".ts")) files.push(p);
  }
}
collect(join(ROOT, "apps"));
collect(join(ROOT, "packages"));

const bad = [];
const use = new RegExp(`\\b([A-Za-z_$][\\w$.]*)\\.(${[...KEYS].join("|")})\\b`, "g");
for (const f of files) {
  const src = readFileSync(f, "utf8");
  const lines = src.split("\n");
  for (const mm of src.matchAll(use)) {
    const at = src.slice(0, mm.index);
    const ln = at.split("\n").length;
    const line = lines[ln - 1] ?? "";
    /* **الحقنُ قد يقع على السطر أو حولَه** — `withPlatform(V.footer, name)`
       أو متغيّرٌ يُبنى منه. فيُنظر في السطر وسطرٍ قبلَه. */
    const around = (lines[ln - 2] ?? "") + line + (lines[ln] ?? "");
    if (around.includes("withPlatform")) continue;
    if (/@platform-ok/.test((lines[ln - 2] ?? "") + (lines[ln - 3] ?? ""))) continue;
    /* تعريفُ المعجم نفسِه ليس استعمالاً */
    if (f.includes("locales")) continue;
    bad.push(`${relative(ROOT, f)}:${ln}  ${mm[0]}  —  ${line.trim().slice(0, 70)}`);
  }
}

if (bad.length) {
  console.log(`اسمُ المنصة يُطبع خامّاً في ${bad.length} موضعاً — {platform} يظهر للمستخدم:\n`);
  for (const b of bad) console.log("  " + b);
  console.log("\nالعلاج: withPlatform(النصّ, اسمُ المنصة) — أو @platform-ok فوقَه بسببٍ إن كان لا يُعرض.");
  process.exit(1);
}
console.log(`اسمُ المنصة محقونٌ في كلّ موضع — ${KEYS.size} مفتاحاً موسوماً.`);
