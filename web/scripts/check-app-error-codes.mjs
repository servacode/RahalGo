/**
 * ══════════════════════════════════════════════════════════════════════
 * **كلُّ رمزِ خطأٍ يصل التطبيقَ له عربيّة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (شكوى المالك ٢٠٢٦-٠٨-١٨: «أرسل رمزاً لحذف حسابي لا يعمل».)
 *
 * # ما كان يقع
 *
 * **المحرّكُ يردّ سبباً صحيحاً** (`open_orders`: «لديك طلبٌ جارٍ»)،
 * **ولا ترجمةَ له في خريطة التطبيق** — فيسقط إلى عرض الرمز الخام،
 * **ويقرأ صاحبُه لاتينيّةً لا تعني له شيئا.**
 *
 * **وخمسةُ رموزٍ لحذف الحساب كانت كذلك**، ومثلُها `invalid_otp` قبلها.
 *
 * # ولماذا حارسٌ لا مراجعة
 *
 * **الرمزُ يُضاف في المحرّك ولا يُفتح ملفُّ أندرويد** — **ولا يظهر
 * العطبُ إلّا حين يقع الخطأُ عند مستعمِلٍ حقيقيّ**، وهو نادرٌ في
 * التجربة وكثيرٌ في الاستعمال.
 *
 * # وما لا يُحرَس
 *
 * **رموزٌ لا تبلغ التطبيقَ أصلاً** — أبوابُ الإدارة واللوحات. **وحارسٌ
 * يطلب ترجمةَ ما لا يُعرض يُملأ بنصوصٍ لا يقرؤها أحد**، فيُهمَل.
 */
import { readFileSync, readdirSync, statSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const here = dirname(fileURLToPath(import.meta.url));
const repo = join(here, "..", "..");

const map = readFileSync(
  join(repo, "mobile/ui/src/main/kotlin/com/rahalgo/ui/ApiErrors.kt"),
  "utf8",
);
/** **ما تعرفه الخريطة** — `"code" to R.string.x`. */
const known = new Set(
  [...map.matchAll(/"([a-z0-9_]+)" to R\.string\./g)].map((m) => m[1]),
);

/**
 * **الحزمُ التي تخدم التطبيقات** — أخطاؤها تصل شاشةَ هاتف.
 *
 * **و`server` تُستثنى**: فيها أبوابُ الإدارة واللوحات، **ورمزٌ لا يبلغ
 * هاتفاً لا يُطلب له نصٌّ في أندرويد.**
 */
const PACKAGES = ["identity", "orders", "wallet", "catalog", "cashbox"];

const found = new Map();
for (const pkg of PACKAGES) {
  const dir = join(repo, "backend/internal", pkg);
  for (const name of readdirSync(dir)) {
    const full = join(dir, name);
    if (statSync(full).isDirectory() || !name.endsWith(".go")) continue;
    if (name.endsWith("_test.go")) continue;
    const src = readFileSync(full, "utf8");
    for (const m of src.matchAll(/httpx\.NewError\([^,]+,\s*"([a-z0-9_]+)"/g)) {
      if (!found.has(m[1])) found.set(m[1], `${pkg}/${name}`);
    }
  }
}

const missing = [...found].filter(([code]) => !known.has(code));
if (missing.length > 0) {
  console.error("رموزُ خطأٍ يردّها المحرّكُ بلا عربيّةٍ في التطبيق:");
  for (const [code, where] of missing) {
    console.error(`  · «${code}» (${where}) — يُعرض خامّاً على شاشةٍ عربيّة`);
  }
  process.exit(1);
}
console.log(
  `رموزُ الخطأ مترجَمة — ${found.size} رمزاً يردّها المحرّكُ وكلُّها بعربيّة.`,
);
