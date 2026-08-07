/**
 * **حارسُ اللغة — لا حرفَ لاتينيٍّ يُعرض على مستعمل.**
 *
 * (شكوى المالك ٢٠٢٦-٠٨-٠٧: «يوجد الكثير من النصوص الإنكليزيّة، مو بالترجمة
 *  واللغات، بكلّ اللوحات — تحقّق منها».)
 *
 * # ولماذا لا يمسكه حارسُ المركزيّة
 *
 * `check-central` يمنع **نصّاً عربيّاً في الشيفرة** فيُدفع إلى المعجم.
 * **ولا يقول شيئاً عن الإنكليزيّ** — والشيفرةُ كلُّها إنكليزيّة، فلا يُميَّز
 * الاسمُ البرمجيُّ من النصّ المعروض.
 *
 * # والبابُ الذي يدخل منه أخطرُ من الحرفيّ
 *
 * **قيمةٌ خامٌّ من الخادم تُطبع كما هي**: رمزُ حالةٍ أو نوعٍ لا يجد ترجمتَه
 * فيسقط على نفسِه. **ووقع مرّتين اليوم**:
 *
 *   · «استرجاع طلب (cancelled)» — الملاحظةُ تُبنى من ثابت الحالة.
 *   · `order_collection` و`settlement` في صندوق السائق — **والمحرّكُ يكتب
 *     اسماً والمعجمُ يسمّي آخر، فالبحثُ يفشل دائماً.**
 *
 * **ولا يُمسك ذلك بقراءة شيفرة** — النوعان نصّان في طرفين لا يعرف أحدُهما
 * الآخر. **فيُقرأ ما يُعرض على الشاشة نفسِها.**
 *
 * # وما يُستثنى — ولماذا
 *
 * **أرقامٌ وعملاتٌ ومسارات**: لا لغةَ لها. **وأسماءٌ عالميّة** (`WhatsApp`)
 * تُكتب هكذا في كلّ مكان. **وأمثلةُ صيغةٍ في الحقول** (`09xxxxxxxx`) هي
 * الصيغةُ نفسُها لا نصّاً يُترجَم.
 *
 * التشغيل: `node scripts/check-arabic.mjs` — ويحتاج المحرّكَ واللوحاتِ تعمل.
 */
import { chromium } from "playwright-core";

const CHROME = ["C:", "Program Files", "Google", "Chrome", "Application", "chrome.exe"].join(
  String.fromCharCode(92),
);
const API = process.env.API_URL ?? "http://localhost:8080";

/** **كلُّ دورٍ بحسابه وبوّابته** — ولا شاشةَ تُقرأ بلا من يملكها. */
const ROLES = [
  {
    name: "الزبون",
    port: 3003,
    phone: "+963935667788",
    pass: "Zaboon@2026",
    pages: ["/", "/shop", "/orders", "/wallet", "/complaints", "/account", "/notifications", "/favorites", "/offers", "/invite", "/cart"],
  },
  {
    name: "المتجر",
    port: 3002,
    phone: "+963932556677",
    pass: "Matam@2026",
    pages: ["/portal", "/portal/menu", "/portal/reports", "/portal/wallet", "/portal/complaints", "/portal/account", "/portal/notifications"],
  },
  {
    name: "المندوب",
    port: 3004,
    phone: "+963944778899",
    pass: "Mandoub@2026",
    pages: ["/portal", "/portal/link", "/portal/merchants", "/portal/incentives", "/portal/wallet", "/portal/account", "/portal/notifications"],
  },
  {
    name: "السائق",
    port: 3005,
    phone: "+963941112233",
    pass: "Saeq@2026",
    pages: ["/portal", "/portal/incoming", "/portal/history", "/portal/wallet", "/portal/cash", "/portal/incentives", "/portal/reviews", "/portal/complaints", "/portal/account", "/portal/notifications"],
  },
  {
    name: "الإدارة",
    port: 3001,
    phone: "+963999000001",
    pass: "RahalGo@2026",
    pages: ["/dashboard", "/dashboard/orders", "/dashboard/users", "/dashboard/sections", "/dashboard/tickets", "/dashboard/wallet", "/dashboard/reports", "/dashboard/settings", "/dashboard/promos", "/dashboard/ratings", "/dashboard/notifications", "/dashboard/broadcast", "/dashboard/incentives", "/dashboard/claims", "/dashboard/losses", "/dashboard/emergencies", "/dashboard/history", "/dashboard/audit", "/dashboard/leads", "/dashboard/cash", "/dashboard/payouts"],
  },
];

/**
 * **ما يُغتفر** — ولكلٍّ سببُه.
 *
 * **ولا يُوسَّع إلّا بسبب**: قائمةٌ تكبر بلا حجّةٍ تُبطل الحارس.
 */
const OK = [
  /^[\d\s.,:/%+−-]+$/,                 // أرقامٌ وتواريخُ ونسب
  /^(?:AM|PM|ID|QR|URL|API|SMS|OTP|PDF|CSV|GPS|KM|SY[PR]?)$/i,
  /^WhatsApp$/i,                            // اسمٌ عالميٌّ يُكتب هكذا
  /^RahalGo$/i,                             // العلامةُ اللاتينيّة
  /^0?9[x\d]{8,9}$/i,                       // صيغةُ الهاتف مثالاً
  /^RH-[X\d]+$/i,                           // صيغةُ رمزٍ مثالاً
  /^[A-Z]{2,}\d+$/,                         // رموزُ خصمٍ مثالاً: WELCOME50
  /^https?:/,
  /^RH-[A-Z0-9]+$/,
  /^(?:CSV|JPG|PNG|WebP|MB|KB|GB)$/i,
  /^[0-9a-f]{8}-[0-9a-f]{4}-/i,
  /^"?[0-9a-f-]{8,}"?\s*(?:←|→)/,
];

/** **وصفحاتٌ تقنيّةٌ تُستثنى بذاتها** — سجلُّ التدقيق يعرض فروقاً خامّةً
    للأدمن عمداً: معرِّفاتٌ وقيمٌ قبل وبعد، **وترجمتُها تُفسدها.** ويبقى
    اسمُ الفعل مفحوصاً فيه. */
const RAW_PAGES = ["/dashboard/audit"];

/**
 * **رموزٌ لاتينيّةٌ لا تُترجَم — ولو وقعت داخلَ جملةٍ عربيّة.**
 *
 * **«ارفع ملفَّ apk» عربيّةٌ سليمة** كـ«تصدير CSV**
 *
 * **«تصدير CSV» عربيّةٌ سليمة**: `CSV` اسمُ صيغةٍ يُكتب هكذا في كلّ لغة،
 * **وترجمتُه تُفقده معناه.** وكذلك `JPG` و`PNG` و`WhatsApp`.
 *
 * **فتُنزع هذه من النصّ ثمّ يُسأل: أبقيَ حرفٌ لاتينيّ؟** — وهو أدقُّ من
 * مطابقة النصّ كلِّه: **يمسك كلمةً إنكليزيّةً في جملةٍ عربيّة، ويغفر رمزاً.**
 *
 * # والموضعُ النائبُ يُكتب حرفيّاً
 *
 * (وقع ٢٠٢٦-٠٨-٠٧ حين ظهر `whatsapp.otp_template` في اللوحة: شرحُه يقول
 *  «يجب أن يحوي {code}» — **وهو موضعُ الرمز في القالب**، يكتبه المالكُ
 *  كما هو أو لم تحمل الرسالةُ رمزاً.)
 *
 * **وترجمتُه تكسر الميزة** — كما تكسر ترجمةُ `CSV` معناها. فما بين قوسين
 * معقوفين يُنزع كما تُنزع أسماءُ الصيغ.
 */
const TOKENS = /\{[a-z_]+\}|(?:^|[^A-Za-z])(?:CSV|JPG|JPEG|PNG|WebP|GIF|PDF|SVG|MB|KB|GB|QR|SMS|OTP|API|URL|ID|GPS|KM|AM|PM|WhatsApp|RahalGo|SYP|SYR|APK)(?![A-Za-z])/gi;

const b = await chromium.launch({ executablePath: CHROME, headless: true });
let bad = 0;
let pages = 0;
/* **وعطبُ الأداة ليس عطبَ المنتج.**

   (وقع ٢٠٢٦-٠٨-٠٧: ماتت ثلاثةُ خوادمِ تطويرٍ فقال الحارسُ «٤٥ نصّاً
    لاتينيّاً في ١١ صفحة» — **وهي `ERR_CONNECTION_REFUSED` لا كلمةً
    إنكليزيّة.**)

   **وحارسٌ يخلط الاثنين يُطفأ**: يُقرأ سقوطُه كذباً فيُتجاوز، ثمّ يمرّ
   تحته عطبٌ حقيقيّ. **فلكلٍّ عدّادُه ورسالتُه.** */
let dead = 0;

for (const role of ROLES) {
  const ctx = await b.newContext({ locale: "ar", viewport: { width: 1400, height: 950 } });
  const lg = await ctx.request.post(`${API}/api/v1/auth/login`, {
    data: { phone: role.phone, password: role.pass },
  });
  const tok = (await lg.json())?.data?.tokens;
  if (!tok?.access_token) {
    console.log(`✗ ${role.name}: تعذّر الدخول`);
    dead++;
    await ctx.close();
    continue;
  }
  await ctx.addInitScript(
    ([a, r]) => {
      localStorage.setItem("rahalgo_access", a);
      localStorage.setItem("rahalgo_refresh", r);
    },
    [tok.access_token, tok.refresh_token ?? ""],
  );

  for (const path of role.pages) {
    const p = await ctx.newPage();
    try {
      await p.goto(`http://localhost:${role.port}${path}`, { waitUntil: "domcontentloaded", timeout: 180000 });
      await p.waitForTimeout(2200);
      pages++;
      /* **ويُقرأ ما يُرى وحدَه** — عقدةُ نصٍّ في عنصرٍ مخفيٍّ لا يقرؤها أحد،
         **ورفعُها يُغرق الجردَ فيُهمَل.** */
      const found = await p.evaluate(() => {
        const out = [];
        const walk = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
        for (let n = walk.nextNode(); n; n = walk.nextNode()) {
          const t = (n.nodeValue || "").trim();
          if (!t || !/[A-Za-z]{3}/.test(t)) continue;
          const el = n.parentElement;
          if (!el) continue;
          const cs = getComputedStyle(el);
          if (cs.display === "none" || cs.visibility === "hidden" || +cs.opacity === 0) continue;
          if (el.closest("script,style,noscript")) continue;
          const rc = el.getBoundingClientRect();
          if (rc.width === 0 || rc.height === 0) continue;
          out.push({ t: t.slice(0, 40), where: (el.tagName + "." + String(el.className || "").split(" ")[0]).slice(0, 30) });
        }
        return out;
      });
      const raw = RAW_PAGES.includes(path);
      for (const f of found) {
        if (OK.some((re) => re.test(f.t))) continue;
        // **وتُنزع الرموزُ ثمّ يُسأل عمّا بقي** — لا تُطابَق الجملةُ كلُّها.
        if (!/[A-Za-z]{3}/.test(f.t.replace(TOKENS, ""))) continue;
        if (raw && !/^[a-z_]+\.[a-z_]+$/.test(f.t)) continue;
        bad++;
        console.log(`✗ ${role.name.padEnd(8)} ${path.padEnd(26)} «${f.t}»  ${f.where}`);
      }
    } catch (e) {
      // **ولا يُعدّ الفشلُ مرّتين** — كان يزيد `bad` أيضاً فيُقرأ خادمٌ
      // ميّتٌ عشرَ كلماتٍ إنكليزيّة.
      dead++;
      console.log(`✗ ${role.name} ${path}: ${String(e).slice(0, 60)}`);
    }
    await p.close();
  }
  await ctx.close();
}
await b.close();

if (dead) {
  console.log(`\n✗ ${dead} صفحةً لم تُقرأ أصلاً — خادمٌ لا يردّ أو حسابٌ لا يدخل، لا لغة.`);
  console.log("   شغّل المنافذ 3001–3005 ثمّ أعد الفحص.");
}
if (bad) {
  console.log(`\n✗ ${bad} نصّاً لاتينيّاً على الشاشة من ${pages} صفحة.`);
}
if (bad || dead) process.exit(1);
console.log(`✓ لا حرفَ لاتينيٍّ معروضاً في ${pages} صفحة`);
