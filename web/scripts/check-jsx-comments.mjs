/**
 * **لا تعليقَ يُطبع للمستخدم.**
 *
 * (شهده المالك ٢٠٢٦-٠٨-٠٩ في لوحة المندوب: سطرٌ من الشيفرة معروضٌ في بطاقة
 *  «وثّق حسابك أولاً».)
 *
 * # المرض
 *
 * **`/* … *​/` داخلَ JSX ليس تعليقاً — إنّما نصٌّ يُعرض.** والتعليقُ هناك
 * يحتاج قوسين: `{/* … *​/}`.
 *
 * **ولا يُمسك في بناءٍ ولا في `tsc`**: النصُّ صالحٌ تماماً — **هو محتوى.**
 * ولا يُمسك في مراجعةٍ بالعين لأنّه يشبه التعليق حرفاً بحرف.
 *
 * **ويُقرأ للمستخدم عطباً في المنصة**: يرى قوساً ونجمةً وكلاماً عن «قرار
 * المالك» في شاشةِ توثيقِ حسابه.
 *
 * # وكيف يُكشف
 *
 * **بموضعه لا بشكله**: سطرٌ يبدأ بـ`/*` **وقبلَه سطرٌ ينتهي بـ`>` أو `}`**
 * — أي أنّنا بين عناصرِ JSX، حيث كلُّ ما يُكتب يُعرض.
 *
 * **وتعليقُ الوثائق `/**` يُستثنى** — موضعُه فوق الدوالّ لا بين العناصر.
 */
import { readdirSync, readFileSync, statSync } from "node:fs";
import { join } from "node:path";

const ROOTS = ["apps", "packages"];
const SKIP = new Set(["node_modules", ".next", "dist", ".turbo"]);

/** كلُّ ملفّات `.tsx` تحت جذرٍ ما. */
function walk(dir, out = []) {
  for (const name of readdirSync(dir)) {
    if (SKIP.has(name)) continue;
    const p = join(dir, name);
    if (statSync(p).isDirectory()) walk(p, out);
    else if (p.endsWith(".tsx")) out.push(p);
  }
  return out;
}

const bad = [];
let files = 0;

for (const root of ROOTS) {
  for (const file of walk(root)) {
    files++;
    const lines = readFileSync(file, "utf8").split("\n");
    // **والسطورُ غيرُ الفارغة وحدَها** — الفراغُ لا يقطع سياقاً.
    const idx = lines.map((l, i) => [l.trim(), i]).filter(([t]) => t !== "");

    idx.forEach(([t, i], k) => {
      if (!t.startsWith("/*") || t.startsWith("/**")) return;
      const before = k > 0 ? idx[k - 1][0] : "";
      // **وموضعُ الأبناء له توقيعٌ لا يُخطئ**: وسمٌ أُغلق قبله (`>`)، ووسمٌ
      // يُفتح بعده (`<`).
      //
      // **و`}` وحدَها لا تكفي**: تُغلق واجهةً أو دالّةً كما تُغلق تعبيراً في
      // JSX — **وحارسٌ يُنذر على تعليقٍ سليمٍ فوق دالّةٍ يُطفَأ في أسبوع.**
      let after = "";
      for (let j = k + 1; j < idx.length; j++) {
        const s = idx[j][0];
        // **ويُتخطّى جسمُ التعليق نفسُه** — أسطرُه الوسطى وآخرُه.
        if (s.startsWith("*") || s.endsWith("*/")) continue;
        after = s;
        break;
      }
      if (before.endsWith(">") && after.startsWith("<")) {
        bad.push(`${file.replace(/\\/g, "/")}:${i + 1}  ${t.slice(0, 60)}`);
      }
    });
  }
}

if (bad.length > 0) {
  console.error(
    `\n✗ ${bad.length} تعليقاً يُطبع للمستخدم — **الأقواسُ ناقصة**: {/* … */}\n`,
  );
  for (const b of bad) console.error("   " + b);
  console.error("");
  process.exit(1);
}
console.log(`لا تعليقَ يُطبع للمستخدم — فُحص ${files} ملفّاً.`);
