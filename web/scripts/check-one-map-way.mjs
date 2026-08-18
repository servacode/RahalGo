/**
 * ══════════════════════════════════════════════════════════════════════
 * **أسلوبٌ واحدٌ لالتقاط الموقع — لا اثنان**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «مسألةُ تحديد العنوان على الخريطة تستخدم
 *  أكثرَ من أسلوبٍ وأكثرَ من طريقة وهذا غلط — المفروض تفتح الخريطةُ
 *  وهناك أيقونةُ تحديد الموقع بدقّة ثمّ حفظ وانتهى الأمر».)
 *
 * # ثلاثةُ ثوابتَ تُقاس
 *
 * **١ · لا يكتب في `LastPoint` إلّا ثلاثة**: قارئا جهاز التموضع في
 * التطبيقين، وخدمةُ تتبّع السائق، والحقلُ الموحَّد بعد أن تردّ الخريطة.
 * **ورابعٌ يكتب فيه يعني شاشةً تثبّت نقطةً بلا خريطة** — وهو الأسلوبُ
 * الثاني الذي حُذف.
 *
 * **٢ · ولا يعود نصُّ الزرّ المحذوف** (`acc_addr_pin`) — كان «حدّد
 * موقعي على الخريطة» بلا خريطةٍ تُفتح.
 *
 * **٣ · وأيقونةُ «موقعي» في المنتقي شرطُ الأسلوب الواحد** — بلاها يعود
 * من أراد موضعَه إلى تحريك الخريطة بإصبعه، **فيُطلب زرٌّ ثانٍ من جديد.**
 *
 * # ولماذا حارسٌ لا مراجعة
 *
 * **الأسلوبُ الثاني لا يُضاف عمداً** — يُضاف لأنّ شاشةً جديدةً تحتاج
 * نقطةً واستُسهل كتابتُها مباشرةً. **وثلاثُ شاشاتٍ بثلاث طرقٍ تجعل
 * التطبيقَ يبدو ثلاثةَ تطبيقات.**
 */
import { readFileSync, readdirSync, statSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const here = dirname(fileURLToPath(import.meta.url));
const mobile = join(here, "..", "..", "mobile");

/** **من يحقّ له أن يكتب موضعاً** — كلٌّ بسببه. */
const WRITERS = new Set([
  "Here.kt",          // **قارئُ جهاز التموضع** — هو مصدرُ الموضع.
  "LocationService.kt", // **تتبّعُ السائق** — ورديّةٌ مفتوحةٌ تُبَثّ.
  "PointField.kt",    // **الحقلُ الموحَّد** — يكتب ما ردّته الخريطة.
  "LastPoint.kt",     // **الحاملُ نفسُه.**
]);

const offenders = [];
function walk(dir) {
  for (const name of readdirSync(dir)) {
    const full = join(dir, name);
    if (statSync(full).isDirectory()) {
      if (name !== "build") walk(full);
      continue;
    }
    if (!name.endsWith(".kt") || WRITERS.has(name)) continue;
    if (/LastPoint\.set\s*\(/.test(readFileSync(full, "utf8"))) {
      offenders.push(full.slice(mobile.length + 1).split("\\").join("/"));
    }
  }
}
walk(mobile);

const problems = offenders.map(
  (f) => `«${f}» يكتب موضعاً بيده — **شاشةٌ تثبّت نقطةً بلا أن تفتح الخريطة**`,
);

const strings = readFileSync(
  join(mobile, "ui/src/main/res/values/strings.xml"),
  "utf8",
);
if (/name="acc_addr_pin"/.test(strings)) {
  problems.push("عاد «acc_addr_pin» — زرُّ الأسلوب الثاني الذي حُذف");
}

const pick = readFileSync(
  join(mobile, "map/src/main/kotlin/com/rahalgo/map/PickPoint.kt"),
  "utf8",
);
if (!pick.includes("ic_my_location")) {
  problems.push("المنتقي بلا أيقونة «موقعي» — والأسلوبُ الواحدُ يقوم عليها");
}

if (problems.length > 0) {
  console.error("التقاطُ الموقع بأكثرَ من أسلوب:");
  for (const p of problems) console.error("  · " + p);
  process.exit(1);
}
console.log(
  "التقاطُ الموقع بأسلوبٍ واحد — خريطةٌ فيها أيقونةُ موقعي ثمّ حفظ.",
);
