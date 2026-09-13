/**
 * ══════════════════════════════════════════════════════════════════════
 * **مركزُ التنزيل — ولا يَعِد بما لا يملك** (`DLC`، ٢٠٢٦-٠٩-١٣)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ما يُحرَس هنا
 *
 * **بنيةُ الويب**: **أنّ الصفحاتَ الخمسَ موجودةٌ عامّةً** · **وأنّ لا
 * رابطَ توزيعٍ مكتوبٌ في شيفرةِ صفحةٍ** · **وأنّ القرارَ من السجلّ.**
 *
 * **وسلوكُ الحالِ يُقاس بالنداء**: **`stateText` و`AppPage` تُقرآن نصّاً
 * هنا**، **والمنطقُ الحاكمُ في المحرّك ويحرسه `TestDLC1..8`.**
 *
 * # ولماذا نصٌّ لا نداء
 *
 * **صفحاتُ Next مكوّناتُ خادمٍ تجلب من المحرّك** — **ونداؤها يحتاج
 * محرّكاً وقاعدةً**، **وذاك عملُ فحوص `qa` وقد فعلته.** **وهذا يحرس
 * ما لا يراه ذاك**: **رابطٌ مكتوبٌ في صفحة.**
 */
import { readFileSync, existsSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const web = join(here, "..");
const app = join(web, "apps/rahalgo/src");

let bad = 0;
const fail = (m) => {
  console.error(`  ✗ ${m}`);
  bad++;
};
const read = (p) => (existsSync(p) ? readFileSync(p, "utf8") : "");

// ── ١ · الصفحاتُ الخمسُ موجودةٌ في القسم العامّ ──────────────────────
//
// **والقسمُ `(site)` هو العامُّ** — **وصفحةٌ في `dashboard` تطلب دخولاً.**
const pages = {
  center: "app/(site)/download/page.tsx",
  customer: "app/(site)/download/customer/page.tsx",
  driver: "app/(site)/download/driver/page.tsx",
  merchant: "app/(site)/download/merchant/page.tsx",
  rep: "app/(site)/download/rep/page.tsx",
};
for (const [name, rel] of Object.entries(pages)) {
  if (!existsSync(join(app, rel))) fail(`صفحةٌ غائبة: ${name} — ${rel}`);
}

// ── ٢ · ولا رابطَ توزيعٍ مكتوبٌ في شيفرةِ صفحة ───────────────────────
//
// **وهو العطبُ الذي كُشف**: **`/app` حملت رابطَ متجرٍ في شيفرتها
// وإعدادُ المتجر فارغٌ في الإنتاج** — **فوعدٌ بمتجرٍ لا يُعرَف أنّه
// فُتح.**
const HARDCODED = [
  [/https:\/\/play\.google\.com/, "رابطُ متجرٍ مكتوبٌ في الشيفرة"],
  [/["'`][^"'`]*\.apk["'`]/, "اسمُ ملفِّ أثرٍ مكتوبٌ في الشيفرة"],
];
const scanned = [
  ...Object.values(pages),
  "components/download/AppCard.tsx",
  "app/(site)/app/page.tsx",
  "lib/releases.ts",
];
for (const rel of scanned) {
  const src = read(join(app, rel));
  if (!src) continue;
  // **والتعليقاتُ تُطرح**: **شرحُ العطب يذكره بالضرورة** — **ونصٌّ
  // يحرس نفسَه من شرحه لا يُكتب شرحُه.**
  const code = src
    .replace(/\/\*[\s\S]*?\*\//g, "")
    .split("\n")
    .filter((l) => !l.trimStart().startsWith("//"))
    .join("\n");
  for (const [re, why] of HARDCODED) {
    if (re.test(code)) fail(`${rel}: ${why}`);
  }
}

// ── ٣ · والقرارُ من السجلّ — موضعٌ واحدٌ يقرأ المحرّك ────────────────
const lib = read(join(app, "lib/releases.ts"));
if (!lib.includes("/api/v1/public/releases"))
  fail("سجلُّ الويب لا يقرأ بابَ المحرّك");
if (!lib.includes("APP_KEYS") || !/customer.*driver.*merchant.*rep/s.test(lib))
  fail("مفاتيحُ التطبيقات الأربعةُ غيرُ معلَنةٍ في موضعٍ واحد");
// **والفشلُ مغلق**: **محرّكٌ لا يُجيب ⇒ «غيرُ متوفّر».**
if (!lib.includes("fallback") || !lib.includes('status: "unavailable"'))
  fail("سقوطُ القراءة لا يُغلق البابَ — **والغيابُ يجب أن يُغلق**");

for (const [name, rel] of Object.entries(pages)) {
  if (name === "center") continue;
  const src = read(join(app, rel));
  if (!src.includes("readRelease(")) fail(`${name}: لا يقرأ السجلَّ`);
  if (!src.includes("AppPage")) fail(`${name}: لا يعرض بالمكوّن المشترك`);
}

// ── ٤ · والحالُ مُسمّاةٌ لا مستنتَجةٌ من نصّ ──────────────────────────
const card = read(join(app, "components/download/AppCard.tsx"));
for (const s of ["unavailable", "play", "direct", "play_and_direct"]) {
  if (!card.includes(s)) fail(`الحالُ ${s} غيرُ مقروءةٍ في العرض`);
}
// **ولا زرَّ متجرٍ يُخترَع** — **والغائبُ يُقال بصراحة.**
if (!card.includes("soonPlay") || !card.includes("soonDirect"))
  fail("لا نصَّ صريحاً لغير المتوفّر — **وزرٌّ ميّتٌ يبدو حيّاً أسوأ**");

// ── ٥ · والنصوصُ من المعجم المركزيّ ─────────────────────────────────
const dict = JSON.parse(
  readFileSync(join(web, "packages/i18n/src/locales/ar.json"), "utf8"),
);
const D = dict.site?.download;
if (!D) fail("لا بابَ `site.download` في المعجم");
else {
  for (const k of [
    "title", "lead", "playCta", "directCta", "soonPlay", "soonDirect",
    "state", "apps", "hashLabel", "officialNote", "installNote",
  ]) {
    if (!(k in D)) fail(`مفتاحٌ ناقصٌ في المعجم: site.download.${k}`);
  }
  for (const k of ["customer", "driver", "merchant", "rep"]) {
    const a = D.apps?.[k];
    if (!a?.name || !a?.short || !a?.long)
      fail(`نصوصُ ${k} ناقصةٌ في المعجم`);
  }
  // **ولا نصيحةٌ تُضعِف أمنَ الجهاز** — (قرارُ المالك ٢٠٢٦-٠٩-١٣).
  const all = JSON.stringify(D);
  for (const danger of ["مصادر غير معروفة", "عطّل", "أوقف الحماية", "من أي مكان"]) {
    if (all.includes(danger))
      fail(`نصٌّ يدفع إلى إضعاف أمن الجهاز: «${danger}»`);
  }
}

// ── ٦ · والمسارُ في خريطة الموقع · ولا حجبَ في robots ───────────────
const sitemap = read(join(app, "app/sitemap.ts"));
for (const p of ["download", "download/customer", "download/driver",
  "download/merchant", "download/rep"]) {
  if (!sitemap.includes(`"${p}"`)) fail(`خريطةُ الموقع لا تحمل ${p}`);
}
// **ولا مجلَّدَ آثارٍ يُعلَن** — **والآثارُ تُخدَم بمفاتيحها من المحرّك،
// ولا مجلَّدَ يُستعرَض.** (**و`download/<key>` صفحاتٌ لا مجلَّد.**)
if (/\.apk|\/downloads\/|uploads/.test(sitemap))
  fail("خريطةُ الموقع تعلن مجلَّدَ آثارٍ أو ملفّاً");

const robots = read(join(app, "app/robots.ts"));
if (/disallow[\s\S]{0,120}download/i.test(robots))
  fail("robots يحجب مركزَ التنزيل — **وهو بابٌ عامّ**");

// ── ٧ · و`/adminrahalgo` لم يُمَسّ ──────────────────────────────────
const portal = read(join(app, "app/adminrahalgo/page.tsx"));
if (!portal.includes('mode="password"'))
  fail("بابُ الإدارة تبدّل — **وهو خارجَ هذه الدورة**");
if (/download/i.test(portal))
  fail("بابُ الإدارة خُلط بمركز التنزيل");

if (bad) {
  console.error(`\n**مركزُ التنزيل مكسور** — ${bad} خرقاً.`);
  process.exit(1);
}
console.log(
  "مركزُ التنزيل: خمسُ صفحاتٍ عامّةٍ · ولا رابطَ في شيفرة · والغيابُ يُغلق.",
);
