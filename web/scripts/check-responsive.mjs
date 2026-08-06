/**
 * **حارسُ التجاوب — لا صفحةَ تنزلق أفقيّاً على جوّال.**
 *
 * (شكوى المالك ٢٠٢٦-٠٨-٠٧: «التجاوبُ على الموبايل وكلّ الجوّالات والشاشات
 *  أبداً مو مضبوط».)
 *
 * # ولماذا حارسٌ حيٌّ لا قاعدةُ أصناف
 *
 * **الفيضُ الأفقيُّ ليس صنفاً واحداً يُمنع.** جاء اليومَ من شبكةٍ بلا عمودٍ
 * أساسيّ، ومن صفِّ أزرارٍ لا يلتفّ — **ولا يجمعهما نمطٌ نصّيّ.** وغداً يأتي
 * من ثالثٍ لم يخطر ببال.
 *
 * **وحارسُ الأصناف يمنع ما عرفناه، وهذا يمنع ما لم نعرفه بعد.**
 *
 * # ولا يُقاس على صفحةٍ فارغة
 *
 * **أوّلُ قياسٍ قال «لا فيض» وهو يفحص صفحاتٍ فارغة** — كلُّ نداءٍ مردودٌ
 * بـ`{}`. **والفيضُ يأتي من المحتوى**: بطاقةُ طلبٍ فيها أصنافٌ وشريطُ تتبّعٍ
 * وأزرار. فيدخل بحساب الزراعة ويقرأ ما يقرأه صاحبُ الحساب.
 *
 * # والقياسُ عنصرٌ عنصرٌ لا مستنداً واحداً
 *
 * `scrollWidth` يقول «فيها فيض» **ولا يقول من**. فيُمشى على كلّ عنصرٍ
 * ويُرفع من تجاوز الحافّة — **فيُصلَح السببُ لا العَرَض.**
 *
 * التشغيل: `node scripts/check-responsive.mjs` — ويحتاج المحرّكَ وموقعَ
 * الزبون يعملان.
 */
import { chromium } from "playwright-core";

const CHROME = ["C:", "Program Files", "Google", "Chrome", "Application", "chrome.exe"].join(
  String.fromCharCode(92),
);
const WEB = process.env.WEB_URL ?? "http://localhost:3003";
const API = process.env.API_URL ?? "http://localhost:8080";
const PHONE = process.env.SEED_PHONE ?? "+963935667788";
const PASS = process.env.SEED_PASSWORD ?? "Zaboon@2026";

/** **المقاساتُ التي يُباع بها الهاتفُ في الرقّة** — لا مقاساتُ المصمّمين. */
const SIZES = [
  [320, 640, "أضيق"],
  [360, 740, "جالاكسي"],
  [390, 844, "آيفون"],
  [768, 1024, "لوح"],
  [1024, 768, "لابتوب"],
];

const PAGES = [
  "/", "/shop", "/orders", "/wallet", "/complaints", "/account",
  "/notifications", "/favorites", "/offers", "/invite", "/cart", "/help",
];

const b = await chromium.launch({ executablePath: CHROME, headless: true });
const ctx = await b.newContext({ locale: "ar" });

const login = await ctx.request.post(`${API}/api/v1/auth/login`, {
  data: { phone: PHONE, password: PASS },
});
const tok = (await login.json())?.data?.tokens;
if (!tok?.access_token) {
  console.error("تعذّر الدخول بحساب الزراعة — شغّل المحرّك وازرع البيانات أوّلاً.");
  await b.close();
  process.exit(2);
}
await ctx.addInitScript(
  ([a, r]) => {
    localStorage.setItem("rahalgo_access", a);
    localStorage.setItem("rahalgo_refresh", r);
  },
  [tok.access_token, tok.refresh_token ?? ""],
);

let bad = 0;
let checked = 0;
for (const page of PAGES) {
  const p = await ctx.newPage();
  try {
    await p.setViewportSize({ width: 360, height: 740 });
    await p.goto(WEB + page, { waitUntil: "domcontentloaded", timeout: 180000 });
    await p.waitForTimeout(1400);
    for (const [w, h, name] of SIZES) {
      await p.setViewportSize({ width: w, height: h });
      await p.waitForTimeout(350);
      checked++;
      const r = await p.evaluate((vw) => {
        const over = [];
        for (const el of document.querySelectorAll("body *")) {
          const rc = el.getBoundingClientRect();
          if (rc.width === 0 || rc.height === 0) continue;
          const past = Math.round(Math.max(rc.right - vw, -rc.left));
          if (past <= 1) continue;
          const cs = getComputedStyle(el);
          if (cs.position === "fixed" && cs.visibility === "hidden") continue;
          over.push({
            t: el.tagName.toLowerCase(),
            c: String(el.className || "").slice(0, 46),
            past,
            w: Math.round(rc.width),
          });
        }
        over.sort((a, b2) => b2.past - a.past);
        return { doc: document.documentElement.scrollWidth, n: over.length, over: over.slice(0, 3) };
      }, w);
      /* ══════════════════════════════════════════════════════════════
         **وزخرفةٌ تقع فوق نصِّ حقلِها**
         ══════════════════════════════════════════════════════════════

         (شهده المالك ٢٠٢٦-٠٨-٠٧: «الأيقونةُ فايتة برقم الهاتف».)

         **الأيقونةُ تُوضع فوق الحقل بموضعٍ مطلق، والحشوةُ تُخلي لها مكاناً.**
         فإن حُسبت الحشوةُ لجهةٍ والأيقونةُ لأخرى **وقع النصُّ فوقها** —
         ولا يظهر ذلك في فيضٍ ولا في نوعٍ ولا في صنف.

         **فيُقاس التقاطعُ نفسُه**: صندوقُ محتوى الحقل بعد الحشوة، وصندوقُ
         الزخرفة — **وأيُّ تداخلٍ عطب.** */
      const clash = await p.evaluate(() => {
        const out = [];
        for (const inp of document.querySelectorAll("input, textarea")) {
          const wrap = inp.parentElement;
          if (!wrap) continue;
          const rc = inp.getBoundingClientRect();
          if (rc.width === 0) continue;
          const cs = getComputedStyle(inp);
          const box = {
            l: rc.left + parseFloat(cs.paddingLeft),
            r: rc.right - parseFloat(cs.paddingRight),
          };
          for (const dec of wrap.children) {
            if (dec === inp) continue;
            const dcs = getComputedStyle(dec);
            if (dcs.position !== "absolute") continue;
            const d = dec.getBoundingClientRect();
            if (d.width === 0) continue;
            const over = Math.min(box.r, d.right) - Math.max(box.l, d.left);
            if (over > 1) out.push({ id: inp.id || inp.name || "(بلا اسم)", over: Math.round(over) });
          }
        }
        return out;
      });
      for (const c of clash) {
        bad++;
        console.log(`✗ ${page.padEnd(15)} ${name.padEnd(8)} ${w}px → زخرفةٌ فوق نصِّ الحقل «${c.id}» بمقدار ${c.over}px`);
      }

      if (r.doc <= w + 1) continue;
      bad++;
      console.log(`✗ ${page.padEnd(15)} ${name.padEnd(8)} ${w}px → المستند ${r.doc} (فيضٌ ${r.doc - w}) · ${r.n} عنصراً`);
      for (const o of r.over) console.log(`     ${String(o.past).padStart(5)}px  <${o.t}> w=${o.w}  ${o.c}`);
    }
  } catch (e) {
    bad++;
    console.log(`✗ ${page}: ${String(e).slice(0, 70)}`);
  }
  await p.close();
}
await b.close();

if (bad) {
  console.log(`\n✗ ${bad} حالةَ فيضٍ من ${checked} — الصفحةُ تنزلق أفقيّاً.`);
  process.exit(1);
}
console.log(`✓ لا فيضَ أفقيّاً في ${checked} حالة (${PAGES.length} صفحةً × ${SIZES.length} مقاسات)`);
