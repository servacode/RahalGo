/**
 * **لا مفتاحَ آلةٍ يُطبع نصّاً لإنسان.**
 *
 * (شهده المالك ٢٠٢٦-٠٨-٠٧: «نصوصٌ إنكليزيّة بكلّ اللوحات» — فأُصلحت بوّابةُ
 *  المتجر. **وبقيت الإدارةُ تطبعها**، وكشفه فحصٌ يدويٌّ ٢٠٢٦-٠٨-٠٨: شاشةُ
 *  العملاء المحتملين تقول «التصنيف food مطاعم».)
 *
 * # المرض
 *
 * `category_icon` مفتاحٌ لا نصّ: قيمتُه `food` و`grocery` و`pharmacy`.
 * **والمكوّنُ المركزيُّ `CategoryIcon` يحوّله إلى رسم** — ومن طبعه في
 * `<span>` طبع كلمةً إنكليزيّةً وسطَ عربيّة.
 *
 * # ولماذا حارسٌ لا انتباه
 *
 * **أُصلح مرّةً وعاد**: الحقلُ اسمُه «أيقونة» فيُظنُّ رسماً، **ولا شيءَ في
 * الشيفرة يقول إنّه مفتاح.** وثلاثُ شاشاتٍ تعرضه، أُصلحت واحدةٌ وبقيت اثنتان
 * تسع عشرة يوماً.
 *
 * # وما يُفحص
 *
 * طباعةُ حقلٍ يُعرف أنّه مفتاحٌ داخلَ JSX نصّاً: `{x.category_icon}`.
 * **ولا يُعترض على تمريره وسيطاً** (`name={x.category_icon}`) — هناك يُحوَّل.
 */
import { readFileSync, globSync } from "node:fs";
import { join } from "node:path";

const ROOT = new URL("..", import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, "$1");

/** حقولٌ قيمتُها مفتاحٌ للآلة لا نصٌّ لإنسان. */
const KEYS = ["category_icon", "categoryIcon"];

const files = globSync("{apps,packages}/**/*.{ts,tsx}", { cwd: ROOT })
  .filter((f) => !f.includes("node_modules") && !f.includes(".next"));

const hits = [];
for (const rel of files) {
  const posix = rel.split("\\").join("/");
  const src = readFileSync(join(ROOT, rel), "utf8");
  src.split("\n").forEach((line, i) => {
    for (const key of KEYS) {
      // `{something.category_icon}` وحدَه في الطفل — لا `name={...}`
      const re = new RegExp(`(^|[^=])\\{\\s*[\\w.?[\\]]*${key}\\s*\\}`);
      if (re.test(line)) hits.push(`${posix}:${i + 1}  ${line.trim().slice(0, 90)}`);
    }
  });
}

if (hits.length) {
  console.error("✗ مفتاحُ آلةٍ يُطبع نصّاً — والقارئُ إنسان:\n");
  hits.forEach((h) => console.error("   " + h));
  console.error(`\n   ${hits.length} موضعاً. استعمل المكوّنَ المركزيّ:`);
  console.error("     <CategoryIcon name={x.category_icon} size={15} />");
  process.exit(1);
}
console.log(`لا مفتاحَ آلةٍ مطبوعاً — فُحص ${files.length} ملفّاً.`);
