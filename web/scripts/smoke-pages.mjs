/**
 * **حارسُ الشاشات — يفتحها في متصفّحٍ حقيقيٍّ ويسقط على أوّل انهيار.**
 *
 * # ولماذا لا يكفي رمزُ الحالة
 *
 * **فحصتُ ٧٧ صفحةً فردَّت كلُّها ٢٠٠** (٢٠٢٦-٠٨-١٠) — **وكانت نصفُها
 * تنهار في المتصفّح.** الخادمُ يرسل الصفحةَ سليمةً، **ثمّ يقع الانهيارُ
 * بعد أن تعمل الشيفرةُ في الجهاز**: `Application error: a client-side
 * exception`.
 *
 * **ورمزُ الحالة أعمى عن ذلك تماماً** — فكان تحقّقي يقول «سليم» عن شاشةٍ
 * بيضاء.
 *
 * # وما كشفه أوّلُ تشغيل
 *
 * **خطأُ React رقم ٣١**: كائنٌ يُرسَم ولداً. **أيقوناتُ lucide تُصنع
 * بـ`forwardRef` فهي كائناتٌ لا دوال**، وفحصٌ بـ`typeof === "function"`
 * يكذب عليها. **و٤٢ ملفّاً يستعمل المكوّنَ الذي كان يفحص هكذا.**
 *
 * # وكيف يُشغَّل
 *
 *   node scripts/smoke-pages.mjs http://localhost:3003
 *
 * **ويُسقط البناءَ في CI** — فلا تُرفع شاشةٌ تنهار.
 */

import { chromium } from "playwright";

const BASE = (process.argv[2] || "http://localhost:3003").replace(/\/+$/, "");

/**
 * **الصفحاتُ العامّةُ وحدَها هنا** — ما خلفَ الدخول يحتاج جلسةً، **وهو
 * عملُ اختبارات E2E لا حارسِ دخان.**
 *
 * **ولوحاتُ الأدوار تُفتح بلا جلسةٍ عمداً**: تُحوّل إلى `/login`، **وهذا
 * نفسُه يُختبر** — فتحويلٌ ينهار أسوأُ من تحويلٍ يقع.
 */
const PAGES = [
  "/",
  "/shop",
  "/login",
  "/signup",
  "/forgot",
  "/cart",
  "/orders",
  "/offers",
  "/favorites",
  "/chats",
  "/notifications",
  "/wallet",
  "/account",
  "/complaints",
  "/help",
  "/contact",
  "/terms",
  "/privacy",
  "/join",
  "/invite",
  "/custom",
  "/dashboard",
  "/store",
  "/rep",
  "/driver",
];

/**
 * **وما يُتجاهَل من الأخطاء** — لا كلُّ سطرٍ أحمرَ عطبٌ في الشاشة.
 *
 * **CORS ونداءٌ فاشلٌ حالُ بيئةٍ لا حالُ شيفرة**: من شغّل الحارسَ على
 * منفذٍ محلّيٍّ والمحرّكُ لا يسمح له **يرى أخطاءَ شبكةٍ صحيحة**، والشاشةُ
 * تعرض «لا اتّصال» كما ينبغي.
 *
 * **والانهيارُ لا يُتجاهَل أبداً** — وهو ما يُصاد هنا.
 */
const IGNORE = [/CORS policy/i, /Failed to load resource/i, /net::ERR_/i, /favicon/i];

const browser = await chromium.launch();
const ctx = await browser.newContext({ locale: "ar" });
const bad = [];
let ok = 0;

for (const path of PAGES) {
  const page = await ctx.newPage();
  const errs = [];
  page.on("pageerror", (e) => errs.push(String(e.message || e).slice(0, 220)));
  page.on("console", (msg) => {
    if (msg.type() !== "error") return;
    const t = msg.text();
    if (!IGNORE.some((re) => re.test(t))) errs.push(t.slice(0, 220));
  });
  try {
    await page.goto(BASE + path, { waitUntil: "networkidle", timeout: 90_000 });
    // **ومهلةٌ بعد السكون** — الانهيارُ يقع في أثرٍ يلي أوّلَ رسم.
    await page.waitForTimeout(1500);
    const body = (await page.textContent("body")) || "";
    if (/Application error|client-side exception/i.test(body)) {
      errs.push("الصفحةُ عرضت شاشةَ انهيار Next");
    }
  } catch (e) {
    errs.push("تعذّر الفتح: " + String(e.message).slice(0, 160));
  }
  await page.close();
  if (errs.length) {
    bad.push({ path, errs: [...new Set(errs)] });
    console.log(`  ✗ ${path}`);
    for (const e of [...new Set(errs)].slice(0, 3)) console.log(`      ${e}`);
  } else {
    ok++;
  }
}

await browser.close();
console.log(`\nحارسُ الشاشات: ${ok} سليمة · ${bad.length} منهارة  (${BASE})`);
process.exit(bad.length ? 1 : 0);
