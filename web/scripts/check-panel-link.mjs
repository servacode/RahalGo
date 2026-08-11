/**
 * **يفحص زرَّ «لوحتي» في الشريط السفليّ — أيقود إلى لوحةٍ موجودةٍ فعلاً؟**
 *
 * (سؤالُ المالك ٢٠٢٦-٠٨-١١: «تحقّق إذا كان زرُّ لوحتي يعمل بشكلٍ صحيحٍ بعد
 *  نقله إلى أسفل الشاشة».)
 *
 * # ولماذا لا يُقرأ من الشيفرة
 *
 * **كتبتُ `String(homeFor(user.roles))` — و`homeFor` تعيد كائناً** فيه أصلٌ
 * ومسار. **فصار الرابطُ `[object Object]`**: زرٌّ في أكثر المواضع ضغطاً عند
 * السائق والمتجر والمندوب **يقود إلى صفحةٍ غيرِ موجودة.**
 *
 * **ولا تحقّقُ الأنواعِ يمسكها** (`String` تقبل كلَّ شيء)، **ولا حارسُ
 * الشاشات** (الصفحةُ تُطلى سليمةً، والعطبُ في وجهةِ رابطٍ فيها).
 *
 * **فيُشغَّل التطبيقُ ويُزرَع مستخدمٌ بكلّ دورٍ ويُقرأ الرابطُ من الصفحة
 * المطلوّة** — وهو الشيءُ الوحيدُ الذي كان سيمسكه.
 *
 * يحتاج خادماً يعمل:  node scripts/check-panel-link.mjs http://localhost:3200
 */
import { chromium } from "playwright";

const BASE = process.argv[2] ?? "http://localhost:3200";
const ROLES = [
  ["admin", "/dashboard"],
  ["merchant", "/store"],
  ["sales", "/rep"],
  ["driver", "/driver"],
  ["customer", null], // الزبونُ لا لوحةَ له — تبقى «الرئيسيّة»
];

const b = await chromium.launch();
const ctxFor = () => b.newContext({ viewport: { width: 390, height: 800 } });
let bad = 0;

for (const [role, want] of ROLES) {
  const ctx = await ctxFor(role);
  const pg = await ctx.newPage();
  // **كلُّ نداءٍ للمحرّك يُردّ محلّيّاً** — الفحصُ للواجهة لا للخادم.
  await pg.route("**/api/v1/**", (route) => {
    const u = route.request().url();
    if (u.includes("/auth/me"))
      return route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ data: { id: "u1", phone: "0900000000", full_name: "فحص", roles: [role] } }),
      });
    return route.fulfill({ status: 200, contentType: "application/json", body: '{"data":{}}' });
  });
  await pg.addInitScript(() => {
    localStorage.setItem("rahalgo_access", "x.y.z");
    localStorage.setItem("rahalgo_refresh", "r.r.r");
  });
  await pg.goto(BASE, { waitUntil: "domcontentloaded" });
  await pg.waitForTimeout(1500);

  const nav = await pg.evaluate(() => {
    const bar = document.querySelector("nav");
    if (!bar) return null;
    return [...bar.querySelectorAll("a")].map((a) => ({
      href: a.getAttribute("href"),
      label: (a.textContent || "").trim(),
    }));
  });

  if (!nav) {
    console.log(`  ${role}: لا شريطَ سفليّ (لعلّ الجلسةَ لم تُقبل)`);
    bad++;
    await ctx.close();
    continue;
  }
  const first = nav[0];
  const ok = want ? first.href === want : first.href === "/";
  if (!ok) bad++;
  console.log(
    `  ${ok ? "✓" : "✗"} ${role.padEnd(9)} أوّلُ بندٍ: ${JSON.stringify(first)}` +
      (want ? `  (المنتظَر ${want})` : "  (المنتظَر / — الرئيسيّة)"),
  );
  await ctx.close();
}

await b.close();
console.log(bad ? `\n${bad} حالةً مخالفة` : "\nزرُّ «لوحتي» سليمٌ في كلّ دور.");
process.exit(bad ? 1 : 0);
