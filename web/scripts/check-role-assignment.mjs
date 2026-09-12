#!/usr/bin/env node
/**
 * ══════════════════════════════════════════════════════════════════════
 *  **سياسةُ إسناد الأدوار — حارسٌ يُشغّل الشيفرةَ ويقيس القصّ**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ما وقع (٢٠٢٦-٠٩-١٢)
 *
 * **أنشأ المالكُ `observability` في الإنتاج فلم يجده في نافذة أدوار
 * الحساب** — **والنافذةُ لم تكن تُرشِّح، كانت تقصّ.**
 *
 * `Chips` افتراضُها `nowrap` مع `overflow-x-auto` **وشريطُ التمرير
 * مخفيٌّ بالأنماط**، والنافذةُ لم تُمرِّر `wrap`. **فقِيست المستطيلاتُ
 * في متصفّحٍ حقيقيٍّ على التجهيز**:
 *
 *	scrollWidth = 2002  ·  clientWidth = 398
 *	حبّاتٌ = 18  ·  **مرئيّةٌ = 3**
 *	«مراقبة التشغيل» على **-820px** — خارجَ الإطار
 *
 * **وعطبٌ ثانٍ في الاتّجاه المقابل**: `owner_super_admin` كان حبّةً
 * كبقيّتها — **نقرةٌ واحدةٌ تمنح كلَّ قدرةٍ في المعجم.**
 *
 * # وما يُحرَس
 *
 *	١ · **`observability` أهلٌ للإسناد** ويظهر في مجموعة الموظّفين
 *	٢ · و`operations` و`finance` وكلُّ أدوار العمل المعتمدة
 *	٣ · و`owner_super_admin` **لا يُعرَض في المسار العاديّ**
 *	٤ · **ويُعرَض مقفلاً إن كان الحسابُ يملكه** — وإلّا مُنع النزع
 *	٥ · ودورٌ مجهولٌ **لا يكسب أهليّةً صامتة** — يصير `custom` لا `staff`
 *	٦ · **والوجودُ من المحرّك** — رمزٌ ليس في جوابه لا يظهر أبداً
 *	٧ · وكلُّ رمزٍ في هجرات المحرّك **مصنَّفٌ صراحةً** لا يسقط في `custom`
 *	٨ · والأسماءُ عربيّةٌ في كلّ مجموعة، ولا مفتاحَ ترجمةٍ يُعرَض
 *	٩ · **ومصدرُ السياسةِ واحدٌ** — لا مصفوفةَ رموزٍ في شاشة
 *	١٠ · **وكلُّ `Chips` في مسار الأدوار يلتفّ** — وهذا حارسُ القصّ
 *
 * # ولماذا يُشغَّل لا يُقرَأ
 *
 * **وفحصٌ نصّيٌّ يسأل «هل الرمزُ مكتوبٌ في الملفّ؟»** ولا يسأل «هل
 * الدالّةُ تُرجعه في مجموعةٍ صحيحة؟» **فتُستورَد الوحدةُ ويُنادى ما
 * فيها.**
 */
import { readFileSync, readdirSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { register } from "node:module";

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

/** **يطوي التعليقَ قبل الفحص** — وشرحٌ يذكر رمزاً ليس استعمالاً له. */
const strip = (s) => s.replace(/\/\*[\s\S]*?\*\//g, "").replace(/\/\/.*$/gm, "");

// ── المحرّكُ هو المرجع ────────────────────────────────────────────────
const migDir = join(root, "backend", "internal", "migrate", "migrations");
const backendRoles = new Set();
for (const f of readdirSync(migDir)) {
  const src = read(join(migDir, f));
  for (const hit of src.matchAll(/INSERT INTO roles \(code, name_key\) VALUES([\s\S]*?);/g)) {
    for (const [, code] of hit[1].matchAll(/\(\s*'([a-z_]+)'\s*,/g)) backendRoles.add(code);
  }
}
if (backendRoles.size === 0) {
  problems.push("لم يُقرأ رمزُ دورٍ واحدٌ من هجرات المحرّك — والفاحصُ بلا مرجعٍ لا يقيس شيئاً");
}

// ── ويُشغَّل المصدرُ الحقيقيّ ─────────────────────────────────────────
register(pathToFileURL(join(here, "_tsload.mjs")).href);
let meta;
try {
  meta = await import(pathToFileURL(join(web, "apps", "rahalgo", "src", "lib", "rolemeta.ts")).href);
} catch (e) {
  console.error("تعذّر تشغيلُ وحدةِ السياسة — والحارسُ لا يقيس بلا تشغيل:\n  " + e.message);
  process.exit(1);
}
const {
  assignmentGroups,
  signupGroups,
  classifyRole,
  canGrantNew,
  creatableAtSignup,
  isOwner,
  STAFF_ASSIGNABLE_CODES,
  PROTECTED_ROLE_CODES,
  ELEVATED_ROLE_CODES,
  LEGACY_ROLE_CODES,
} = meta;

const ARABIC = /[؀-ۿ]/;

/** **جوابُ المحرّك كما يأتي فعلاً** — رمزٌ ومفتاحٌ أو اسمٌ حرّ. */
const backendAnswer = [
  ...[...backendRoles].map((code) => ({ code, name_key: `roles.${code}` })),
  // **والدورُ الذي أنشأه المالكُ من اللوحة** — اسمٌ حرٌّ لا مفتاح.
  { code: "observability", name_key: "مراقبة التشغيل" },
];

const flat = (groups) => groups.flatMap((g) => g.roles.map((r) => ({ ...r, cls: g.cls })));
const codesOf = (groups) => new Set(flat(groups).map((r) => r.code));

// ══════════════════════════════════════════════════════════════════════
//  ١ · **أدوارُ العمل المعتمدةُ ظاهرةٌ في مجموعة الموظّفين**
// ══════════════════════════════════════════════════════════════════════
//
// **وهذه قائمةُ المالك نصّاً** (بندُ ٣) — **بالرموز الفعليّة للمحرّك.**
const REQUIRED_STAFF = [
  "operations",
  "finance",
  "analytics",
  "customer_support",
  "driver_verification",
  "merchant_verification",
  "marketing_content",
  "trust_safety",
  "observability",
];
{
  const groups = assignmentGroups(backendAnswer, []);
  const staff = new Set(
    groups.filter((g) => g.cls === "staff").flatMap((g) => g.roles.map((r) => r.code)),
  );
  const missing = REQUIRED_STAFF.filter((c) => !staff.has(c));
  if (missing.length > 0) {
    problems.push(
      "أدوارُ عملٍ لا تظهر في نافذة إسناد الأدوار — **والمالكُ لا يستطيع إسنادَها**:\n   " +
        missing.join(" · "),
    );
  } else {
    notes.push(`${staff.size} دورَ عملٍ أهلٌ للإسناد — ومنها «مراقبة التشغيل»`);
  }
  // **وكلٌّ منها باسمٍ عربيٍّ لا بمفتاحٍ ولا برمز.**
  for (const r of flat(groups)) {
    if (!ARABIC.test(r.label) || r.label.includes("roles.")) {
      problems.push(`دورٌ يُعرَض بغير عربيّةٍ في نافذة الإسناد: ${r.code} ⇒ ${r.label}`);
    }
  }
}

// ══════════════════════════════════════════════════════════════════════
//  ٢ · **والمالكُ الأعلى لا يُسند من المسار العاديّ**
// ══════════════════════════════════════════════════════════════════════
{
  if (PROTECTED_ROLE_CODES.length === 0 || !PROTECTED_ROLE_CODES.includes("owner_super_admin")) {
    problems.push("`owner_super_admin` غيرُ محميٍّ في السياسة — **نقرةٌ واحدةٌ تمنح كلَّ قدرة**");
  }
  const ordinary = assignmentGroups(backendAnswer, []);
  if (codesOf(ordinary).has("owner_super_admin")) {
    problems.push(
      "`owner_super_admin` معروضٌ في نافذةِ أدوارٍ عاديّة — **وزلّةُ نقرةٍ تمنح كلَّ شيء**",
    );
  }
  // ── ٣ · **ويُرى مقفلاً إن كان مملوكاً** — وإلّا مُنع النزعُ لا المنح.
  const held = assignmentGroups(backendAnswer, ["owner_super_admin"]);
  const row = flat(held).find((r) => r.code === "owner_super_admin");
  if (!row) {
    problems.push(
      "حسابٌ يملك `owner_super_admin` لا يُعرَض دورُه — **فلا يُنزَع من اللوحة إطلاقاً**",
    );
  } else if (!row.locked || row.cls !== "protected") {
    problems.push(
      `الدورُ المحميُّ المملوكُ غيرُ مقفل (locked=${row.locked} cls=${row.cls}) — فيُبدَّل بنقرة`,
    );
  } else {
    notes.push("`owner_super_admin`: محجوبٌ في المسار العاديّ · ومقفلٌ مرئيٌّ لمن يملكه");
  }
}

// ══════════════════════════════════════════════════════════════════════
//  ٤ · **ودورٌ مجهولٌ لا يكسب أهليّةً صامتة**
// ══════════════════════════════════════════════════════════════════════
{
  if (classifyRole("zz_future_role") !== "custom") {
    problems.push("دورٌ مجهولٌ يُصنَّف غيرَ `custom` — **فيكسب أهليّةً لم يقرّرها أحد**");
  }
  if (classifyRole("zz_future_role") === "staff") {
    problems.push("دورٌ مجهولٌ يُحسب دورَ عملٍ صامتاً");
  }
  const withUnknown = assignmentGroups(
    [...backendAnswer, { code: "zz_night_crew", name_key: "فريقُ الليل" }],
    [],
  );
  const row = flat(withUnknown).find((r) => r.code === "zz_night_crew");
  if (!row) {
    problems.push(
      "دورٌ أنشأه المالكُ من اللوحة لا يُعرَض — **وهذا هو العطبُ نفسُه يعود** (`observability`)",
    );
  } else if (row.cls !== "custom") {
    problems.push(`دورٌ مجهولٌ وقع في مجموعة ${row.cls} لا في «أُنشئت من اللوحة»`);
  } else if (row.label !== "فريقُ الليل") {
    problems.push(`دورٌ مجهولٌ لا يُعرَض باسمه من المحرّك: ${row.label}`);
  }
}

// ══════════════════════════════════════════════════════════════════════
//  ٥ · **والوجودُ من المحرّك وحدَه**
// ══════════════════════════════════════════════════════════════════════
{
  // **جوابٌ فيه دورانِ فقط** — **فلا تظهر بقيّةُ ما تعرفه الواجهة.**
  const tiny = assignmentGroups([{ code: "finance", name_key: "roles.finance" }], []);
  const got = [...codesOf(tiny)].sort();
  if (got.length !== 1 || got[0] !== "finance") {
    problems.push(
      "الواجهةُ تخترع أدواراً لم يقلها المحرّك: " + got.join(" · ") + " — **وذاك `ADG-1` يعود**",
    );
  }
  if (assignmentGroups([], []).length !== 0) {
    problems.push("جوابٌ فارغٌ من المحرّك يُنتج مجموعاتٍ — **قائمةٌ من الواجهة**");
  } else {
    notes.push("جوابٌ فارغٌ ⇒ لا حبّةَ واحدة — والوجودُ من المحرّك");
  }
}

// ══════════════════════════════════════════════════════════════════════
//  ٦ · **وكلُّ رمزٍ في هجرات المحرّك مصنَّفٌ صراحةً**
// ══════════════════════════════════════════════════════════════════════
//
// **ولو سقط رمزٌ مبذورٌ في `custom` لظهر تحت «أُنشئت من اللوحة»** —
// **خبرٌ كاذبٌ للموظّف**، وهجرةٌ جديدةٌ تُسقط الحارسَ بدلاً من أن تمرّ.
{
  const unclassified = [...backendRoles].filter((c) => classifyRole(c) === "custom").sort();
  if (unclassified.length > 0) {
    problems.push(
      "رموزٌ مبذورةٌ في المحرّك بلا تصنيفٍ في السياسة — **تظهر «أُنشئت من اللوحة» وهي ليست كذلك**:\n   " +
        unclassified.join(" · "),
    );
  } else {
    notes.push(`${backendRoles.size} رمزاً مبذوراً — كلُّها مصنَّفةٌ صراحةً`);
  }
  const ghosts = STAFF_ASSIGNABLE_CODES.filter(
    (c) => !backendRoles.has(c) && c !== "observability",
  );
  if (ghosts.length > 0) {
    problems.push("السياسةُ تصنّف رمزاً لا يعرفه المحرّك:\n   " + ghosts.join(" · "));
  }
}

// ══════════════════════════════════════════════════════════════════════
//  ٦ب · **والمرتفعُ للمالك وحدَه · والإرثُ لا يُمنَح** (٢٠٢٦-٠٩-١٢)
// ══════════════════════════════════════════════════════════════════════
//
// **وقِيس في المحرّك أنّ `admin` يبلغ `roles.manage`** — **فمن منحه
// منح سلطةَ السلطات.** **وسلسلةُ تصعيدٍ كاملةٌ مرّت منه**: موظّفٌ بلا
// `roles.manage` أنشأ حسابَ أدمنٍ بكلمةٍ يختارها ثمّ دخل به.
{
  const OWNER = ["owner_super_admin"];
  if (ELEVATED_ROLE_CODES.length === 0 || !ELEVATED_ROLE_CODES.includes("admin")) {
    problems.push("`admin` غيرُ مصنَّفٍ مرتفعاً — **وهو يبلغ `roles.manage`**");
  }
  if (canGrantNew("admin", [])) {
    problems.push("**`admin` يُعرَض لمنحٍ جديدٍ لغير المالك** — **وهو سلطةُ السلطات**");
  }
  if (!canGrantNew("admin", OWNER)) {
    problems.push("**المالكُ لا يرى `admin`** — فلا يستطيع تعيينَ أدمنٍ إطلاقاً");
  }
  if (creatableAtSignup("admin") || creatableAtSignup("observability")) {
    problems.push("**دورٌ غيرُ صفةِ حسابٍ يُخلَق به حسابٌ مباشرةً** — **وذاك بابُ التصعيد**");
  }
  if (!creatableAtSignup("customer") || !creatableAtSignup("driver")) {
    problems.push("**صفةُ الحساب لا تُخلَق مباشرةً** — وأُغلق بابٌ مشروع");
  }
  if (!isOwner(OWNER) || isOwner(["admin"])) {
    problems.push("`isOwner` تُخطئ من هو المالك");
  }
  // ── والإرثُ `ops` يُقرأ ولا يُمنَح (`OPS-5`) ──────────────────────
  if (!LEGACY_ROLE_CODES.includes("ops")) {
    problems.push("`ops` غيرُ مصنَّفٍ إرثاً — **واسمُه العربيُّ كاسم `operations` وقدراتُهما مختلفة**");
  }
  if (canGrantNew("ops", OWNER)) {
    problems.push("**`ops` يُعرَض لمنحٍ جديد** — ولو للمالك");
  }
  const answer = [
    { code: "admin", name_key: "roles.admin" },
    { code: "ops", name_key: "roles.ops" },
    { code: "observability", name_key: "مراقبة التشغيل" },
    { code: "owner_super_admin", name_key: "roles.owner_super_admin" },
  ];
  // **ما لا يُمنَح جديداً لا يُعرَض إلّا مملوكاً** — فيُرى ليُنزَع.
  const plain = codesOf(assignmentGroups(answer, [], ["admin"]));
  for (const code of ["admin", "ops", "owner_super_admin"]) {
    if (plain.has(code)) {
      problems.push(`**${code} معروضٌ لمنحٍ جديدٍ على مشغّلٍ غيرِ مالك**`);
    }
  }
  const heldAll = flat(assignmentGroups(answer, ["admin", "ops", "owner_super_admin"], ["admin"]));
  for (const code of ["admin", "ops", "owner_super_admin"]) {
    const row = heldAll.find((r) => r.code === code);
    if (!row) {
      problems.push(`**حسابٌ يحمل ${code} لا يُعرَض دورُه** — **فلا يُنزَع من اللوحة إطلاقاً**`);
    }
  }
  // **والمحميُّ وحدَه مقفل** — و`ops` و`admin` يُنزعان.
  for (const [code, wantLocked] of [["owner_super_admin", true], ["ops", false], ["admin", false]]) {
    const row = heldAll.find((r) => r.code === code);
    if (row && row.locked !== wantLocked) {
      problems.push(`${code}: locked=${row.locked} والمنتظَرُ ${wantLocked}`);
    }
  }
  // ── وشاشةُ الإنشاء: ثلاثُ فئاتٍ ولا محميَّ ولا إرث ───────────────
  const sg = signupGroups(answer, ["owner_super_admin"]);
  const cls = sg.map((g) => g.cls);
  if (cls.includes("protected") || cls.includes("legacy")) {
    problems.push("**شاشةُ الإنشاء تعرض المحميَّ أو الإرث** — ولا يُخلَق بهما حساب");
  }
  for (const g of sg) {
    const want = g.cls !== "account_type";
    if (!!g.viaGrant !== want) {
      problems.push(`مجموعةُ ${g.cls} في الإنشاء: viaGrant=${g.viaGrant} والمنتظَرُ ${want} — ` +
        "**والفرقُ بين «يُخلَق به» و«يُمنَح بعده» يجب أن يُعلَن**");
    }
  }
  if (!sg.some((g) => g.cls === "elevated")) {
    problems.push("**المالكُ لا يرى «الإدارة المرتفعة» في الإنشاء**");
  }
  if (signupGroups(answer, ["admin"]).some((g) => g.cls === "elevated")) {
    problems.push("**غيرُ المالك يرى «الإدارة المرتفعة» في الإنشاء** — ونقرةٌ يردُّها المحرّك");
  }
  if (problems.length === 0) {
    notes.push("المرتفعُ للمالك · والإرثُ لا يُمنَح · والمحميُّ مقفلٌ مرئيّاً لحامله");
  }
}

// ══════════════════════════════════════════════════════════════════════
//  ٧ · **ومصدرُ السياسةِ واحد** — ولا مصفوفةَ رموزٍ في شاشة
// ══════════════════════════════════════════════════════════════════════
const SCREENS = [
  ["نافذةُ أدوار الحساب", join(web, "apps", "rahalgo", "src", "components", "admin", "ManageRolesModal.tsx")],
  ["شاشةُ الحسابات", join(web, "apps", "rahalgo", "src", "components", "admin", "accounts", "all.tsx")],
];
for (const [name, path] of SCREENS) {
  const src = strip(read(path));
  if (src === "") {
    problems.push(`${name}: الملفُّ لم يُقرأ — والحارسُ بلا مرجع`);
    continue;
  }
  // **والمصدرُ واحدٌ وإن اختلف بابُه**: نافذةُ الإسناد تنادي
  // `assignmentGroups`، **وشاشةُ الإنشاء `signupGroups` وهي تنادي
  // الأولى داخلَها** — **ومن كتب سياسةً ثالثةً يسقط هنا.**
  if (!/(assignmentGroups|signupGroups)\(/.test(src)) {
    problems.push(`${name} لا تنادي سياسةَ الإسناد — **سياسةٌ ثانيةٌ تُكتب من جديد**`);
  }
  // **ومصفوفةُ رموزِ أدوارٍ مكتوبةٌ في الشاشة** — ثلاثةٌ أو أكثرُ متجاورة.
  const arr = src.match(
    /\[\s*"(?:customer|driver|merchant|sales|ops|finance|admin|operations)"(?:\s*,\s*"[a-z_]+"){2,}/,
  );
  if (arr) {
    problems.push(`${name} فيها مصفوفةُ رموزِ أدوارٍ مكتوبةً: ${arr[0].slice(0, 80)}`);
  }
  if (/Object\.keys\(\s*ROLE_LABELS/.test(src)) {
    problems.push(`${name} تقرأ الوجودَ من معجم العرض — \`ADG-1\``);
  }
  // ── ٨ · **وكلُّ `Chips` في مسار الأدوار يلتفّ** — حارسُ القصّ.
  for (const [, block] of src.matchAll(/<Chips\b([\s\S]*?)\/>/g)) {
    if (!/\bwrap\b/.test(block)) {
      problems.push(
        `${name}: \`Chips\` بلا \`wrap\` — **تنزلق أفقيّاً وشريطُها مخفيٌّ بالأنماط**، ` +
          "**فما بعد الحبّةِ الثالثةِ لا يُرى ولا يُعرَف أنّه موجود.** (عطبُ ٢٠٢٦-٠٩-١٢.)",
      );
    }
  }
}

// ══════════════════════════════════════════════════════════════════════
//  ٩ · **والرمزُ التقنيُّ ظاهرٌ ثانويّاً** — و`ops` و`operations` اسمُهما واحد
// ══════════════════════════════════════════════════════════════════════
{
  // **و`ops` لا يُعرَض إلّا مملوكاً بعد ٢٠٢٦-٠٩-١٢** — **فيُقاس
  // مملوكاً**، وإلّا صار فحصُ الاسمين المتطابقين لا يقيس شيئاً.
  const groups = assignmentGroups(backendAnswer, ["ops"]);
  const byLabel = new Map();
  for (const r of flat(groups)) {
    byLabel.set(r.label, [...(byLabel.get(r.label) ?? []), r.code]);
  }
  const dupes = [...byLabel.entries()].filter(([, c]) => c.length > 1);
  for (const [, path] of SCREENS) {
    const src = strip(read(path));
    if (!/\{r\.code\}/.test(src)) {
      problems.push(
        `${path.split(/[\\/]/).pop()}: رمزُ الدور غيرُ ظاهرٍ في الحبّة — ` +
          "**واسمانِ عربيّانِ متطابقانِ يُقاسان**: " +
          (dupes.map(([l, c]) => `${l} = ${c.join("/")}`).join(" · ") || "لا شيءَ اليوم") +
          " — **فبلا الرمزِ لا يعرف الموظّفُ أيَّهما يُسند.**",
      );
    }
  }
  if (dupes.length > 0) {
    notes.push(
      "أسماءٌ عربيّةٌ مشتركة: " + dupes.map(([l, c]) => `${l} = ${c.join("/")}`).join(" · ") +
        " — والرمزُ يفرّقها",
    );
  }
}

if (problems.length > 0) {
  console.error("سياسةُ إسناد الأدوار — خلل:");
  for (const p of problems) console.error("  ✗ " + p);
  process.exit(1);
}
for (const n of notes) console.log("  · " + n);
console.log("سياسةُ الإسناد مركزيّةٌ · الوجودُ من المحرّك · ولا حبّةَ مقصوصة.");
