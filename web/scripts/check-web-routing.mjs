/**
 * ══════════════════════════════════════════════════════════════════════
 * **بابُ الويب بقدرةٍ لا باسم دور** (`WEBA`، ٢٠٢٦-٠٩-١٣)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ما وقع
 *
 * **حسابُ الرصد `customer + observability` دخل من `/adminrahalgo`
 * فسيق إلى `/app`** — **وقدرتُه `observability.read` قدرةُ ويبٍ
 * شرعيّة.** **والجذرُ أنّ القدراتَ لم تُسأل**: القرارُ كان قائمةَ
 * ثلاثةِ أسماءٍ — `admin · ops · finance` — **فيها دورٌ متقاعدٌ
 * وليس فيها الدورانِ القائمان.**
 *
 * # ولمَ يُنادى الكودُ ولا يُقرأ نصُّه
 *
 * **قراءةُ سطرٍ تُثبت أنّه مكتوبٌ لا أنّه يحكم** — **وشرطٌ مقلوبٌ
 * يمرّ القراءةَ ويسقط في الميدان.** **فتُنادى `isWebAuthorized`
 * و`homeFor` بعشر حالاتٍ وتُقاس وجهتُها.**
 *
 * **ولا مطرحَ ثانياً للقرار**: **يُقاس أنّ بابَ اللوحة والتوجيهَ
 * يقرآن الدالّةَ نفسَها** — **وقاعدتان تفترقان يومَ تُبدَّل إحداهما،
 * وقد افترقتا فعلاً يوماً كاملاً.**
 */
import { readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { register } from "node:module";

const here = dirname(fileURLToPath(import.meta.url));
const web = join(here, "..");

// **ومُحمِّلُ الـTS نفسُه الذي يستعمله حارسُ التأكيد** — `_tsload.mjs`.
register(pathToFileURL(join(here, "_tsload.mjs")).href);

// ── مِسندُ متصفّحٍ أصغرُ ما يكفي ─────────────────────────────────────
//
// **وعميلُ الـAPI يقرأ التخزينَ والنافذةَ عند التحميل** — **ولا يُبدَّل
// المنتَجُ لأجل حارس.**
const mem = () => {
  const s = new Map();
  return {
    getItem: (k) => (s.has(k) ? s.get(k) : null),
    setItem: (k, v) => s.set(k, String(v)),
    removeItem: (k) => s.delete(k),
    clear: () => s.clear(),
  };
};
globalThis.localStorage = mem();
globalThis.sessionStorage = mem();
globalThis.window = globalThis;
globalThis.__RAHALGO_CONFIG__ = Object.freeze({ apiUrl: "https://api.test" });

const { homeFor, PANEL_PATHS } = await import(
  pathToFileURL(join(web, "packages/auth/src/routing.ts")).href
);

const DASH = PANEL_PATHS.admin; // "/dashboard"
const APP = PANEL_PATHS.app; // "/app"

let bad = 0;
const fail = (msg) => {
  console.error(`  ✗ ${msg}`);
  bad++;
};

/** case يقيس وجهةَ حسابٍ بحاله كما هو. */
function dest(roles, caps, capsLoaded = true) {
  return homeFor(roles, caps, capsLoaded).path;
}

// ── ١ · زبونٌ وحدَه بلا قدرة ⇒ التطبيق ──────────────────────────────
if (dest(["customer"], []) !== APP)
  fail("١ · زبونٌ بلا قدرةٍ لم يُساق إلى باب التطبيقات");

// ── ٢ · زبونٌ + رصدٌ بقدرةِ قراءة ⇒ اللوحة ───────────────────────────
if (dest(["customer", "observability"], ["observability.read"]) !== DASH)
  fail("٢ · `customer + observability.read` لم يدخل اللوحة — **وهو العطبُ بعينه**");

// ── ٣ · وترتيبُ الأدوار لا يقرّر شيئاً ───────────────────────────────
if (dest(["observability", "customer"], ["observability.read"]) !== DASH)
  fail("٣ · ترتيبُ الأدوار غيّر الوجهة — **والقرارُ بالقدرة لا بالترتيب**");

// ── ٤ · العمليّاتُ بقدراتها ⇒ اللوحة ────────────────────────────────
const OPS = ["orders.read", "orders.intervene", "drivers.manage"];
if (dest(["customer", "operations"], OPS) !== DASH)
  fail("٤ · حسابُ العمليّات لم يدخل اللوحة");

// ── ٥ · الماليّةُ بقدراتها ⇒ اللوحة ─────────────────────────────────
const FIN = ["finance.read", "finance.manage", "payouts.decide"];
if (dest(["customer", "finance"], FIN) !== DASH)
  fail("٥ · حسابُ الماليّة لم يدخل اللوحة");

// ── ٦ · المالكُ والأدمن ⇒ اللوحة ────────────────────────────────────
if (dest(["admin"], ["analytics.read", "orders.read"]) !== DASH)
  fail("٦ · الأدمن لم يدخل اللوحة");
if (dest(["customer", "owner_super_admin"], ["roles.manage"]) !== DASH)
  fail("٦ · المالكُ لم يدخل اللوحة");

// ── ٧ · أدوارُ الميدان بلا قدرةٍ ⇒ التطبيق ──────────────────────────
for (const r of ["driver", "merchant", "sales"]) {
  if (dest(["customer", r], []) !== APP)
    fail(`٧ · صاحبُ دور ${r} بلا قدرةٍ لم يُساق إلى باب التطبيقات`);
}

// ── ٨ · اسمُ دورٍ حاضرٌ والقدرةُ غائبةٌ ⇒ لا يدخل ────────────────────
//
// **وهذه هي التي تمنع أن يعود الاسمُ حاكماً من الخلف**: **دورُ
// `observability` بلا قدراتٍ واصلةٍ لا يفتح اللوحة.**
if (dest(["customer", "observability"], []) !== APP)
  fail("٨ · اسمُ الدور وحدَه فتح اللوحةَ — **والقدرةُ هي الحاكمة**");
if (dest(["customer", "operations"], []) !== APP)
  fail("٨ · اسمُ `operations` وحدَه فتح اللوحة");

// ── ٩ · دورٌ مخصَّصٌ بقدرةٍ شرعيّةٍ ⇒ يعمل بلا سطرٍ يُكتب له ─────────
if (dest(["customer", "night_desk_2027"], ["orders.read"]) !== DASH)
  fail("٩ · دورٌ مخصَّصٌ بقدرةٍ لم يدخل — **فسيُكتب اسمُه في الشيفرة، وهو ما نمنعه**");

// ── ١٠ · وشبكةُ الأمان: قدراتٌ لم تصل ──────────────────────────────
//
// **ولا يُحبَس الأدمنُ خارجَ لوحته لأنّ نداءً سقط** — **وتُقرأ
// الأسماءُ القديمةُ في هذه الحال وحدَها.**
if (dest(["admin"], [], false) !== DASH)
  fail("١٠ · الأدمن حُبس خارجَ لوحته حين تعذّرت قراءةُ القدرات");
if (dest(["customer"], [], false) !== APP)
  fail("١٠ · زبونٌ دخل اللوحةَ بحجّة تعذّر القراءة");
// **والرصدُ ليس في القائمة القديمة** — **فيُساق إلى التطبيق حين لا
// تُقرأ قدراتُه**، وهو الحدُّ المقصود: **شبكةُ أمانٍ لا بابٌ ثانٍ.**
if (dest(["customer", "observability"], [], false) !== APP)
  fail("١٠ · الرصدُ دخل بشبكة الأمان — **وهي للأسماء القديمة وحدَها**");

// ══════════════════════════════════════════════════════════════════════
// **وموضعُ القرار واحدٌ — يُقاس بالنصّ لأنّه بنيةٌ لا سلوك**
// ══════════════════════════════════════════════════════════════════════
const panelGate = readFileSync(join(web, "apps/rahalgo/src/lib/auth.tsx"), "utf8");
if (!panelGate.includes("isWebAuthorized"))
  fail("بابُ اللوحة لا يقرأ `isWebAuthorized` — **قاعدةٌ ثانيةٌ ستفترق**");
if (/hasRole\(user,\s*\.\.\.PANEL_ROLES\)/.test(panelGate))
  fail("بابُ اللوحة عاد إلى أسماء الأدوار");

const routing = readFileSync(join(web, "packages/auth/src/routing.ts"), "utf8");
if (!routing.includes("isWebAuthorized"))
  fail("التوجيهُ لا يقرأ `isWebAuthorized`");
if (/has\("admin"\)\s*\|\|\s*has\("ops"\)/.test(routing))
  fail("التوجيهُ عاد إلى قائمة الأسماء الثلاثة");
if (!routing.includes("authApi.capabilities()"))
  fail("التوجيهُ لا يسأل المحرّكَ عن القدرات بعد الدخول");

// **وقائمةُ الأسماء القديمةُ في موضعٍ واحد.**
const provider = readFileSync(join(web, "packages/auth/src/provider.tsx"), "utf8");
if (/PANEL_ROLES:\s*Role\[\]\s*=\s*\[\s*"admin"/.test(provider))
  fail("قائمتان بالأسماء نفسِها — **تفترقان يومَ تُبدَّل إحداهما**");

// **وبوّابةُ الكلمة المؤقّتة تسمع ردَّ المحرّك لا العلَمَ وحدَه.**
const gate = readFileSync(join(web, "packages/auth/src/PasswordGate.tsx"), "utf8");
if (!gate.includes("isPasswordChangeRequired"))
  fail("بوّابةُ التبديل لا تسمع `password_change_required` — **فيُمنَع صاحبُها بلا طريق**");
if (!gate.includes("clearPasswordChangeRequired"))
  fail("الإشارةُ لا تُنسى بعد التبديل — **فتبقى البوّابةُ بعد زوال سببها**");

// ══════════════════════════════════════════════════════════════════════
// **ولا يُحكَم بالغياب قبل وصول القدرات**
// ══════════════════════════════════════════════════════════════════════
//
// **ومن حوّل قبلها قرأ صاحبَ القدرةِ بلا قدرةٍ فساقه إلى `/app`** —
// **وهو العطبُ الذي أُصلح، ويعود بسطرٍ واحدٍ ناقص.**
const portal = readFileSync(
  join(web, "apps/rahalgo/src/app/portal/[[...rest]]/page.tsx"),
  "utf8",
);
if (!/if \(loading \|\| !capsLoaded\) return;/.test(portal))
  fail("صفحةُ `portal` تحوّل قبل وصول القدرات — **فيُساق صاحبُ القدرة إلى `/app`**");
for (const [file, path] of [
  ["Header", "apps/rahalgo/src/components/Header.tsx"],
  ["BottomNav", "apps/rahalgo/src/components/BottomNav.tsx"],
]) {
  const src = readFileSync(join(web, path), "utf8");
  if (/portalFor\(user\.roles\)/.test(src) || /homeFor\(user\.roles\)/.test(src))
    fail(`${file} يقرأ الوجهةَ بالأدوار وحدَها — **بلا قدرات**`);
}

const client = readFileSync(join(web, "packages/auth/src/client.ts"), "utf8");
if (!client.includes("notePasswordChangeRequired"))
  fail("عميلُ الـAPI لا يلتقط الإشارةَ مركزيّاً");

if (bad) {
  console.error(`\n**بابُ الويب مكسور** — ${bad} خرقاً.`);
  process.exit(1);
}
console.log(
  "بابُ الويب بقدرةٍ لا باسم دور — عشرُ حالاتٍ قِيست بالنداء · وموضعُ القرار واحد.",
);
