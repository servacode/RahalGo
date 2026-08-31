/**
 * ══════════════════════════════════════════════════════════════════════
 * **حارسُ الظلّ — اسمُ نصٍّ في معجمين**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ومعجمُ التطبيق يغلب معجمَ `:ui` صامتاً.** أندرويد يدمج موارد
 * المكتبة مع موارد التطبيق، **وحين يتكرّر الاسمُ يفوز التطبيق** — بلا
 * تحذيرٍ في البناء ولا سطرٍ في سجلّ.
 *
 * # وقعت ٢٠٢٦-٠٨-٣١ — **بعد النشر لا قبله**
 *
 * **نُقلت شاشةُ الشكاوى إلى `:ui`** بلفظٍ واحدٍ للثلاثة: «شكاوى عليّ» و
 * «شكاوى رفعتها». **وبُنيت الأربعةُ وثُبّتت** — **ثمّ ظهر السائقُ يقول
 * «ما رفع عليك»**: معجمُه كان يحمل `tik_on_me` باسمه.
 *
 * **ولم يسقط بناء، ولم يظهر خطأ** — الشاشةُ مركزيّةٌ والنصُّ ليس. **ولا
 * يُكشف إلّا بعينٍ تقرأ الجهاز.**
 *
 * # ورأسُ معجم `:ui` كان يحذّر منه حرفاً
 *
 * «**والنصُّ ينتقل مع قطعته** — ولا يبقى في المعجمين: اسمٌ في معجمين
 * يُغلب أحدُهما صامتاً، **فيُصحَّح في واحدٍ ويبقى القديمُ معروضاً**».
 *
 * **وتحذيرٌ في تعليقٍ لا يمنع** — وهذا يمنع.
 */

import { readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";

const MOBILE = join(process.cwd(), "..", "mobile");
const UI = join(MOBILE, "ui", "src", "main", "res", "values", "strings.xml");

/** **الأسماءُ في معجمٍ واحد** — بلا قيمها: التكرارُ هو الخبر. */
function keysOf(file) {
  const out = new Set();
  let xml;
  try {
    xml = readFileSync(file, "utf8");
  } catch {
    return out;
  }
  for (const m of xml.matchAll(/<string name="([^"]+)"/g)) out.add(m[1]);
  return out;
}

const central = keysOf(UI);
if (central.size === 0) {
  console.error("لم يُقرأ معجمُ :ui — الحارسُ لا يعمل على فراغ.");
  process.exit(1);
}

const apps = readdirSync(MOBILE).filter((d) => d.startsWith("app-"));
const clashes = [];
for (const app of apps) {
  const own = keysOf(join(MOBILE, app, "src", "main", "res", "values", "strings.xml"));
  for (const k of own) if (central.has(k)) clashes.push(`${app}  ${k}`);
}

if (clashes.length > 0) {
  console.error("\nاسمُ نصٍّ في معجمين — ومعجمُ التطبيق يغلب :ui صامتاً:\n");
  for (const c of clashes) console.error("  " + c);
  console.error(
    "\nويُحذف من معجم التطبيق — **النصُّ ينتقل مع قطعته**، " +
      "ومن أبقاه صحّح في واحدٍ وبقي القديمُ معروضاً.\n",
  );
  process.exit(1);
}

console.log(
  `لا اسمَ في معجمين — ${central.size} نصّاً مركزيّاً و${apps.length} تطبيقاتٍ، ولا ظلّ.`,
);
