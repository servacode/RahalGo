#!/usr/bin/env node
/**
 * ══════════════════════════════════════════════════════════════════════
 *  **اسمُ الدور العربيُّ — حارسٌ دائمٌ يُشغّل الشيفرةَ لا يقرؤها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ما وقع
 *
 * **هجرةُ `0138` أضافت ثمانيةَ أدوارٍ** (`analytics` و`customer_support`
 * و`driver_verification` و`marketing_content` وغيرَها) **ولم يُضف لها
 * اسمٌ عربيّ** — **فظهرت رموزُها الإنجليزيّةُ لموظّفٍ عربيٍّ يقرأ.**
 *
 * **والحارسُ النصّيُّ لا يمسك هذا**: المعجمُ سليمُ الشكل، **وناقصٌ
 * بالمقارنة بالمحرّك.** **فيُقرأ المحرّكُ نفسُه**: رموزُ الأدوار من
 * هجراته، ومعجمُ القدرات من `authz/catalog.go`.
 *
 * # وما يُحرَس
 *
 *	١ · كلُّ رمزِ دورٍ في المحرّك له اسمٌ عربيٌّ ووصفٌ عربيّ
 *	٢ · والمحرّكُ يسبق: اسمُه الحرُّ يُعرَض كما كتبه صاحبُه
 *	٣ · ومفتاحُ الترجمة لا يُعرَض لإنسان أبداً
 *	٤ · ودورٌ لا تعرفه الواجهةُ يُعرَض رمزُه بأمانٍ ولا يسقط
 *	٥ · وكلُّ قدرةٍ في المعجم تُعرَض بعربيّة
 *	٦ · وقدرةٌ مجهولةٌ تُعرَض رمزَها بأمان
 *	٧ · والمصدرُ واحدٌ — لا معجمَ أسماءٍ في شاشة
 *	٨ · والوجودُ من المحرّك — لا `Object.keys` على معجم عرض
 *
 * # ولماذا يُشغَّل لا يُقرَأ
 *
 * **وفحصٌ نصّيٌّ يسأل «هل الاسمُ مكتوبٌ في الملفّ؟»** — **ولا يسأل
 * «هل الدالّةُ تُرجعه؟»** فتُستورَد الوحدةُ ويُنادى ما فيها.
 * **و`node` ينزع الأنواعَ بنفسه، وخُطّافٌ صغيرٌ يحلّ JSON وامتدادَ
 * `.ts`** — مِسندُ اختبارٍ لا تبديلَ في المصدر.
 */
import { readFileSync, readdirSync, statSync } from "node:fs";
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

// ── المحرّكُ هو المرجع: رموزُ الأدوار من هجراته ───────────────────────
const migDir = join(root, "backend", "internal", "migrate", "migrations");
const backendRoles = new Set();
for (const f of readdirSync(migDir)) {
  const src = read(join(migDir, f));
  const m = /INSERT INTO roles \(code, name_key\) VALUES([\s\S]*?);/g;
  let hit;
  while ((hit = m.exec(src)) !== null) {
    for (const [, code] of hit[1].matchAll(/\(\s*'([a-z_]+)'\s*,/g)) backendRoles.add(code);
  }
}
if (backendRoles.size === 0) {
  problems.push("لم يُقرأ رمزُ دورٍ واحدٌ من هجرات المحرّك — والفاحصُ بلا مرجعٍ لا يقيس شيئاً");
}

// ── ومعجمُ القدرات من الشيفرة ────────────────────────────────────────
const catalog = read(join(root, "backend", "internal", "authz", "catalog.go"));
const backendCaps = new Map();
for (const [, code, desc] of catalog.matchAll(/^\s*([A-Za-z]+):\s*"([^"]+)",\s*$/gm)) {
  backendCaps.set(code, desc);
}
const capCodes = [...catalog.matchAll(/Capability\s*=\s*"([a-z_.]+)"/g)].map((x) => x[1]);

// ── ويُشغَّل المصدرُ الحقيقيّ ─────────────────────────────────────────
register(pathToFileURL(join(here, "_tsload.mjs")).href);
let meta;
try {
  meta = await import(pathToFileURL(join(web, "apps", "rahalgo", "src", "lib", "rolemeta.ts")).href);
} catch (e) {
  console.error("تعذّر تشغيلُ وحدةِ الأسماء — والحارسُ لا يقيس بلا تشغيل:\n  " + e.message);
  process.exit(1);
}
const { roleLabelByCode, roleLabel, roleDescription, capabilityLabel, ASSIGNABLE_ROLE_CODES } = meta;

const ARABIC = /[؀-ۿ]/;

// ── ١ · كلُّ رمزِ دورٍ في المحرّك له اسمٌ ووصفٌ عربيّان ───────────────
const noLabel = [];
const noDesc = [];
for (const code of [...backendRoles].sort()) {
  const label = roleLabelByCode(code);
  if (label === code || !ARABIC.test(label)) noLabel.push(code);
  const desc = roleDescription(code);
  if (!desc || !ARABIC.test(desc)) noDesc.push(code);
}
if (noLabel.length > 0) {
  problems.push(
    `أدوارٌ في المحرّك بلا اسمٍ عربيّ — يقرؤها موظّفٌ برمزها الإنجليزيّ:\n   ` + noLabel.join(" · "),
  );
}
if (noDesc.length > 0) {
  problems.push(`أدوارٌ بلا وصفٍ عربيّ:\n   ` + noDesc.join(" · "));
}
if (noLabel.length === 0 && noDesc.length === 0) {
  notes.push(`${backendRoles.size} رمزَ دورٍ في المحرّك — كلُّها باسمٍ عربيٍّ ووصف`);
}

// ── ٢ · والمحرّكُ يسبق: اسمُه الحرُّ يُعرَض كما كُتب ──────────────────
if (roleLabel({ code: "observability", name_key: "مراقبة التشغيل الليلي" }) !== "مراقبة التشغيل الليلي") {
  problems.push("اسمُ المحرّك الحرُّ لا يُعرَض — ودورٌ سمّاه المالكُ يظهر باسمٍ آخر");
}
if (roleLabel({ code: "zz_owner_named", name_key: "فريقُ الليل" }) !== "فريقُ الليل") {
  problems.push("دورٌ مخصَّصٌ باسمٍ من المحرّك لا يُعرَض باسمه");
}

// ── ٣ · ومفتاحُ الترجمة لا يُعرَض أبداً ──────────────────────────────
for (const code of ["admin", "finance", "analytics", "trust_safety"]) {
  const out = roleLabel({ code, name_key: `roles.${code}` });
  if (out.includes("roles.") || !ARABIC.test(out)) {
    problems.push(`مفتاحُ ترجمةٍ يُعرَض لإنسان: ${code} ⇒ ${out}`);
  }
}

// ── ٤ · ودورٌ مجهولٌ يُعرَض بأمانٍ ولا يسقط ──────────────────────────
try {
  const unknown = roleLabel({ code: "zz_future_role", name_key: "" });
  if (unknown !== "zz_future_role") {
    problems.push(`دورٌ مجهولٌ لا يُعرَض رمزُه: ${unknown}`);
  }
  if (roleLabelByCode("zz_future_role") !== "zz_future_role") {
    problems.push("دورٌ مجهولٌ بلا اسمِ محرّكٍ لا يُعرَض رمزُه");
  }
  if (roleDescription("zz_future_role") !== "") {
    problems.push("وصفٌ مخترَعٌ لدورٍ مجهول");
  }
} catch (e) {
  problems.push("دورٌ مجهولٌ يُسقط العرضَ: " + e.message);
}

// ── ٥ · وكلُّ قدرةٍ في المعجم تُعرَض بعربيّة ─────────────────────────
//
// **والوصفُ يوصَل بمفتاحه لا بترتيبه** — `ObservabilityRead` في جدول
// الأوصاف هو `observability.read` في الثوابت.
const descByCode = new Map();
{
  const nameOf = new Map();
  for (const [, ident, code] of catalog.matchAll(/([A-Za-z]+)\s+Capability\s*=\s*"([a-z_.]+)"/g)) {
    nameOf.set(ident, code);
  }
  for (const [ident, d] of backendCaps) {
    const code = nameOf.get(ident);
    if (code) descByCode.set(code, d);
  }
}
const capNoArabic = capCodes.filter((code) => !ARABIC.test(capabilityLabel(code, descByCode.get(code))));
if (capCodes.length === 0) {
  problems.push("لم تُقرأ قدرةٌ واحدةٌ من معجم الشيفرة — والفاحصُ بلا مرجع");
} else if (capNoArabic.length > 0) {
  problems.push("قدراتٌ تُعرَض برمزها الإنجليزيّ:\n" + capNoArabic.join(" · "));
} else {
  notes.push(`${capCodes.length} قدرةً في المعجم — كلُّها تُعرَض بعربيّة`);
}
if (capabilityLabel("observability.read", "") !== "عرض مراقبة النظام") {
  problems.push("اسمُ العرض المطلوب لـ observability.read غيرُ مطبَّق");
}

// ── ٦ · وقدرةٌ مجهولةٌ تُعرَض رمزَها بأمان ───────────────────────────
if (capabilityLabel("zz.unknown_capability") !== "zz.unknown_capability") {
  problems.push("قدرةٌ مجهولةٌ لا تُعرَض رمزَها");
}
if (capabilityLabel("zz.unknown_capability", "وصفٌ من محرّكٍ أحدثَ") !== "وصفٌ من محرّكٍ أحدثَ") {
  problems.push("وصفُ المحرّكِ لقدرةٍ لا تعرفها الواجهةُ يُهمَل");
}

// ── ٧ · والمصدرُ واحد — لا معجمَ أسماءٍ في شاشة ──────────────────────
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
    else if (/\.(ts|tsx)$/.test(name)) out.push(full);
  }
  return out;
}
const files = sources(join(web, "apps", "rahalgo", "src")).concat(
  sources(join(web, "packages", "ui", "src")),
);
const owner = join("lib", "rolemeta.ts");
const strays = [];
for (const f of files) {
  if (f.endsWith(owner)) continue;
  const body = read(f)
    .replace(/\/\*[\s\S]*?\*\//g, "")
    .replace(/\/\/.*$/gm, "");
  if (/terms\.(roleNames|roleDescriptions|capabilityNames)/.test(body)) {
    strays.push(f.replace(web + "\\", "").replace(web + "/", ""));
  }
}
if (strays.length > 0) {
  problems.push("معجمُ أسماءٍ يُقرأ خارجَ مصدره الواحد:\n   " + strays.join("\n   "));
} else {
  notes.push(`مصدرٌ واحدٌ للأسماء — ${files.length} ملفّاً فُحص`);
}

// ── ٨ · والوجودُ من المحرّك — لا `Object.keys` على معجم عرض ──────────
const keysOnLabels = [];
for (const f of files) {
  const body = read(f)
    .replace(/\/\*[\s\S]*?\*\//g, "")
    .replace(/\/\/.*$/gm, "");
  if (/Object\.keys\(\s*(ROLE_LABELS|ROLE_NAMES|[A-Za-z_]*roleNames)/.test(body)) {
    keysOnLabels.push(f.replace(web + "\\", "").replace(web + "/", ""));
  }
}
if (keysOnLabels.length > 0) {
  problems.push(
    "وجودُ الأدوار يُشتقّ من معجم عرضٍ — والمحرّكُ هو المرجع:\n   " + keysOnLabels.join("\n   "),
  );
}
// **وشاشةُ الأدوارِ ونافذةُ الإسناد تقرآن المحرّكَ** — وإلّا عاد العطب.
for (const [p, why] of [
  [join(web, "apps", "rahalgo", "src", "app", "dashboard", "roles", "page.tsx"), "شاشةُ الأدوار"],
  [join(web, "apps", "rahalgo", "src", "components", "admin", "ManageRolesModal.tsx"), "نافذةُ إسناد الأدوار"],
]) {
  const body = read(p)
    .replace(/\/\*[\s\S]*?\*\//g, "")
    .replace(/\/\/.*$/gm, "");
  if (!/listRoles\s*\(/.test(body)) {
    problems.push(`${why} لا تقرأ الأدوارَ من المحرّك — والواجهةُ لا تقول ما يوجد`);
  }
}
if (!Array.isArray(ASSIGNABLE_ROLE_CODES) || ASSIGNABLE_ROLE_CODES.length === 0) {
  problems.push("قائمةُ الاختيار في شاشة الحسابات غيرُ مصرَّحٍ بها");
} else {
  const outside = ASSIGNABLE_ROLE_CODES.filter((c) => !backendRoles.has(c));
  if (outside.length > 0) {
    problems.push("قائمةُ الاختيار فيها رمزٌ لا يعرفه المحرّك:\n   " + outside.join(" · "));
  }
}

// ── ٩ · والرمزُ التقنيُّ يبقى ظاهراً ثانويّاً ────────────────────────
//
// **ومن يمنح قدرةً يحتاج أن يرى ما يمنحه بالضبط** — **فالاسمُ العربيُّ
// للقراءة والرمزُ للدقّة، ولا يُحجَب.** (طلبُ المالك.)
{
  const page = read(join(web, "apps", "rahalgo", "src", "app", "dashboard", "roles", "page.tsx"))
    .replace(/\/\*[\s\S]*?\*\//g, "")
    .replace(/\/\/.*$/gm, "");
  // **والنمطُ يطابق نصّاً معروضاً لا خاصّيّةً** — **و`{r.code}` وحدَه
  // يطابق `key={r.code}` كذلك، فيمرّ فحصٌ لا يقيس شيئاً.** (كشفه شاهدُه
  // السالب.)
  if (!/>\s*\{r\.code\}\s*</.test(page)) {
    problems.push("شاشةُ الأدوار لا تُظهر رمزَ الدور نصّاً — ومن يمنح يحتاج الرمزَ بالضبط");
  }
  if (!/<code[\s\S]{0,120}\{c\}/.test(page)) {
    problems.push("شاشةُ الأدوار لا تُظهر رمزَ القدرة");
  }
  if (!/capLabel\(|capabilityLabel\(/.test(page)) {
    problems.push("شاشةُ الأدوار لا تُسمّي القدراتَ بعربيّة");
  }
  if (!/roleDescription\(/.test(page)) {
    problems.push("شاشةُ الأدوار بلا وصفٍ عربيٍّ للدور");
  }
}

if (problems.length > 0) {
  console.error("أسماءُ الأدوار — خلل:");
  for (const p of problems) console.error("  ✗ " + p);
  process.exit(1);
}
for (const n of notes) console.log("  · " + n);
console.log("أسماءُ الأدوار والصلاحيّات عربيّةٌ من مصدرٍ واحد — والوجودُ من المحرّك.");
