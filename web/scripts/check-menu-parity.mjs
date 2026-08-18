/**
 * ══════════════════════════════════════════════════════════════════════
 * **محرّرُ الأصناف: الويبُ والتطبيقُ يقولان الكلامَ نفسَه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «لا يجوز أن يشعر الشخصُ بالفرق بين الويب
 *  والتطبيق أصلاً — أيَّهما يفتح يكون العملُ موحّدا».)
 *
 * # لماذا حارسٌ لا مراجعة
 *
 * **الفرقُ لا يقع دفعةً — يقع كلمةً كلمة.** الويبُ كان يقول «متوفر»
 * والتطبيقُ «متاح»، والويبُ «الوصف» والتطبيقُ «وصف مختصر». **ولا أحدَ
 * يقرأ المعجمين جنباً إلى جنبٍ في مراجعةٍ عادية**، فيمرّ.
 *
 * **ومن عدّل كلمةً في الويب اليومَ لن يفتح `strings.xml` غدا** — إلّا
 * أن يُسقطه حارس.
 *
 * # وما يُقارَن
 *
 * **قيمُ `shared.menuEditor` في الويب مقابل نظائرها `mn_*` في تطبيق
 * المندوب** — بالخريطة أدناه. **ونصٌّ يختلف حرفاً يُسقط الفحص.**
 *
 * **وما ليس له نظيرٌ لا يُقارَن**: «خارجَ الدوام» و«حفظ» و«إلغاء»
 * أزرارُ شاشةٍ لا يحتاجها الويب.
 */
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const here = dirname(fileURLToPath(import.meta.url));
const repo = join(here, "..", "..");

const dict = JSON.parse(
  readFileSync(join(here, "..", "packages/i18n/src/locales/ar.json"), "utf8"),
);
const web = dict.shared.menuEditor;
const terms = dict.terms;
const rep = dict.rep;

const xml = readFileSync(
  join(repo, "mobile/app-rep/src/main/res/values/strings.xml"),
  "utf8",
);

/** **نصوصُ كلّ ما يُبنى** — الوحدةُ المشتركةُ والتطبيقاتُ الثلاثة. */
const APP_STRINGS = [
  "ui",
  "app-rep",
  "app-customer",
  "app-driver",
].map((m) => [
  m,
  readFileSync(join(repo, `mobile/${m}/src/main/res/values/strings.xml`), "utf8"),
]);

/** **يقرأ قيمةَ مفتاحٍ من ملفّ النصوص** — بلا مكتبةِ XML. */
function androidString(name) {
  const re = new RegExp(`<string name="${name}">([\\s\\S]*?)</string>`);
  const m = xml.match(re);
  return m ? m[1] : null;
}

/** **نظيرُ كلّ مفتاح** — يسارُه الويب ويمينُه التطبيق. */
const PAIRS = [
  [web.empty, "mn_empty"],
  [web.addItem, "mn_add_item"],
  [web.editItem, "mn_edit_item"],
  [web.noItems, "mn_no_items"],
  [web.itemName, "mn_item_name"],
  [web.itemDescription, "mn_desc"],
  [web.itemImage, "mn_item_image"],
  [web.price, "mn_price"],
  [web.salePrice, "mn_sale_price"],
  [web.available, "mn_available"],
  [web.unavailable, "mn_unavailable"],
  [web.markAvailable, "mn_mark_available"],
  [web.markUnavailable, "mn_mark_unavailable"],
  [web.confirmDeleteItem, "mn_confirm_delete"],
  [web.platformSection, "mn_platform_section"],
  [web.noPlatformSection, "mn_no_platform_section"],
  [web.noPlatformSectionHint, "mn_no_platform_section_hint"],
  [web.pendingReview, "mn_pending_review"],
  [web.rejected, "mn_rejected"],
  [web.modifiers, "mn_modifiers"],
  [web.addGroup, "mn_add_group"],
  [web.groupName, "mn_group_name"],
  [web.minSelect, "mn_min_select"],
  [web.maxSelect, "mn_max_select"],
  [web.required, "mn_required"],
  [web.optional, "mn_optional"],
  [web.optionName, "mn_option_name"],
  [web.priceDelta, "mn_price_delta"],
  [web.addOption, "mn_add_option"],
  // **وعنوانُ الشاشة والبابُ إليها** — الويبُ يسمّيهما `terms.menu`.
  [terms.menu, "mn_title"],
  [terms.menu, "cd_menu"],
  [rep.menuHint, "mn_hint"],
];

const problems = [];
for (const [want, key] of PAIRS) {
  const got = androidString(key);
  if (got === null) {
    problems.push(`«${key}» غيرُ موجودٍ في نصوص التطبيق — والويبُ يقول «${want}»`);
  } else if (got !== want) {
    problems.push(`«${key}»: التطبيقُ «${got}» والويبُ «${want}»`);
  }
}

// ══════════════════════════════════════════════════════════════════════
// **ولا كلمةَ «قسم جديد» في أيّهما**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٨: «الأقسامُ الإدارةُ هي التي تضعها، والمتجرُ
//  أو المندوبُ يختار منتجَه بأيّ قسمٍ سينزل — ما يصير كلُّ متجرٍ يعمل
//  قسماً خاصّاً فيه».)
//
// **ونصٌّ يأمر بما لا زرَّ له أسوأُ من زرٍّ ناقص**: يقف صاحبُه يبحث عمّا
// لا وجودَ له.
//
// **ولا يُبحث عن «قسم جديد» في المعجم كلِّه**: الإدارةُ تُنشئ أقسامَ
// المنصة فعلاً وتحتاج لفظَها. **إنّما يُمنع الأمرُ بإنشاء قسمٍ في
// محرّر الأصناف** — وهو ما لا زرَّ له.
const raw = readFileSync(
  join(here, "..", "packages/i18n/src/locales/ar.json"),
  "utf8",
);
if (/أضف قسم/.test(raw)) {
  problems.push("معجمُ الويب يأمر بإضافة قسم — ولا زرَّ له في محرّر الأصناف");
}
if (/أضف قسم|اسم القسم/.test(xml)) {
  problems.push("نصوصُ التطبيق تذكر إنشاءَ قسم — والأقسامُ تزرعها الإدارةُ وحدَها");
}

// ══════════════════════════════════════════════════════════════════════
// **ولا معجمَ ثانياً لمحرّر الأصناف**
// ══════════════════════════════════════════════════════════════════════
//
// **كان `admin.menu` نسخةً ثانيةً كاملةً** — لا يقرؤها أحد، **وتُبنى في
// الحزمة وتُقرأ في المراجعة على أنّها ما تعرضه الشاشة.** ومنها جاء
// «السعر» و«أضف قسماً» بعد أن صُلّح المشتركُ.
//
// **ونسختان من معجمٍ واحدٍ تفترقان حتماً** — والأولى تُصلَح وتُنسى
// الثانية.
// ══════════════════════════════════════════════════════════════════════
// **وأسماءُ حقول المُعدِّلات تُقارَن أيضاً — لا الكلماتُ وحدَها**
// ══════════════════════════════════════════════════════════════════════
//
// **الكلماتُ يراها المستعمِلُ فيشتكي، وأسماءُ الحقول لا يراها أحد** —
// **وحقلٌ باسمٍ مختلفٍ يُرسَل فيُتجاهَل بصمت**: يحفظ المندوبُ مجموعةً
// فتُقبل ٢٠٠ ولا تُخزَّن، **فلا خطأَ ولا أثر.**
//
// **والويبُ هو العقدُ**: `ModifierGroup` و`ModifierOption` في
// `MenuManager.tsx`. **وكوتلن يعلن اسمَه بـ`SerialName`.**
const tsx = readFileSync(join(here, "..", "packages/ui/src/MenuManager.tsx"), "utf8");
const kt = readFileSync(
  join(repo, "mobile/shared/src/main/kotlin/com/rahalgo/shared/rep/RepApi.kt"),
  "utf8",
);
for (const field of ["price_delta", "min_select", "max_select"]) {
  if (!tsx.includes(field)) {
    problems.push(`«${field}» لم يعد في عقد الويب — راجع الخريطة`);
  } else if (!kt.includes(`SerialName("${field}")`)) {
    problems.push(
      `«${field}» في عقد الويب ولا يعلنه كوتلن بـSerialName — ` +
        "**يُرسَل باسمٍ آخرَ فيُتجاهَل بصمت.**",
    );
  }
}
if (!/val modifiers: List<ModifierGroup>/.test(kt)) {
  problems.push("كوتلن لا يحمل المُعدِّلات — والويبُ يرسلها في كلّ حفظ");
}

// ══════════════════════════════════════════════════════════════════════
// **ولفظٌ واحدٌ للتوفّر في المنصّة كلِّها**
// ══════════════════════════════════════════════════════════════════════
//
// (طلبُ المالك ٢٠٢٦-٠٨-١٨: «متوفر غير متوفر أفضل من إيقاف مؤقت وإعادة
//  توفير، وأيضاً بالإدارة متاح وغير متاح نخلّيه متوفر وغير متوفر —
//  برأيي التسمياتُ تكون موحّدة».)
//
// **وكانت ثلاثَ مفردات**: «نافد» في محرّر الأصناف، و«غير متاح» في أقسام
// الإدارة، و«إيقاف مؤقت / إعادة توفير» على أزرار القلب.
//
// **وثلاثُ كلماتٍ لحالٍ واحدةٍ تجعل من تعلّم شاشةً يتعلّم الأخرى من
// جديد** — ويسأل: أهي حالٌ ثالثةٌ أم الاسمُ نفسُه؟
const BANNED = ["نافد", "إيقاف مؤقت", "إعادة توفير", "غير متاح"];
for (const word of BANNED) {
  if (raw.includes(word)) {
    problems.push(`عاد لفظُ «${word}» — والتوفّرُ يُقال «متوفر / غير متوفر»`);
  }
  // **ونصوصُ التطبيقات الثلاثة لا نصوصُ المندوب وحدَه** — **ولفظٌ
  // يُحرَس في ملفٍّ ويُترك في ثلاثةٍ ليس محروسا.**
  for (const [app, text] of APP_STRINGS) {
    if (text.includes(word)) {
      problems.push(`«${app}» يقول «${word}» — والتوفّرُ لفظٌ واحد`);
    }
  }
}

if (dict.admin && dict.admin.menu) {
  problems.push(
    "عاد «admin.menu» — **نسخةٌ ثانيةٌ من معجم محرّر الأصناف**، " +
      "والمحرّرُ يقرأ «shared.menuEditor» وحدَه.",
  );
}

if (problems.length > 0) {
  console.error("محرّرُ الأصناف يختلف بين الويب والتطبيق:");
  for (const p of problems) console.error("  · " + p);
  process.exit(1);
}
console.log(
  `محرّرُ الأصناف موحَّد — ${PAIRS.length} كلمةً تُطابق بين الويب والتطبيق.`,
);
