#!/usr/bin/env node
/**
 * ══════════════════════════════════════════════════════════════════
 * **عقدُ الموارد — مقيسٌ من الناتج لا مكتوبٌ باليد**
 * ══════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦أ، قرارُ المالك ٢٠٢٦-٠٨-٢١، البند ١٠.)
 *
 * **أمرُ المالك نصّاً**: «Manifest للموارد يحتوي: اسم الخطّ ونسخته
 * وترخيصه وبصمته · نطاقات الحروف المبنيّة · الأيقونات · بصمة».
 *
 * # **ولماذا يُقاس ولا يُكتب**
 *
 * **العقدُ الذي يُكتب باليد يصف نيّةً لا واقعاً.** فلو سقط نطاقٌ من
 * البناء **بقي مذكوراً في العقد**، ولو بُدّل خطٌّ بقيت بصمتُه القديمة
 * — **وصار العقدُ يشهد لما لم يُبنَ.**
 *
 * **فكلُّ حقلٍ هنا مقروءٌ من القرص بعد البناء.**
 *
 * # **وقائمةُ الملفّات هي ما يجلبه الهاتف**
 *
 * (المرحلة ٦ب — إغلاقُ `TD-RESOURCE-FETCH`، ٢٠٢٦-٠٨-٢١.)
 *
 * **كان العقدُ يصف مجموعاتٍ لا ملفّات**: عددَ النطاقات ومجموعَ
 * حجمِها، **وبصمةَ الأيقونات والنمط دونَ الحروف.** فالهاتفُ يعرف أنّ
 * ثمّة اثنَي عشرَ ملفَّ حرفٍ **ولا يعرف عنوانَ واحدٍ منها ولا بصمتَه.**
 *
 * **فأُضيف `files[]`** — سطرٌ لكلّ ملفٍّ يُنزَّل: مسارُه النسبيُّ
 * وحجمُه وبصمتُه. **وهو ما يجعل الجلبَ تحقُّقاً لا ثقة.**
 *
 * **والمسارُ نسبيٌّ داخلَ حزمة الموارد** — لا مطلقٌ ولا صاعد،
 * **ويُفحص في الهاتف قبل أن يصير ملفّاً.**
 *
 *   الاستعمال:  node resources-manifest.mjs <مجلّد الموارد>
 */
import { createHash } from 'node:crypto';
import { readFileSync, readdirSync, statSync, existsSync } from 'node:fs';
import { join, sep } from 'node:path';
import { requiredRanges } from './glyph-ranges.mjs';

const outDir = process.argv[2];
if (!outDir) {
  console.error('الاستعمال: node resources-manifest.mjs <مجلّد الموارد>');
  process.exit(2);
}

const sha256 = (p) => createHash('sha256').update(readFileSync(p)).digest('hex');
const bytes = (p) => statSync(p).size;

/** **الإعدادُ يُقرأ من مصدره** — لا يُكرَّر في الشيفرة. */
function readEnv(path) {
  const env = {};
  for (const line of readFileSync(path, 'utf8').split(/\r?\n/)) {
    const m = /^([A-Z_][A-Z0-9_]*)=(.*)$/.exec(line.trim());
    if (m) env[m[1]] = m[2].replace(/^["']|["']$/g, '');
  }
  return env;
}
const env = readEnv(new URL('../config/resources.env', import.meta.url));

// ── الخطوطُ والحروف ───────────────────────────────────────────────
const glyphDir = join(outDir, 'glyphs');
const stacks = readdirSync(glyphDir)
  .filter((e) => statSync(join(glyphDir, e)).isDirectory())
  .sort();

const required = requiredRanges();

const fontstacks = stacks.map((name) => {
  const dir = join(glyphDir, name);
  const ttf = join(outDir, 'fonts', `${name}.ttf`);

  /**
   * **والنطاقاتُ المُعلَنةُ هي المطلوبةُ الموجودةُ غيرُ الفارغة.**
   *
   * **لا كلُّ ما في المجلّد** — المولّدُ يكتب ٢٥٦ ملفّاً لكامل المستوى
   * الأساسيّ، **وأكثرُها فارغٌ لخطٍّ عربيّ.** فالعقدُ يُعلن ما يُعتمد
   * عليه، **والباقي وزنٌ لا وعد.**
   */
  const built = required.filter((r) => {
    const f = join(dir, `${r}.pbf`);
    return existsSync(f) && statSync(f).size > 0;
  });

  return {
    name,
    family: env.FONT_FAMILY,
    version: env.FONT_VERSION,
    license: env.FONT_LICENSE,
    licenseUrl: env.FONT_LICENSE_URL,
    source: existsSync(ttf) ? { bytes: bytes(ttf), sha256: sha256(ttf) } : null,
    glyphRanges: built,
    glyphBytes: readdirSync(dir)
      .filter((f) => f.endsWith('.pbf'))
      .reduce((n, f) => n + bytes(join(dir, f)), 0),
    // **والناقصُ يُعلَن ولا يُطمس** — البناءُ يسقط قبله، فإن ظهر هنا
    // فالعقدُ وُلّد بيدٍ خارجَ المسار.
    missingRanges: required.filter((r) => !built.includes(r)),
  };
});

// ── الأيقونات ─────────────────────────────────────────────────────
const spriteJson = join(outDir, 'sprite.json');
const icons = existsSync(spriteJson)
  ? Object.keys(JSON.parse(readFileSync(spriteJson, 'utf8'))).sort()
  : [];

const spriteFiles = ['sprite.json', 'sprite.png', 'sprite@2x.json', 'sprite@2x.png']
  .filter((f) => existsSync(join(outDir, f)))
  .map((f) => ({ file: f, bytes: bytes(join(outDir, f)), sha256: sha256(join(outDir, f)) }));

// ── النمطُ بارتباطاته ─────────────────────────────────────────────
const styles = ['style.online.json', 'style.offline.json']
  .filter((f) => existsSync(join(outDir, f)))
  .map((f) => ({ file: f, bytes: bytes(join(outDir, f)), sha256: sha256(join(outDir, f)) }));

const licenseFile = join(outDir, 'fonts', 'LICENSE.txt');

/**
 * **كلُّ ملفٍّ يحتاجه الهاتف** — بمساره النسبيّ وحجمه وبصمته.
 *
 * **ولا يُدرَج ملفُّ الخطّ نفسُه** (`fonts/*.ttf`): **الهاتفُ يعرض
 * الحروفَ المرسومةَ مسبقاً `.pbf` ولا يفتح خطّاً.** فهو أداةُ بناءٍ
 * لا مورِدَ تشغيل — **ونصفُ ميغابايت لا داعيَ لحملها إلى الهاتف.**
 *
 * **ونصُّ الرخصة يُدرَج** — شرطُ OFL أن يُوزَّع مع ما بُني منه.
 */
function resourceFiles() {
  const out = [];
  const add = (rel) => {
    const abs = join(outDir, rel);
    if (!existsSync(abs)) return;
    out.push({ path: rel.split(sep).join('/'), bytes: bytes(abs), sha256: sha256(abs) });
  };

  for (const name of stacks) {
    for (const range of required) add(join('glyphs', name, `${range}.pbf`));
  }
  for (const f of ['sprite.json', 'sprite.png', 'sprite@2x.json', 'sprite@2x.png']) add(f);
  add('style.offline.json');
  add(join('fonts', 'LICENSE.txt'));
  return out;
}

const files = resourceFiles();

process.stdout.write(`${JSON.stringify(
  {
    resourcesVersion: env.RESOURCES_VERSION,
    styleVersion: env.STYLE_VERSION,
    glyphVersion: env.GLYPH_VERSION,
    spriteVersion: env.SPRITE_VERSION,
    // **عقدُ العنوان مُعلَنٌ** (البند ١١) — فمن ربط النمطَ عرف الشكل
    // دونَ أن يقرأ سكربتاً.
    glyphUrlTemplate: '{fontstack}/{range}.pbf',
    fontstacks,
    fontLicense: existsSync(licenseFile)
      ? {
          spdx: env.FONT_LICENSE,
          file: 'fonts/LICENSE.txt',
          bytes: bytes(licenseFile),
          sha256: sha256(licenseFile),
        }
      : null,
    sprite: { icons, files: spriteFiles },
    styles,
    /**
     * **وقائمةُ ما يُنزَّل** — البند ٢ من إغلاق ٦ب الوظيفيّ.
     *
     * **مسارٌ نسبيٌّ وحجمٌ وبصمةٌ لكلّ ملفّ**، **فالجالبُ يتحقّق من
     * كلّ واحدٍ على حدة** ولا يُفعّل نسخةً نقص منها ملفّ.
     */
    files,
    totalBytes: files.reduce((n, f) => n + f.bytes, 0),
  },
  null,
  2,
)}\n`);
