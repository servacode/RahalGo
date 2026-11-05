#!/usr/bin/env node
/**
 * ══════════════════════════════════════════════════════════════════════
 *  لوحة المتجر على الموقع تقول نفس ما يقوله التطبيق
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قاعدة المالك: أي تعديل بالتطبيقات ينعمل بالموقع كمان.)
 *
 * # شو بيتحرس
 *
 *	١ · **سطر الدين تحت الرصيد** في صفحة المحفظة (/store/wallet). كان غايب عن الموقع
 *	    كله، فصاحب المتجر على الآيفون ما بيشوف دين «تعديل طلب بعد الاستلام» اللي
 *	    بيشوفه صاحبه على أندرويد. النص وأسباب الدين لازم يطابقوا `WalletScreen.kt`.
 *	    ومكانه تحت بطاقة الرصيد مباشرة وقبل شريط التبويب (متل التطبيق) — مو بعده،
 *	    وإلا على الجوّال بيبين كأنه من محتوى التبويب.
 *	٢ · **عمر الطلب** في صفحة طلبات المتجر (/store/orders) بكلمات كاملة:
 *	    «منذ 5 دقائق»، «منذ ساعتين»، وبعد ٣ أيام التاريخ — نفس `Since.text`.
 *	    مو «5 د» و«2 س».
 *	٣ · **المستلمين السابقين** بنموذج «اطلب توصيلة» (/store/delivery، قرار المالك
 *	    ٢٠٢٦-١١-٠٥): حرفين بالاسم أو الرقم بتطلع قائمة تحت الحقل، واللمسة بتعبّي كلشي.
 *	    نفس القواعد (`RecipientSuggest.kt` ↔ `recipientSuggest.ts`) ونفس الكلام
 *	    (`md_saved_*` بـ strings.xml تبع تطبيق المتجر ↔ `store.delivery.saved*`)،
 *	    والصفحتين فعلاً بيندهوا `delivery-recipients`.
 *
 * # ليش المقارنة مع التطبيق نفسه
 *
 * النصوص بتنقرأ من `strings.xml` وأسباب الدين بتنقرأ من `WalletScreen.kt`
 * (دالة `debtCause`). فمين أضاف سبب دين بالتطبيق ونسي الموقع بيسقط الفحص،
 * ومين غيّر كلمة بواحد منهم بيسقط عليه الثاني.
 *
 * والدوال المفحوصة هي الحقيقية: `fmtSince` من `@rahalgo/i18n` و`debtLineText`
 * من `packages/ui/src/debtText.ts` — مو نسخة منها.
 */
import { readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { register } from "node:module";

const here = dirname(fileURLToPath(import.meta.url));
const web = join(here, "..");
const repo = join(web, "..");
register(pathToFileURL(join(here, "_tsload.mjs")).href);

let fails = 0;
const fail = (msg) => {
  fails++;
  console.error(`✗ ${msg}`);
};
const eq = (name, got, want) => {
  if (got !== want) fail(`${name}\n   got:  ${JSON.stringify(got)}\n   want: ${JSON.stringify(want)}`);
};
const read = (p) => {
  try {
    return readFileSync(p, "utf8");
  } catch {
    return "";
  }
};
/** يشيل التعليقات — الفحص على الشيفرة مو على كلام عنها. */
const strip = (s) => s.replace(/\{\/\*[\s\S]*?\*\/\}/g, "").replace(/\/\*[\s\S]*?\*\//g, "").replace(/\/\/.*$/gm, "");
const load = async (path) => {
  try {
    return await import(pathToFileURL(path).href);
  } catch (e) {
    fail(`تعذّر تحميل ${path.replace(web, "web")}: ${e.code ?? e.message}`);
    return {};
  }
};

// ── نصوص التطبيق: strings.xml هو الأصل ───────────────────────────────
const xml = read(join(repo, "mobile", "ui", "src", "main", "res", "values", "strings.xml"));
const android = (name) => {
  const m = xml.match(new RegExp(`<string name="${name}">([\\s\\S]*?)</string>`));
  if (!m) {
    fail(`النص ${name} مو موجود بـ strings.xml`);
    return "";
  }
  return m[1];
};
/** مثل `getString`: يملأ %1$s و%1$d. */
const fill = (tpl, ...args) => tpl.replace(/%(\d)\$[sd]/g, (_, i) => String(args[Number(i) - 1]));
const dict = JSON.parse(read(join(web, "packages", "i18n", "src", "locales", "ar.json")));

const i18n = await load(join(web, "packages", "i18n", "src", "index.ts"));
const debt = await load(join(web, "packages", "ui", "src", "debtText.ts"));

// ══════════════════════════════════════════════════════════════════════
// ٢ · عمر الطلب — `Since.text`
// ══════════════════════════════════════════════════════════════════════
const NOW = Date.parse("2026-11-05T12:00:00Z");
const ago = (min, sec = 0) => new Date(NOW - (min * 60 + sec) * 1000).toISOString();
/** صيغة العدد عند التطبيق (`Since.form`): ١ مفرد، ٢ مثنّى، ٣–١٠ جمع، وما فوق تمييز. */
const form = (n) => (n === 1 ? "one" : n === 2 ? "two" : n >= 3 && n <= 10 ? "few" : "many");
const appSince = (unit, n) => fill(android("ord_since"), fill(android(`since_${unit}_${form(n)}`), n));

if (typeof i18n.fmtSince !== "function") {
  fail("fmtSince غير موجودة بـ @rahalgo/i18n — الموقع لسّا بيكتب العمر بالحروف");
} else {
  const since = (iso) => i18n.fmtSince(iso, NOW);
  eq("عشر ثوان بتنقرأ دقيقة", since(ago(0, 10)), appSince("min", 1));
  eq("دقيقتين", since(ago(2)), appSince("min", 2));
  eq("5 دقائق", since(ago(5)), appSince("min", 5));
  eq("10 دقائق", since(ago(10)), appSince("min", 10));
  eq("11 دقيقة", since(ago(11)), appSince("min", 11));
  eq("59 دقيقة", since(ago(59)), appSince("min", 59));
  eq("60 دقيقة = ساعة", since(ago(60)), appSince("hr", 1));
  eq("130 دقيقة = ساعتين", since(ago(130)), appSince("hr", 2));
  eq("5 ساعات", since(ago(5 * 60)), appSince("hr", 5));
  eq("23 ساعة وكسر", since(ago(23 * 60 + 59)), appSince("hr", 23));
  eq("24 ساعة وشوي = يوم", since(ago(24 * 60 + 5)), appSince("day", 1));
  eq("48 ساعة وشوي = يومين", since(ago(48 * 60 + 5)), appSince("day", 2));
  eq("قبل ما يكمل 3 أيام لسّا يومين", since(ago(72 * 60 - 1)), appSince("day", 2));
  // وقت بالمستقبل (ساعة الجهاز تسبق الخادم) بيتقرأ دقيقة، مو «منذ ناقص ثلاث».
  eq("المستقبل دقيقة", since(ago(-3)), appSince("min", 1));
  eq("الإزاحة الزمنية بتنقرأ", since("2026-11-05T14:00:00+03:00"), appSince("hr", 1));
  eq("وقت رقم (ms) بيمشي", i18n.fmtSince(NOW - 5 * 60_000, NOW), appSince("min", 5));
  // من 3 أيام وطالع: تاريخ ووقت بتوقيت دمشق (+3)، بأرقام لاتينية — مو «منذ ...».
  eq("3 أيام = تاريخ", since(ago(72 * 60)), "2026-11-02 15:00");
  eq("تاريخ بيوم دمشق مو يوم UTC", since("2026-10-31T21:30:00Z"), "2026-11-01 00:30");
  // وقت ما بينفهم ما بينخمّن.
  for (const bad of ["", "لا وقت", "2026-13-45"]) eq(`وقت فاسد «${bad}» بيرجع فاضي`, since(bad), "");
  // وما في حروف مختصرة بأي جواب.
  for (const m of [1, 5, 11, 90, 60 * 30]) {
    const t = since(ago(m));
    if (/\d\s*[دسي]\s*$/.test(t)) fail(`العمر بالحروف المختصرة: «${t}»`);
  }
}

// ══════════════════════════════════════════════════════════════════════
// ١ · سطر الدين — `DebtLineText` و`debtCause`
// ══════════════════════════════════════════════════════════════════════
const walletKt = read(join(repo, "mobile", "ui", "src", "main", "kotlin", "com", "rahalgo", "ui", "WalletScreen.kt"));
const fnStart = walletKt.indexOf("private fun debtCause(");
const fnBody = fnStart < 0 ? "" : walletKt.slice(fnStart, walletKt.indexOf("\n}\n", fnStart));
/** `"a", "b" -> stringResource(R.string.wal_debt_x)` — كل سطر سبب (أو أكتر) ونصه. */
const rows = [...fnBody.matchAll(/((?:"[a-z_]+"\s*,\s*)*"[a-z_]+")\s*->\s*stringResource\(R\.string\.(wal_debt_[a-z_]+)\)/g)];
const elseName = fnBody.match(/else\s*->\s*stringResource\(R\.string\.(wal_debt_[a-z_]+)\)/)?.[1];
const appCauses = rows.flatMap((r) => [...r[1].matchAll(/"([a-z_]+)"/g)].map((c) => [c[1], r[2]]));

// حارس يعمى صامت أسوأ من لا حارس: لو ما انقرأ سبب من التطبيق نوقف.
if (appCauses.length < 9 || !elseName) fail(`ما قدرت أقرأ أسباب الدين من WalletScreen.kt (قرأت ${appCauses.length})`);
if (!appCauses.some(([c]) => c === "order_edit")) fail("سبب order_edit مو موجود بـ WalletScreen.kt");

if (typeof debt.debtCauseText !== "function" || typeof debt.debtLineText !== "function") {
  fail("debtCauseText/debtLineText غير موجودتين — الموقع ما بيعرض سطر الدين");
} else {
  for (const [cause, name] of appCauses) {
    eq(`سبب الدين ${cause}`, debt.debtCauseText(cause), android(name));
  }
  eq("سبب مجهول = «دين سابق» متل التطبيق", debt.debtCauseText("some_future_cause"), android(elseName ?? "wal_debt_old"));

  const money = (n) => `${n.toLocaleString("en-US")} ${dict.common.currency}`;
  // الأسباب المتكررة بتنكتب مرة (distinct) وبتنفصل بالفاصلة العربية.
  const debts = [
    { amount: 40_000, cause: "refund_merchant_earning" },
    { amount: 25_000, cause: "refund_rep_commission" },
    { amount: 85_000, cause: "order_edit" },
  ];
  const causes = [android("wal_debt_refund"), android("wal_debt_order_edit")].join(dict.common.listSep);
  eq("سطر الدين كامل", debt.debtLineText(150_000, debts), fill(android("wal_debt_line"), money(150_000), causes));
  eq("دين order_edit لوحده", debt.debtLineText(85_000, [debts[2]]), fill(android("wal_debt_line"), money(85_000), android("wal_debt_order_edit")));
  eq("ما في دين = ما في سطر", debt.debtLineText(0, []), "");
}

// ══════════════════════════════════════════════════════════════════════
// الصفحات فعلاً بتستعمل هالدوال
// ══════════════════════════════════════════════════════════════════════
const walletPage = strip(read(join(web, "packages", "ui", "src", "WalletPage.tsx")));
if (!walletPage.includes("debt_total") || !/\bdebtLineText\(/.test(walletPage)) {
  fail("WalletPage.tsx ما بتقرأ debt_total ولا بتعرض سطر الدين — صاحب المتجر على الموقع ما بيشوف دينه");
}

// ── مكان السطر: تحت بطاقة الرصيد مباشرة، قبل شريط التبويب (متل WalletScreen.kt) ──
// على الجوّال الصف عمودي: لو انحط بعد `<Tabs` بيبين كأنه من محتوى التبويب وبيضل ظاهر
// وأنت بـ«طلبات السحب» — ومتجر عنده طلب سحب هو بالضبط اللي بيطلعله الشريط وعليه الدين.
const iBalance = walletPage.indexOf("{balanceLabel}");
const iDebt = walletPage.indexOf('data-testid="wallet-debt"');
const iTabs = walletPage.indexOf("<Tabs");
if (iBalance < 0 || iDebt < 0 || iTabs < 0) {
  fail("ما لقيت بطاقة الرصيد أو سطر الدين أو <Tabs> بـ WalletPage.tsx — الحارس صار أعمى");
} else {
  if (!(iBalance < iDebt && iDebt < iTabs)) {
    fail("سطر الدين مو تحت بطاقة الرصيد مباشرة: لازم يجي بعدها وقبل <Tabs> (متل WalletScreen.kt)، وإلا على الجوّال بيبين من محتوى التبويب");
  }
  // وشرطه `debtText` وحده — مو `payouts` ولا `cardsOnly`: المتجر والزبون بيشاركوا الصفحة.
  if (!/\{debtText && \(\s*<p\b[^>]*data-testid="wallet-debt"/.test(walletPage)) {
    fail("شرط ظهور سطر الدين لازم يكون debtText وحده — مو payouts ولا cardsOnly، لأن المتجر والزبون بيشاركوا الصفحة");
  }
  // وعلى الشاشة العريضة بينزل لسطر كامل تحت البطاقة والتبويب — مو عمود ثالث بينهم.
  const classesOf = (tag) => (tag.match(/className="([^"]*)"/)?.[1] ?? "").split(/\s+/);
  const debtTag = walletPage.match(/<p\b[^>]*data-testid="wallet-debt"[^>]*>/)?.[0] ?? "";
  const cardAt = walletPage.lastIndexOf('<div className="surface', iBalance);
  const rowTag = walletPage.slice(0, cardAt).match(/<div className="[^"]*">\s*$/)?.[0] ?? "";
  for (const cls of ["sm:order-last", "sm:basis-full"]) {
    if (!classesOf(debtTag).includes(cls)) fail(`سطر الدين ما عليه ${cls} — على الشاشة العريضة بيقع عمود ثالث بين البطاقة والتبويب`);
  }
  if (!classesOf(rowTag).includes("sm:flex-wrap")) {
    fail("صف الرصيد والتبويب ما عليه sm:flex-wrap — سطر الدين ما بينزل لسطر كامل على الشاشة العريضة");
  }
}

const ordersPage = strip(read(join(web, "apps", "rahalgo", "src", "app", "(store)", "store", "orders", "page.tsx")));
if (/\bfmtSpan\b/.test(ordersPage)) fail("صفحة طلبات المتجر لسّا بتكتب عمر الطلب بـ fmtSpan («5 د» و«2 س») والتطبيق بالكلمات");
if (!/\bfmtSince\(/.test(ordersPage)) fail("صفحة طلبات المتجر ما بتستعمل fmtSince لعمر الطلب");

// ══════════════════════════════════════════════════════════════════════
// ٣ · المستلمين السابقين — `RecipientSuggest.kt` و`md_saved_*`
// ══════════════════════════════════════════════════════════════════════
const merchXml = read(join(repo, "mobile", "app-merchant", "src", "main", "res", "values", "strings.xml"));
const merchant = (name) => {
  const m = merchXml.match(new RegExp(`<string name="${name}">([\\s\\S]*?)</string>`));
  if (!m) {
    fail(`النص ${name} مو موجود بـ strings.xml تبع تطبيق المتجر`);
    return "";
  }
  return m[1];
};
const DD = dict.store?.delivery ?? {};
for (const [app, web] of [
  ["md_saved_title", "savedTitle"],
  ["md_saved_hide", "savedHide"],
  ["md_saved_has_point", "savedHasPoint"],
  ["md_saved_point", "savedPoint"],
  ["md_saved_point_fetch", "savedPointFetch"],
  ["md_saved_point_remove", "savedPointRemove"],
]) {
  eq(`نص ${app} = store.delivery.${web}`, DD[web], merchant(app));
}
eq(
  "سطر المستلم (الرقم — العنوان)",
  String(DD.savedLine ?? "").replace(/\{(\w+)\}/g, (_, k) => ({ phone: "0933111222", address: "الثكنة" })[k] ?? ""),
  fill(merchant("md_saved_line"), "0933111222", "الثكنة"),
);

const suggestKt = read(join(repo, "mobile", "shared", "src", "main", "kotlin", "com", "rahalgo", "shared", "merchant", "RecipientSuggest.kt"));
const ktConst = (name) => Number(suggestKt.match(new RegExp(`const val ${name} = (\\d+)L?`))?.[1] ?? NaN);
const rs = await load(join(web, "apps", "rahalgo", "src", "lib", "store", "recipientSuggest.ts"));
if (typeof rs.shouldSearchRecipients !== "function" || typeof rs.localPhone !== "function") {
  fail("recipientSuggest.ts ما فيها shouldSearchRecipients/localPhone — الموقع ما بيقترح المستلمين");
} else {
  eq("انتظار ما بعد آخر حرف متل التطبيق", rs.RECIPIENT_DEBOUNCE_MS, ktConst("DEBOUNCE_MS"));
  eq("أقل عدد حروف متل التطبيق", rs.RECIPIENT_MIN_CHARS, ktConst("MIN_CHARS"));
  for (const [text, want] of [["", false], ["أ", false], [" ٠ ", false], ["أح", true], ["09", true], ["٠٩", true]]) {
    eq(`بحث «${text}»`, rs.shouldSearchRecipients(text), want);
  }
  eq("+963 بيصير 09", rs.localPhone("+963933111222"), "0933111222");
  eq("رقم أجنبي بيضل متل ما هو", rs.localPhone("+90532123456"), "+90532123456");
  const f = rs.recipientFill?.({ name: "أحمد", phone: "+963933111222", address_text: "الثكنة", lat: null, lng: 1, last_used_at: "", times: 1 });
  eq("بلا نقطة كاملة ما في نقطة", f?.point ?? null, null);
  // **النقطة المحفوظة بتسقط إذا تبدّل الرقم أو العنوان** — نفس حالات `RecipientSuggestTest`.
  if (typeof rs.samePhone !== "function" || typeof rs.keepsSavedPoint !== "function") {
    fail("recipientSuggest.ts ما فيها samePhone/keepsSavedPoint — النقطة القديمة بتضل لغير صاحبها");
  } else {
    eq("09 = +963", rs.samePhone("0933111222", "+963933111222"), true);
    eq("هندي بمسافات = 00963", rs.samePhone("٠٩٣٣ ١١١ ٢٢٢", "00963933111222"), true);
    eq("رقمين مختلفين", rs.samePhone("0933111222", "0933111223"), false);
    eq("فاضي ما بيساوي فاضي", rs.samePhone("", ""), false);
    const picked = { name: "أحمد", phone: "0933111222", address: "الثكنة", point: { lat: 35.95, lng: 39.01 } };
    eq("نفس الرقم بصيغة تانية بتضل", rs.keepsSavedPoint(picked, "+963933111222", "الثكنة"), true);
    eq("رقم تاني بتسقط", rs.keepsSavedPoint(picked, "0933111223", "الثكنة"), false);
    eq("رقم عم ينمحى بتسقط", rs.keepsSavedPoint(picked, "093311122", "الثكنة"), false);
    eq("عنوان تعدّل بتسقط", rs.keepsSavedPoint(picked, "0933111222", "الثكنة، البناية الثانية"), false);
  }
  if (!/fun keepsSavedPoint\(/.test(suggestKt)) fail("RecipientSuggest.kt ما فيها keepsSavedPoint — التطبيق بيخلّي النقطة القديمة");
}

const deliveryPage = strip(read(join(web, "apps", "rahalgo", "src", "app", "(store)", "store", "delivery", "page.tsx")));
const recipientsLib = strip(read(join(web, "apps", "rahalgo", "src", "lib", "store", "recipients.ts")));
const deliveryKt = strip(read(join(repo, "mobile", "app-merchant", "src", "main", "kotlin", "com", "rahalgo", "merchant", "delivery", "DeliveryScreen.kt")));
const deliveriesApiKt = read(join(repo, "mobile", "shared", "src", "main", "kotlin", "com", "rahalgo", "shared", "merchant", "DeliveriesApi.kt"));
if (!/\buseRecipientSuggest\(/.test(deliveryPage) || !/<RecipientSuggestions\b/.test(deliveryPage)) {
  fail("صفحة «اطلب توصيلة» ما بتعرض المستلمين السابقين — التطبيق بيعرضهن");
}
if (!recipientsLib.includes("/delivery-recipients")) fail("lib/store/recipients.ts ما بيندَه delivery-recipients");
if (!/\bkeepsSavedPoint\(/.test(deliveryPage)) fail("صفحة «اطلب توصيلة» ما بتسقّط النقطة المحفوظة لما يتبدّل الرقم أو العنوان");
if (!/\bkeepsSavedPoint\(/.test(deliveryKt)) fail("DeliveryScreen.kt ما بيسقّط النقطة المحفوظة لما يتبدّل الرقم أو العنوان");
if (!/\bRecipientSuggestions\(vm\)/.test(deliveryKt)) fail("DeliveryScreen.kt ما بيعرض المستلمين السابقين — الموقع بيعرضهن");
if (!deliveriesApiKt.includes("/delivery-recipients")) fail("DeliveriesApi.kt ما بيندَه delivery-recipients");

if (fails > 0) {
  console.error(`\n✗ لوحة المتجر والتطبيق: ${fails} فحصاً سقط`);
  process.exit(1);
}
console.log("✓ لوحة المتجر والتطبيق: سطر الدين وعمر الطلب والمستلمين السابقين بنفس الكلام");
