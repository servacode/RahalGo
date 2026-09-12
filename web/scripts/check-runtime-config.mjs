#!/usr/bin/env node
/**
 * ══════════════════════════════════════════════════════════════════════
 *  **تهيئةُ البيئة وقتَ التشغيل — حارسٌ دائم** (دورةُ ٧١و)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # لماذا حارسٌ لا ملاحظةٌ في مراجعة
 *
 * **وسطرٌ واحدٌ يعيد `process.env.NEXT_PUBLIC_API_URL` يمرّ في مراجعةٍ
 * عجلى** — **وأثرُه أنّ الحزمةَ تُخبَز فيها بيئةٌ بعينها**، **فتعود
 * صورةُ التجهيز غيرَ صورة الإنتاج، وما يُختبَر هنا لا يُرقَّى هناك.**
 *
 * **ولا يظهر العطبُ إلّا حين يفتح أحدٌ شاشةً في البيئة الخطأ.**
 *
 * # وما يُحرَس
 *
 *   ١ · لا `process.env.NEXT_PUBLIC_*` في شيفرةٍ تُبنى
 *   ٢ · ولا حقنُ `env:` في `next.config`
 *   ٣ · والعقدُ مركزيٌّ موجود
 *   ٤ · ومسارُ التهيئة لا يُخزَّن ولا يُولَّد ساكناً
 *   ٥ · ولا سرَّ يدخل تهيئةً عامّة
 *   ٦ · ولا عنوانَ بيئةٍ مكتوبٌ في الشيفرة
 */
import { readFileSync, readdirSync, statSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const web = join(here, "..");
const problems = [];
const notes = [];

function read(p) {
  try {
    return readFileSync(p, "utf8");
  } catch {
    return "";
  }
}

/** **يمشي الشجرةَ ويجمع ملفّاتِ المصدر** — بلا `node_modules` و`.next`. */
function sources(dir, out = []) {
  let entries;
  try {
    entries = readdirSync(dir);
  } catch {
    return out;
  }
  for (const name of entries) {
    if (name === "node_modules" || name === ".next" || name === "dist") continue;
    const full = join(dir, name);
    if (statSync(full).isDirectory()) sources(full, out);
    else if (/\.(ts|tsx|js|mjs)$/.test(name) && !/\.test\./.test(name)) out.push(full);
  }
  return out;
}

const files = [
  ...sources(join(web, "apps", "rahalgo", "src")),
  ...sources(join(web, "packages", "ui", "src")),
  ...sources(join(web, "packages", "auth", "src")),
];

// ── ١ · لا قراءةَ مباشرةً لمتغيّرٍ عامٍّ وقتَ البناء ──────────────────
//
// **و`NEXT_PUBLIC_*` تُستبدَل نصّاً في الحزمة** — **فهي هويّةُ بيئةٍ
// مخبوزة، لا تهيئةُ تشغيل.**
const direct = [];
for (const f of files) {
  const src = read(f);
  if (/process\.env\.NEXT_PUBLIC_[A-Z0-9_]+/.test(src)) {
    direct.push(f.replace(web + "\\", "").replace(web + "/", ""));
  }
}
if (direct.length > 0) {
  problems.push(
    "قراءةٌ مباشرةٌ لمتغيّرٍ يُخبَز وقتَ البناء — والعقدُ المركزيُّ هو السبيل:\n   " +
      direct.join("\n   "),
  );
} else {
  notes.push(`لا قراءةَ مخبوزةً — ${files.length} ملفَّ مصدرٍ فُحص`);
}

// ── ٢ · ولا حقنَ في إعداد البناء ────────────────────────────────────
const nextCfg = read(join(web, "apps", "rahalgo", "next.config.ts"));
if (/^\s*env\s*:/m.test(nextCfg)) {
  problems.push("next.config يحقن `env:` — وهو ما يخبز البيئةَ في الحزمة");
}

// ── ٣ · والعقدُ المركزيُّ قائمٌ بعناصره ──────────────────────────────
const contract = read(join(web, "apps", "rahalgo", "src", "lib", "config.ts"));
for (const needed of ["PublicConfig", "readServerConfig", "missingConfig", "CONFIG_GLOBAL"]) {
  if (!contract.includes(needed)) {
    problems.push(`عقدُ التهيئة فقد عنصراً: ${needed}`);
  }
}
const shared = read(join(web, "packages", "ui", "src", "runtimeconfig.ts"));
// **والتعليقُ يُطوى قبل الفحص** — **وإلّا أسقط الحارسَ شرحٌ يذكر ما
// يمنعه.** (وقع أوّلَ تشغيلٍ لهذا الحارس.)
const sharedCode = shared.replace(/\/\*[\s\S]*?\*\//g, "").replace(/\/\/.*$/gm, "");
if (!sharedCode.includes("CONFIG_GLOBAL") || /process\.env/.test(sharedCode)) {
  problems.push("قارئُ الحِزَم ليس على العقد نفسِه — أو يقرأ بيئةَ بناء");
}

// ── ٤ · ومسارُ التهيئة لا يُخزَّن ولا يُولَّد ساكناً ─────────────────
//
// **وتهيئةٌ تُخزَّن تصير تهيئةَ بيئةٍ أخرى بعد نشر.**
const routeRaw = read(join(web, "apps", "rahalgo", "src", "app", "config.js", "route.ts"));
// **والتعليقُ يُطوى قبل كلّ فحص** — **وشرحٌ يذكر `no-store` كان يُمرّر
// ملفّاً لا يحمله.** (قِيس بشاهدٍ سالبٍ أوّلَ مرّة.)
const route = routeRaw.replace(/\/\*[\s\S]*?\*\//g, "").replace(/\/\/.*$/gm, "");
if (!routeRaw) {
  problems.push("لا مسارَ تهيئةٍ — والمتصفّحُ لا بيئةَ له");
} else {
  if (!route.includes("force-dynamic")) {
    problems.push("مسارُ التهيئة قد يُولَّد ساكناً — فتُخبَز فيه بيئةُ البناء");
  }
  if (!/no-store/.test(route)) {
    problems.push("مسارُ التهيئة يُخزَّن — فتُخدَم تهيئةُ بيئةٍ أخرى بعد نشر");
  }
  if (!route.includes("missingConfig")) {
    problems.push("مسارُ التهيئة لا يُعلن النقص — والفشلُ يجب أن يكون مغلقاً");
  }
}

// ── ٥ · ولا سرَّ في تهيئةٍ عامّة ─────────────────────────────────────
//
// **وتهيئةُ التشغيل العامّةُ عامّةٌ بالتعريف** — يقرؤها كلُّ من فتح
// الصفحة. **فاسمٌ يوحي بسرٍّ يُسقط الفحصَ قبل أن يُنشَر.**
const SECRETISH = /(SECRET|PASSWORD|CREDENTIAL|PRIVATE|JWT|_KEY\b|TOKEN)/;
for (const [label, src] of [["العقد", contract], ["قارئُ الحِزَم", shared], ["المسار", routeRaw]]) {
  const body = src.replace(/\/\*[\s\S]*?\*\//g, "").replace(/\/\/.*$/gm, "");
  const envs = body.match(/RAHALGO_[A-Z0-9_]+/g) ?? [];
  for (const name of envs) {
    if (SECRETISH.test(name)) {
      problems.push(`${label}: اسمٌ يوحي بسرٍّ في تهيئةٍ عامّة — ${name}`);
    }
  }
}

// ── ٦ · ولا عنوانَ بيئةٍ مكتوبٌ في الشيفرة ───────────────────────────
//
// **وعنوانٌ مكتوبٌ يجعل الأثرَ مربوطاً ببيئةٍ ولو زالت المتغيّرات.**
const BOUND = /(api\.rahalgo\.com|staging\.rahalgo\.com)/;
const bound = [];
for (const f of files) {
  const body = read(f).replace(/\/\*[\s\S]*?\*\//g, "").replace(/\/\/.*$/gm, "");
  if (BOUND.test(body)) bound.push(f.replace(web + "\\", "").replace(web + "/", ""));
}
if (bound.length > 0) {
  problems.push("عنوانُ بيئةٍ مكتوبٌ في الشيفرة:\n   " + bound.join("\n   "));
}

// ── ٧ · وملفّا SEO يُولَّدان عند الطلب ───────────────────────────────
//
// **و`robots.ts` و`sitemap.ts` لهما توليدُهما الخاصّ** — **لا يخضعان
// لـ`force-dynamic` التخطيطِ الجذريّ.** فمرّا في ٧١و مخبوزين: خرج
// `Sitemap: /sitemap.xml` نسبيّاً و`<loc>` فارغةً، **لأنّ هويّةَ
// البيئة خاليةٌ وقتَ البناء.** (عطبُ ٧١و-ر١، أُصلح في ٧١و-ر٢.)
//
// **و`robots.ts` لا `fetch` فيه** — فلا إعادةَ توليدٍ تشفيه: الجسمُ
// المخبوزُ يُخدَم إلى آخر عمر الأثر.
for (const name of ["robots.ts", "sitemap.ts"]) {
  const raw = read(join(web, "apps", "rahalgo", "src", "app", name));
  if (!raw) {
    problems.push(`ملفُّ ${name} مفقود — وخريطةُ الموقعِ وقواعدُ الزحف جزءٌ من العقد`);
    continue;
  }
  // **ويُطوى التعليقُ قبل الفحص** — **وشرحُ هذا الحارسِ نفسِه يذكر
  // `force-dynamic`**، فلولا الطيُّ لمرّ ملفٌّ لا يحمله.
  const body = raw.replace(/\/\*[\s\S]*?\*\//g, "").replace(/\/\/.*$/gm, "");
  if (!/export\s+const\s+dynamic\s*=\s*["']force-dynamic["']/.test(body)) {
    problems.push(`${name} قد يُولَّد ساكناً — فتُخبَز فيه هويّةُ بيئةِ البناء`);
  }
  if (!body.includes("readServerConfig")) {
    problems.push(`${name} لا يقرأ العقدَ المركزيّ — وهويّةُ البيئة تُقرأ عند الطلب`);
  }
  if (/process\.env/.test(body)) {
    problems.push(`${name} يقرأ بيئةَ العمليّة مباشرةً — والعقدُ المركزيُّ هو السبيل`);
  }
  // **وخزنُ الجلب يناقض توليدَ الطلب** — أثرٌ واحدٌ يخدم بيئتين،
  // **فجسمٌ محفوظٌ من بيئةٍ قد يُخدَم في أخرى.**
  if (/next\s*:\s*\{[^}]*revalidate/.test(body)) {
    problems.push(`${name} يخزّن جلبَه بمدّة — وذاك يناقض توليدَ الطلب`);
  }
  // **وجلبٌ عند الطلب بلا مهلةٍ يورث كلفةَ المحرّك لكلّ زائر** —
  // **والافتراضيّةُ عشرُ ثوانٍ.** (قِيس ١٠٫٥ ثانيةً لـ`/sitemap.xml`
  // على التجهيز قبل أن تُحَدّ المهلة.)
  if (/await fetch\(/.test(body) && !/AbortSignal\.timeout|signal\s*:/.test(body)) {
    problems.push(`${name} يجلب عند الطلب بلا مهلةٍ — ومحرّكٌ لا يُجاب يكلّف عشرَ ثوانٍ لكلّ زائر`);
  }
}
if (problems.length === 0) {
  notes.push("robots و sitemap يُولَّدان عند الطلب — لا هويّةَ بيئةٍ مخبوزة");
}

if (problems.length > 0) {
  console.error("تهيئةُ التشغيل — خلل:");
  for (const p of problems) console.error("  ✗ " + p);
  process.exit(1);
}
for (const n of notes) console.log("  · " + n);
console.log("تهيئةُ البيئة وقتَ التشغيل — أثرٌ واحدٌ يصلح لكلّ بيئة.");
