#!/usr/bin/env node
/**
 * ══════════════════════════════════════════════════════════════════════
 *  **شاشةُ مراقبة التشغيل — قراءةٌ محضةٌ ببابٍ من قدرة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ما تحرسه
 *
 *	١ · **بابُ القائمة بالقدرة لا باسم الدور** — `observability.read`
 *	٢ · **ولا اسمَ دورٍ في شرط الظهور** — `role === "observability"`
 *	    يُسقط البناء
 *	٣ · **ولا زرَّ يُبدّل شيئاً** — لا `POST` ولا `PATCH` ولا `DELETE`
 *	    ولا «إعادة تشغيل» ولا «مسح» ولا «إصلاح»
 *	٤ · **والصفحةُ تستهلك الباب القائم وحدَه** — `/admin/ops/health`
 *	٥ · **ولا حقلٌ يُخترَع**: كلُّ حقلٍ تقرؤه الصفحةُ موجودٌ في عقد
 *	    المحرّك (`ops_health.go`)
 *	٦ · **ولا سرٌّ يُعرَض**: هاتفٌ ولا رمزُ جلسةٍ ولا بصمةٌ ولا موقع
 *	٧ · **والتحديثُ محافظ** — لا نبضَ تحت الدقيقة
 *	٨ · **وحالاتُ العطب الثلاثُ مكتوبةٌ**: ٤٠٣ · ٤٠١ · انقطاع
 *
 * # ولماذا حارسٌ لا مراجعة
 *
 * **وزرُّ «إعادة تشغيل» يُضاف في سطرٍ واحدٍ يومَ يضيق أحدُهم بالعطب** —
 * **ولا يُكتشف إلّا بعد أن يُضغط في الإنتاج.**
 */
import { readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const web = join(here, "..");
const root = join(web, "..");
const problems = [];
const notes = [];

const read = (p) => {
  try {
    return readFileSync(p, "utf8");
  } catch {
    return "";
  }
};
const strip = (s) => s.replace(/\/\*[\s\S]*?\*\//g, "").replace(/\/\/.*$/gm, "");

const pagePath = join(web, "apps", "rahalgo", "src", "app", "dashboard", "ops", "page.tsx");
const layoutPath = join(web, "apps", "rahalgo", "src", "app", "dashboard", "layout.tsx");
const rawPage = read(pagePath);
const page = strip(rawPage);
const layout = strip(read(layoutPath));

if (page === "") problems.push("صفحةُ مراقبة التشغيل لم تُقرأ — والحارسُ بلا مرجع");
if (layout === "") problems.push("تخطيطُ اللوحة لم يُقرأ");

// ── ١ · بابُ القائمة بالقدرة ─────────────────────────────────────────
{
  if (!/caps:\s*\["observability\.read"\]/.test(layout)) {
    problems.push(
      "بندُ «مراقبة التشغيل» غيرُ مبوَّبٍ بالقدرة — " +
        "**فدورٌ يُمنَح `observability.read` غداً لا يرى بابَه**",
    );
  }
  if (!/const can = \(c: string\) => caps\.includes\(c\)/.test(layout)) {
    problems.push("الترشيحُ لا يقرأ القدرات");
  }
  // ── ٢ · ولا اسمَ دورٍ في الشرط ─────────────────────────────────────
  if (/roles:\s*\[[^\]]*"observability"/.test(layout)) {
    problems.push(
      "**البندُ مبوَّبٌ باسم الدور `observability`** — " +
        "**وشرطُ المالك أن يكون بالقدرة** (بندُ ٢)",
    );
  }
  if (/=== ?"observability"|includes\("observability"\)/.test(layout + page)) {
    problems.push("**شرطُ ظهورٍ يقارن اسمَ الدور** — والقدرةُ هي المرجع");
  }
  if (problems.length === 0) notes.push("بابُ القائمة بالقدرة — ولا اسمَ دورٍ في شرطه");
}

// ── ٣ · ولا فعلَ يُبدّل شيئاً ────────────────────────────────────────
{
  const mutating = [
    /method:\s*["'](POST|PUT|PATCH|DELETE)["']/i,
    /\bapi\([^)]*,\s*\{\s*method/i,
  ];
  for (const re of mutating) {
    const hit = page.match(re);
    if (hit) {
      problems.push(
        `**نداءٌ يُبدّل في شاشةِ قراءة**: ${hit[0]} — ` +
          "**وهي تُراقب وتُشخّص وتُبلّغ، ولا تفعل.**",
      );
    }
  }
  // **وألفاظُ الفعل تُمنَع نصّاً كذلك** — زرٌّ يُضاف غداً باسمه.
  for (const word of [
    "إعادة تشغيل",
    "أعد التشغيل",
    "مسح الذاكرة",
    "إفراغ",
    "إصلاح",
    "تشغيل الهجرة",
    "نشر",
    "تراجع",
    "إيقاف",
  ]) {
    if (rawPage.includes(word)) {
      problems.push(`**لفظُ فعلٍ في شاشة قراءة**: «${word}»`);
    }
  }
}

// ── ٤ · وتستهلك البابَ القائمَ وحدَه ─────────────────────────────────
{
  if (!page.includes("/api/v1/admin/ops/health")) {
    problems.push("الصفحةُ لا تنادي `/api/v1/admin/ops/health`");
  }
  const calls = [...page.matchAll(/api<[^>]*>\(\s*"([^"]+)"/g)].map((m) => m[1]);
  const allowed = new Set(["/api/v1/admin/ops/health", "/api/v1/public/identity"]);
  for (const c of calls) {
    if (!allowed.has(c)) {
      problems.push(`**بابٌ ثالثٌ في شاشة الصحّة**: ${c} — ولا حسابَ صحّةٍ ثانٍ`);
    }
  }
  notes.push(`أبوابٌ تناديها الصفحة: ${[...new Set(calls)].join(" · ")}`);
}

// ── ٥ · ولا حقلٌ يُخترَع ─────────────────────────────────────────────
{
  const contract = read(join(root, "backend", "internal", "server", "ops_health.go"));
  if (contract === "") {
    problems.push("عقدُ المحرّك لم يُقرأ — والحارسُ بلا مرجع");
  } else {
    const fields = new Set(
      [...contract.matchAll(/json:"([a-z0-9_]+)"/g)].map((m) => m[1]),
    );
    // **وحقولُ الهويّة من بابها** — بيئةٌ وهجرةٌ ليستا في عقد الصحّة.
    for (const f of ["environment", "migration_version", "source_commit", "build_id"]) {
      fields.add(f);
    }
    // **وما تقرؤه الصفحةُ من الجسم** — `x?.field` أو `x.field`.
    const readFields = new Set(
      [...page.matchAll(/\b(?:health|rt|pg|pool|rd|ident)\??\.\s*([a-z0-9_]{3,})/g)].map(
        (m) => m[1],
      ),
    );
    const invented = [...readFields].filter((f) => !fields.has(f)).sort();
    if (invented.length > 0) {
      problems.push(
        "**حقولٌ تقرؤها الشاشةُ وليست في عقد المحرّك**:\n   " +
          invented.join(" · ") +
          "\n   **ورقمٌ يُخترَع في شاشةِ صحّةٍ أسوأُ من غيابه.**",
      );
    } else {
      notes.push(`${readFields.size} حقلاً تقرؤه الشاشةُ — كلُّها في العقد`);
    }
  }
}

// ── ٦ · ولا سرَّ يُعرَض ──────────────────────────────────────────────
//
// **ومفاتيحُ المعجم تُطوى قبل الفحص** — **و`O.refresh` زرُّ «تحديث»
// لا رمزُ تجديدِ جلسة.** (أنذر كذباً أوّلَ مرّة، **وحارسٌ يُنذر كذباً
// يُطفأ.**)
//
// **ويُقسَم النصُّ رموزاً ولا يُبحَث بتعبيرٍ نمطيّ** — **و`\\b` داخلَ
// قالبٍ نصّيٍّ محرفُ تراجعٍ لا حدُّ كلمة**، **فالقاعدةُ تمرّ أبداً**
// وهي لا تقيس شيئاً. (وقع، وكشفه شاهدُه السالب.)
{
  const body = page.replace(/O\.[A-Za-z0-9_]+/g, " ");
  const tokens = new Set(body.split(/[^A-Za-z0-9_]+/).map((t) => t.toLowerCase()));
  for (const secret of [
    "password",
    "access_token",
    "refresh_token",
    "session_id",
    "phone",
    "pin_hash",
    "secret",
    "database_url",
    "redis_url",
    "latitude",
    "longitude",
  ]) {
    if (tokens.has(secret)) {
      problems.push(`**حقلٌ حسّاسٌ في شاشةِ صحّة**: ${secret}`);
    }
  }
  notes.push(`رموزٌ فُحصت = ${tokens.size}`);
}

// ── ٧ · وتحديثٌ محافظ ────────────────────────────────────────────────
{
  const iv = page.match(/setInterval\([^,]+,\s*([0-9_]+)\)/);
  if (!iv) {
    notes.push("لا تحديثَ تلقائيّ — اليدويُّ وحدَه");
  } else {
    const ms = Number(iv[1].replace(/_/g, ""));
    if (ms < 60000) {
      problems.push(
        `**نبضٌ كلَّ ${ms} مللي ثانية** — **وقراءةُ الصحّة تمسّ القاعدةَ والذاكرةَ في كلّ نداء**`,
      );
    } else {
      notes.push(`تحديثٌ تلقائيٌّ كلَّ ${ms / 1000} ثانية`);
    }
  }
}

// ── ٨ · وحالاتُ العطب الثلاث ─────────────────────────────────────────
{
  // **ولا يكفي وجودُ الرقم في الملفّ** — **`code === 403` موجودةٌ في
  // موضعٍ آخرَ فتمرّ القاعدةُ وهي لا تقيس شيئاً.** (كشفه شاهدُها
  // السالب.) **فيُربَط كلُّ حالٍ برسالتها بعينها.**
  // **ويُربَط الفرعُ برسالته بالموضع لا بالوجود** — **و`O.errForbidden`
  // موجودةٌ في كتلة المنع المسبق، فوجودُها في الملفّ لا يعني أنّ فرعَ
  // ٤٠٣ يقولها.** (كشفه شاهدُه السالبُ مرّتين.)
  const near = (anchor, want, span = 140) => {
    const i = page.indexOf(anchor);
    return i >= 0 && page.slice(i, i + span).includes(want);
  };
  for (const [anchor, msg, what] of [
    ["code === 403", "errForbidden", "ممنوع"],
    ["code === 401", "errUnauthorized", "جلسةٌ منتهية"],
  ]) {
    if (!near(anchor, msg)) {
      problems.push(`**فرعُ ${what} لا يقول رسالتَه**: ${anchor} ⇒ ${msg}`);
    }
  }
  if (!page.includes("O.errUnavailable")) {
    problems.push("**حالُ الانقطاع غيرُ مكتوبة**");
  }
  // **والمنعُ يُقال قبل النداء كذلك** — من لا قدرةَ له يقرأ سببَه.
  if (!/capabilities\.includes\("observability\.read"\)/.test(page)) {
    problems.push("**الصفحةُ لا تقرأ القدرةَ** — فلا تقول «ليس لديك صلاحية» لمن لا يملكها");
  }
  // **ولا تُطوى الشاشةُ لتدهّورِ نظامٍ فرعيّ** — الجسمُ القديمُ يبقى.
  if (!/setError\(""\)/.test(page)) {
    problems.push("الخطأُ لا يُمحى عند نجاحٍ لاحق — فيبقى معروضاً بعد التعافي");
  }
}

// ── ٩ · وصاحبُ القدرةِ يرى بابَه وحدَه ─────────────────────
//
// **وحسابُ الرصد رأى تسعةَ أبوابٍ لا يملك فيها زرّاً** — الطلباتُ
// والشكاوى والطوارئُ والإعدادات — **لأنَّ بنداً بلا شرطٍ كان يظهر
// للكلّ.** (قِيس في متصفّحٍ على التجهيز ٢٠٢٦-٠٩-١٢.)
//
// **وصارت الاثنتان والعشرون كلُّها بالقدرة** (٢٠٢٦-٠٩-١٣) —
// **وتفصيلُها في `check-capability-gates.mjs`.** **والمقيسُ هنا
// أنّ بندَ المراقبة لم يُستثنَ من القاعدة**، **وأنّ من هبط على
// بابٍ لا يملكه يُنزَل على أوّلِ ما يملك.**
{
  const lay = layout.replace(/\s+/g, " ");
  if (!lay.includes("return ALL_NAV.filter((i) => !!i.caps && i.caps.some(can));")) {
    problems.push(
      "**ترشيحُ القائمة لم يعد بالقدرة وحدَها** — " +
        "**وبندٌ بلا شرطٍ يظهر لمن لا يفتحه** (`R-34`)",
    );
  }
  if (!lay.includes("landed.current = true; router.replace(first.href);")) {
    problems.push("**لا هبوطَ على أوّل بابٍ مملوك** — فصاحبُ القدرةِ يرى «الرئيسيّة» تُردّ ٤٠٣");
  }
  if (problems.length === 0) notes.push("بندُ المراقبة داخلَ قاعدة القدرات — والهبوطُ على أوّل بابٍ مملوك");
}

if (problems.length > 0) {
  console.error("شاشةُ مراقبة التشغيل — خلل:");
  for (const p of problems) console.error("  ✗ " + p);
  process.exit(1);
}
for (const n of notes) console.log("  · " + n);
console.log("شاشةُ المراقبة قراءةٌ محضة · بابُها قدرةٌ · ولا حقلَ مخترَع.");
