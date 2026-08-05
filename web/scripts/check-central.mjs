/**
 * **حارسُ المركزية** — يمنع عودةَ ما نُظّف.
 *
 * # لماذا حارسٌ لا جولةُ تنظيفٍ فقط
 *
 * **التنظيفُ يُنقض في أوّل ملفٍّ جديد.** يكتب أحدُنا `text-white` أو
 * `#FD9503` أو `<svg>` مرّةً واحدةً فتعود الفوضى، **ولا يُلاحظ إلّا حين يُطلب
 * تبديلُ اللون فيُبدَّل في مكانٍ ويبقى في عشرة.**
 *
 * **والفحصُ يُشغَّل مع `pnpm lint`** — فمن كسر قاعدةً عرف قبل أن يُودِع.
 *
 * # وسبعُ قواعدَ لا واحدة
 *
 * كلُّ قاعدةٍ تقول **ما الخطأ وأين وبم يُستبدَل** — ورسالةٌ تقول «مخالفة» بلا
 * بديلٍ تُقرأ عائقاً فيُلتفّ عليها.
 */

import { readFileSync, readdirSync, statSync } from "node:fs";
import { join, relative } from "node:path";

const ROOT = new URL("..", import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, "$1");
const SKIP = new Set(["node_modules", ".next", ".turbo", "dist", "build", ".git"]);

/** المصادرُ المسموحُ لها أن تحمل القيمَ الخام — **وهي المركزُ نفسُه.** */
const SOURCES = {
  colors: ["packages/ui/src/theme.css", "packages/ui/src/cssvar.ts"],
  icons: ["packages/ui/src/icons.ts"],
  fonts: ["packages/ui/src/fonts.css"],
};

const PALETTE =
  "slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|" +
  "sky|blue|indigo|violet|purple|fuchsia|pink|rose";

function walk(dir, out = []) {
  for (const name of readdirSync(dir)) {
    if (SKIP.has(name)) continue;
    const p = join(dir, name);
    if (statSync(p).isDirectory()) walk(p, out);
    else out.push(p);
  }
  return out;
}

/** **التعليقاتُ بالعربية أسلوبُ المشروع** — والمقصودُ نصٌّ يبلغ الشاشة. */
function stripComments(src) {
  let out = "";
  for (let i = 0; i < src.length; ) {
    if (src.startsWith("/*", i)) {
      const j = src.indexOf("*/", i + 2);
      const end = j < 0 ? src.length : j + 2;
      // **الأسطرُ تبقى ويُمحى ما فيها.**
      //
      // **وحذفُها يُزيح كلَّ رقمٍ بعده**: يشير الحارسُ إلى سطرٍ بريء، **فيُقرأ
      // كاذباً ويُطفأ** — وحارسٌ لا يُصدَّق أسوأُ من لا حارس.
      out += src.slice(i, end).replace(/[^\n]/g, " ");
      i = end;
    } else if (src.startsWith("//", i)) {
      const j = src.indexOf("\n", i);
      const end = j < 0 ? src.length : j;
      out += " ".repeat(end - i);
      i = end;
    } else {
      out += src[i++];
    }
  }
  return out;
}

const files = walk(ROOT).map((p) => relative(ROOT, p).replace(/\\/g, "/"));
const problems = [];

function report(file, line, rule, found, fix) {
  problems.push({ file, line, rule, found, fix });
}

/** scan يمرّ على أسطر ملفٍّ بعد نزع التعليقات. */
function scan(file, re, rule, fix, { keepComments = false } = {}) {
  const raw = readFileSync(join(ROOT, file), "utf8");
  const src = keepComments ? raw : stripComments(raw);
  src.split("\n").forEach((line, i) => {
    const mm = line.match(re);
    if (mm) report(file, i + 1, rule, mm[0].trim().slice(0, 60), fix);
  });
}

for (const file of files) {
  const isTsx = file.endsWith(".tsx");
  const isCode = isTsx || file.endsWith(".ts");
  const isCss = file.endsWith(".css");

  // ١ · لونٌ خام — **والمركزُ `tokens.ts` و`theme.css` وحدَهما.**
  if ((isCode || isCss) && !SOURCES.colors.includes(file)) {
    scan(file, /#[0-9a-fA-F]{6}\b|rgba?\(\s*\d+/, "لونٌ خام",
      "استعمل توكناً: bg-primary · text-ink · border-line");
  }

  // ٢ · لوحةُ تيلويند الافتراضية — **تتجاوز التوكنز بلا صوت.**
  if (isCode) {
    scan(file, new RegExp(`\\b(?:bg|text|border|ring|from|to|via|fill|stroke|divide)-(?:${PALETTE})-[0-9]{2,3}\\b`),
      "لونٌ من لوحة تيلويند", "استعمل توكناً من theme.css");
    scan(file, /\b(?:bg|text|border)-(?:white|black)\b/, "أبيضُ أو أسودُ خام",
      "text-on-solid فوق تعبئةٍ صلبة · text-ink للنصّ العاديّ · text-shell فوق البرتقاليّ");
  }

  // ٣ · أيقونةٌ مرسومةٌ باليد — **تفترق سماكتُها عن أخواتها.**
  if (isTsx && !SOURCES.icons.includes(file)) {
    scan(file, /<svg[\s>]/, "أيقونةٌ مرسومةٌ باليد", "استوردها من @rahalgo/ui (icons.ts)");
    scan(file, /from ["']lucide-react["']/, "استيرادٌ مباشرٌ من lucide",
      "أضِفها إلى packages/ui/src/icons.ts ثمّ استوردها منه");
    scan(file, /[\u{1F300}-\u{1FAFF}\u{2600}-\u{27BF}]/u, "إيموجي",
      "الإيموجي يُرسم بخطّ النظام فيختلف بين الأجهزة — استعمل أيقونة");
  }

  // ٤ · خطٌّ خارجَ مصدره
  if ((isCss || isCode) && !SOURCES.fonts.includes(file)) {
    scan(file, /@font-face|font-family:\s*[^v\s][^;]*/, "خطٌّ خارجَ fonts.css",
      "الخطُّ يُعرَّف في packages/ui/src/fonts.css ويُقرأ بـ var(--font-sans)");
    if (isCode) scan(file, /\bfont-\[[^\]]+\]/, "خطٌّ عشوائيّ", "استعمل var(--font-sans)");
  }

  // ٥ · نصٌّ عربيٌّ خارجَ المعجم — **وترجمةٌ لا تُجمع لا تُترجَم.**
  if (isTsx && !file.startsWith("packages/i18n/")) {
    scan(file, /[؀-ۿ]/, "نصٌّ عربيٌّ في الشيفرة",
      "انقله إلى packages/i18n/src/locales/ar.json واقرأه بـ getMessages");
  }

  // ٦ · شبكةٌ لا تنهار على الجوّال — **ثلاثةُ أعمدةٍ على ٣٦٠ بكسلاً لا تُقرأ.**
  if (isTsx) {
    const raw = stripComments(readFileSync(join(ROOT, file), "utf8"));
    raw.split("\n").forEach((line, i) => {
      const mm = line.match(/\bgrid-cols-([3-9]|1[0-2])\b/);
      if (mm && !/(sm|md|lg|xl|2xl):grid-cols-/.test(line)) {
        report(file, i + 1, "شبكةٌ بلا أساسٍ للجوّال", mm[0],
          "ابدأ بعمودٍ أو عمودين ثمّ وسّع: grid-cols-1 sm:grid-cols-3");
      }
    });
  }

  // ٧ · عرضٌ ثابتٌ يفيض عن الجوّال
  if (isTsx) {
    scan(file, /\b(?:w|min-w)-\[(?:[4-9]\d{2}|\d{4,})px\]/, "عرضٌ ثابتٌ يفيض",
      "استعمل max-w-* أو w-full — والجوّالُ ٣٢٠ بكسلاً");
  }
}

// ٩ · **صنفٌ يُسمّي توكناً لا وجودَ له.**
//
// # وهو أخطرُ ما في الباب
//
// كُتب `rounded-input` في نافذتين — **وليس في الثيم**. فلا خطأً في البناء ولا
// تحذيراً ولا شيء: **الصنفُ يُقرأ سليماً ويُرسم لا شيء**، وحقلُ الشكوى مربّعُ
// الأركان وكلُّ حقلٍ في المنصة مستدير.
//
// **وخطأٌ صامتٌ لا يُكتشف إلّا بالعين** — ولا أحدَ يفتح كلَّ نافذةٍ في كلّ
// إصدار.
//
// **ويُفحص ما نملكه لا ما تملكه تيلويند**: `rounded-card` لنا و`rounded-full`
// لها — فتُقرأ أسماءُ التوكنز من `theme.css` **ولا يُحكم على ما ليس منها.**
{
  const css = readFileSync(join(ROOT, "packages/ui/src/theme.css"), "utf8");
  const known = {
    color: new Set([...css.matchAll(/--color-([a-z0-9-]+):/g)].map((x) => x[1])),
    radius: new Set([...css.matchAll(/--radius-([a-z0-9-]+):/g)].map((x) => x[1])),
    text: new Set([...css.matchAll(/--text-([a-z0-9-]+):/g)].map((x) => x[1])),
    shadow: new Set([...css.matchAll(/--shadow-([a-z0-9-]+):/g)].map((x) => x[1])),
    drop: new Set([...css.matchAll(/--drop-shadow-([a-z0-9-]+):/g)].map((x) => x[1])),
  };
  // البادئاتُ التي تُشتقّ من توكناتنا، وما تقابله من مجموعات.
  // **وجهاتُ الزوايا من تيلويند لا منّا**: `rounded-t` و`rounded-se` وأخواتُها
  // **تُقصّ قبل الفحص** — وإلّا أبلغ الحارسُ عن صنفٍ سليمٍ فيُقرأ كاذباً.
  const SIDES = "t|b|s|e|l|r|tl|tr|bl|br|ss|se|es|ee";
  const PREFIX = [
    // **والجهةُ وحدَها صنفٌ قائم** (`rounded-t`) — فتُقبل كما تُقبل `full`.
    [new RegExp(`\\brounded(?:-(?:${SIDES}))?-([a-z][a-z0-9-]*)\\b`, "g"),
      known.radius, "radius", ["full", "none", ...SIDES.split("|")]],
    [/\bdrop-shadow-([a-z][a-z0-9-]*)\b/g, known.drop, "drop-shadow", ["none", "sm", "md", "lg", "xl"]],
  ];
  for (const file of files.filter((f) => f.endsWith(".tsx") || f.endsWith(".ts"))) {
    if (SOURCES.colors.includes(file)) continue;
    const src = stripComments(readFileSync(join(ROOT, file), "utf8"));
    src.split("\n").forEach((line, i) => {
      for (const [re, set, label, builtin] of PREFIX) {
        re.lastIndex = 0;
        let mm;
        while ((mm = re.exec(line))) {
          const name = mm[1];
          if (set.has(name) || builtin.includes(name)) continue;
          report(file, i + 1, `صنفٌ يسمّي توكناً غيرَ موجود`, mm[0],
            `لا يوجد --${label}-${name} في theme.css — والمتاح: ${[...set].join(" · ")}`);
        }
      }
    });
  }
}

// ١٠ · **مُحوِّرٌ على صنفٍ لا تعرفه تيلويند.**
//
// `hover:elev-2` كُتبت في تسعة مواضعَ **ولم تفعل شيئاً**: `.elev-2` كان صنفاً
// عاديّاً في `theme.css`، **وتيلويند لا تولّد مُحوِّراً لما لا تملكه.**
//
// **وهو الخطأُ الصامتُ نفسُه**: يُقرأ سليماً ويُرسم لا شيء — ولا بناءَ يصرخ.
//
// **والعلاجُ `@utility`** — فتسجّله عندها ويقبل المُحوِّرات.
{
  const css = readFileSync(join(ROOT, "packages/ui/src/theme.css"), "utf8");
  // ما سُجّل بـ `@utility` يقبل المُحوِّرات، وما كُتب `.صنف {` لا يقبلها.
  const asUtility = new Set([...css.matchAll(/@utility\s+([a-z][a-z0-9-]*)/g)].map((x) => x[1]));
  const asPlain = new Set(
    [...css.matchAll(/^\.([a-z][a-z0-9-]*)\s*\{/gm)].map((x) => x[1]).filter((c) => !asUtility.has(c)),
  );
  if (asPlain.size) {
    const re = new RegExp(`\\b[a-z-]+:(${[...asPlain].join("|")})\\b`, "g");
    for (const file of files.filter((f) => f.endsWith(".tsx"))) {
      const src = stripComments(readFileSync(join(ROOT, file), "utf8"));
      src.split("\n").forEach((line, i) => {
        re.lastIndex = 0;
        let mm;
        while ((mm = re.exec(line))) {
          report(file, i + 1, "مُحوِّرٌ على صنفٍ لا يقبله", mm[0],
            `اجعله @utility في theme.css — الصنفُ العاديُّ لا تولّد له تيلويند مُحوِّراً`);
        }
      });
    }
  }
}

// ٨ · **ولا لوحةَ ألوانٍ ثانيةٍ تعود.**
//
// كانت في `tokens.ts` لوحةٌ كاملةٌ ومثلُها في `theme.css`، وتعليقُها يقول
// «يُعدَّل هناك ثمّ يُعكس هنا» — **وهذا لا يقع أبداً**: بُدِّل الثيمُ إلى
// الداكن وبقيت هي على الفاتح، **فصارت الخريطةُ ترسم بألوان منصّةٍ أخرى.**
//
// **والمرآةُ لا تُحرَس بالتذكير بل بألّا تكون** — فالثيمُ مصدرٌ واحدٌ ومن
// احتاج اللونَ قيمةً يقرؤه بـ`themeColor`.
{
  const ts = readFileSync(join(ROOT, "packages/ui/src/tokens.ts"), "utf8");
  const stray = [...ts.matchAll(/^\s+([a-zA-Z]+):\s*"(#[0-9a-fA-F]{3,8})"/gm)];
  for (const s of stray) {
    problems.push({
      file: "packages/ui/src/tokens.ts",
      line: ts.slice(0, s.index).split("\n").length,
      rule: "لوحةُ ألوانٍ ثانيةٌ عادت",
      found: `${s[1]}: ${s[2]}`,
      fix: "الألوانُ في theme.css وحدَه — واقرأها قيمةً بـ themeColor من cssvar.ts",
    });
  }
}

// ── التقرير ──────────────────────────────────────────────────────────────
if (problems.length === 0) {
  console.log("المركزيةُ سليمة — لا لونَ خارجَ التوكنز ولا أيقونةَ خارجَ المركز ولا نصَّ خارجَ المعجم.");
  process.exit(0);
}

const byRule = new Map();
for (const p of problems) {
  if (!byRule.has(p.rule)) byRule.set(p.rule, []);
  byRule.get(p.rule).push(p);
}
console.error(`المركزية: ${problems.length} مخالفةً في ${byRule.size} قاعدة\n`);
for (const [rule, list] of byRule) {
  console.error(`── ${rule} (${list.length})`);
  console.error(`   الإصلاح: ${list[0].fix}`);
  for (const p of list.slice(0, 12)) console.error(`   ${p.file}:${p.line}  ${p.found}`);
  if (list.length > 12) console.error(`   … و${list.length - 12} غيرها`);
  console.error("");
}
process.exit(1);
