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

/**
 * ══════════════════════════════════════════════════════════════════════
 * **و`server` ليست كلُّها إدارة** — XG-45
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كان استثناؤها كاملاً** — **فسقط منها ما يمرّ به كلُّ نداءٍ من
 * هاتف**: `auth_unavailable` يخرج من وسيط التوثيق نفسِه، **فقُرئ
 * لاتينيّةً على شاشةٍ عربيّة ولم يمسكه حارس.**
 *
 * **فصار التصنيفُ صريحاً لا ضمنيّاً**: **كلُّ رمزٍ يولد في `server`
 * إمّا معلَنٌ أنّه يبلغ الهاتف فيُطلَب له نصّ، وإمّا معلَنٌ أنّه
 * للإدارة وحدَها.** **ومجهولُ التصنيف يوقف الحارس** — **فلا يُضاف
 * رمزٌ في المحرّك ويمرّ بلا قرار.**
 */
const SERVER_MOBILE = new Set([
  // **وبابٌ لم يُفتح بعد** — **يبلغ الهواتفَ الأربعةَ كلَّها**: الزبونُ
  // يقرؤه عند الطلب، والسائقُ عند بدء الدوام، والمندوبُ عند ضمّ متجر.
  // **وهو غيرُ «ممنوع»**: لا يقول «لستَ أهلاً» بل «ليس الآن».
  "launch_closed",
  // ══════════════════════════════════════════════════════════════════
  // **ونيّةُ التوسّع تبلغ الزبونَ** (`CR`، ٢٠٢٦-٠٩-١٤)
  // ══════════════════════════════════════════════════════════════════
  //
  // **و«صارت الخدمةُ متاحة» خبرٌ سارٌّ لا عطب** — **يقع حين يوسّع
  // المكتبُ المنطقةَ بين قراءة الشاشة وضغطة الزبون.**
  //
  // **و«حالٌ لا توافق» يقع حين تتبدّل الحالُ تحت الزرّ** — **والشاشةُ
  // تُحدِّث نفسَها ولا تتّهم صاحبَها.**
  "service_now_available",
  "reason_mismatch",
  "auth_unavailable", "idempotency_reclaimed", "too_many_addresses",
  "already_returned", "merchant_no_returns", "order_not_returnable",
  "not_readyable", "reason_required", "never_picked_up",
  "payout_below_min", "payout_closed", "payout_not_allowed",
  "invite_required", "lead_already_converted", "bad_bbox",
  // **وتعذّرُ بناءِ حمولةٍ آمنة** — يبلغ الزبونَ والمتجرَ لأنّ
  // البابَ بابُهما، **ونصُّه نصُّ العطب العامّ**: لا فعلَ لصاحب
  // الجهاز فيه، **ورسالةٌ تصفُ داخلَنا تُقلق ولا تُفيد.**
  "payload_unsafe",
]);

/** **ما لا يبلغ هاتفاً** — أبوابُ الإدارة واللوحات. */
const SERVER_ADMIN = new Set([
  "app_file_missing", "app_not_android", "app_too_large", "bad_json",
  "bad_placement", "city_bad_point", "city_bad_radius", "city_bad_reach",
  "city_has_merchants", "city_needs_name", "claim_already_settled",
  "dispute_party_has_no_wallet", "division_in_use",
  "division_needs_governorate", "division_needs_name",
  "driver_already_compensated", "driver_has_open_orders", "duplicate_name",
  "goods_already_settled", "goods_flow_changed", "goods_ledger_mismatch",
  "order_has_no_driver", "order_not_failed", "order_still_open",
  "role_exists", "section_has_items", "step_up_invalid", "step_up_required",
   "transfer_same_merchant",
  "transfer_too_late", "bad_channel", "no_merchant_phone",
  // ── staging-only QA fixtures (qa/* endpoints; never reach a real mobile client) ──
  "qa_bad_target", "qa_fault_injected", "qa_flag_not_allowed",
  "qa_needs_custom_cash", "qa_no_backward", "qa_no_driver",
  "qa_no_driver_on_order", "qa_no_free_item", "qa_no_gov_for_point",
  "qa_normal_accepted_only", "qa_normal_cash_only", "qa_not_customer_only",
  "qa_not_qa_order", "qa_order_off_ladder", "qa_phone_not_allowed",
  "qa_seed_kind_not_allowed", "qa_no_option",
  "qa_not_rep_only", "qa_money_key_not_allowed", "qa_release_key_not_allowed",
  "qa_no_open_order", "qa_multiple_open_orders",
  "qa_coverage_zone_exists",
  "qa_no_closed_order",
  "qa_boundary_near_midnight", "qa_no_default_address", "qa_no_zone_for_address",
]);

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

// ── ورموزُ `server`: مصنَّفةٌ صراحةً أو يقف الحارس ──────────────────
const serverDir = join(repo, "backend/internal/server");
const serverCodes = new Map();
for (const name of readdirSync(serverDir)) {
  if (!name.endsWith(".go") || name.endsWith("_test.go")) continue;
  const src = readFileSync(join(serverDir, name), "utf8");
  for (const m of src.matchAll(
    /httpx\.NewError\([\s\S]*?"([a-z0-9_]+)"\s*,\s*"[a-z0-9_.]+"/g,
  )) {
    if (!serverCodes.has(m[1])) serverCodes.set(m[1], `server/${name}`);
  }
}
const unclassified = [...serverCodes].filter(
  ([code]) =>
    !SERVER_MOBILE.has(code) && !SERVER_ADMIN.has(code) && !known.has(code),
);
if (unclassified.length > 0) {
  console.error("رموزُ خطأٍ في `server` بلا تصنيف — أتبلغ الهاتفَ أم لا؟");
  for (const [code, where] of unclassified) {
    console.error(`  · «${code}» (${where}) — يُصنَّف في SERVER_MOBILE أو SERVER_ADMIN`);
  }
  process.exit(1);
}
for (const code of SERVER_MOBILE) {
  if (!found.has(code)) found.set(code, serverCodes.get(code) ?? "server");
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
