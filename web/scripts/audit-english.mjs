/**
 * **جردُ النصوص الإنكليزيّة التي تصل الشاشة.**
 *
 * (شكوى المالك ٢٠٢٦-٠٨-٠٧: «يوجد الكثير من النصوص الإنكليزيّة، مو بالترجمة
 *  واللغات، بكلّ اللوحات — تحقّق منها».)
 *
 * # ولماذا لم يمسكها حارسُ المركزيّة
 *
 * **الحارسُ يمنع نصّاً عربيّاً في الشيفرة** — فيُدفع إلى المعجم. **ولا يقول
 * شيئاً عن الإنكليزيّ**، لأنّ الشيفرةَ كلَّها إنكليزيّةٌ فلا يُميَّز الاسمُ
 * البرمجيُّ من النصّ المعروض.
 *
 * # وثلاثةُ أبوابٍ يدخل منها
 *
 * ١ · **نصٌّ حرفيٌّ في JSX**: `<span>Loading</span>`.
 * ٢ · **خاصّيّةٌ نصّيّةٌ معروضة**: `placeholder="Search"` · `title="Print"`.
 * ٣ · **قيمةٌ خامٌّ من الخادم تُطبع كما هي**: حالةُ طلبٍ أو نوعُ حركةٍ لا
 *     يمرّ على خريطةِ أسماء — **وهو ما رآه المالك في محفظته** (`cancelled`).
 *
 * والثالثُ لا يُمسك بالبحث النصّيّ، **فيُعدّ الأوّلان ويُشار إلى مواضع
 * الثالث بالطباعة المباشرة لحقول الحالة.**
 */
import { readdirSync, statSync, readFileSync } from "node:fs";
import { join, relative, sep } from "node:path";

const ROOT = process.cwd();
const SKIP = new Set(["node_modules", ".next", "dist", ".turbo", "coverage"]);
const walk = (d, out = []) => {
  for (const e of readdirSync(d)) {
    if (SKIP.has(e)) continue;
    const p = join(d, e);
    if (statSync(p).isDirectory()) walk(p, out);
    else if (e.endsWith(".tsx")) out.push(p);
  }
  return out;
};

/** أسماءٌ برمجيّةٌ لا نصوصٌ — تُستثنى لئلّا يغرق الجردُ في ضجيج. */
const CODEY =
  /^(?:[a-z]+(?:[A-Z][a-z]*)+|[A-Z_]+|https?:.*|\/[\w/[\]-]*|#[0-9a-f]+|[\w.-]+@[\w.-]+|[\d\s.,:%+-]*|ltr|rtl|auto|none|true|false|px|rem|em|sm|md|lg|xl)$/;

const cases = [];
for (const abs of walk(ROOT)) {
  const f = relative(ROOT, abs).split(sep).join("/");
  const src = readFileSync(abs, "utf8");
  const lines = src.split("\n");
  lines.forEach((line, i) => {
    // **والتعليقاتُ تُتجاوز** — شرحٌ إنكليزيٌّ لا يصل الشاشة.
    const t = line.trim();
    if (t.startsWith("//") || t.startsWith("*") || t.startsWith("/*")) return;

    // ١ · خاصّيّاتٌ نصّيّةٌ معروضة
    for (const m of line.matchAll(/\b(placeholder|title|aria-label|alt|label)="([^"]{2,})"/g)) {
      const v = m[2];
      if (!/[a-zA-Z]{3}/.test(v) || CODEY.test(v)) continue;
      if (/[؀-ۿ]/.test(v)) continue;
      cases.push([f, i + 1, m[1], v]);
    }
    // ٢ · نصٌّ حرفيٌّ بين وسمين
    for (const m of line.matchAll(/>\s*([A-Za-z][A-Za-z ,.'!?-]{3,})\s*</g)) {
      const v = m[1].trim();
      if (CODEY.test(v)) continue;
      // **والجنريكاتُ ليست نصّاً**: `Promise<T>` و`Record<K,V>` تُطابق الشكلَ
      // نفسَه — **ومسبارٌ يعدّها يُغرق الجردَ فيُهمَل.** (كشفه أوّلُ تشغيل.)
      if (/^(?:Promise|Record|Array|Map|Set|Partial|Omit|Pick|ReadonlyArray)$/.test(v)) continue;
      cases.push([f, i + 1, "نصّ", v]);
    }
  });
}

const bySec = new Map();
for (const [f] of cases) {
  const sec = f.startsWith("apps/") ? f.split("/")[1] : "packages/" + f.split("/")[1];
  bySec.set(sec, (bySec.get(sec) ?? 0) + 1);
}
console.log("النصوصُ الإنكليزيّةُ التي قد تصل الشاشة:", cases.length, "\n");
for (const [s, n] of [...bySec].sort((a, b) => b[1] - a[1])) console.log("  " + String(n).padStart(4) + "  " + s);
console.log("");
for (const [f, ln, kind, v] of cases.slice(0, 40)) console.log(`  ${f}:${ln}  [${kind}] ${v.slice(0, 58)}`);
if (cases.length > 40) console.log(`  … و${cases.length - 40} غيرها`);
