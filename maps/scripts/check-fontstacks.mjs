#!/usr/bin/env node
/**
 * ══════════════════════════════════════════════════════════════════
 * **بوّابةُ أسماء الرصّات — العقدُ بين النمط والمجلّد**
 * ══════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦أ، قرارُ المالك ٢٠٢٦-٠٨-٢١، البندان ٤ و١١.)
 *
 * **عنوانُ الحروف `{fontstack}/{range}.pbf`** — و`{fontstack}` هو نصّاً
 * ما في `text-font`. **فإن أنتج المولّدُ مجلّداً باسمٍ آخرَ ردَّ الخادمُ
 * ٤٠٤**، **والخريطةُ تُرسم بلا اسمِ شارعٍ واحد** — والملفّاتُ كلُّها
 * موجودةٌ سليمةٌ على القرص.
 *
 * **وهو أخبثُ من عطبٍ يسقط**: لا خطأَ في سجلٍّ ولا استثناء، **بلاطاتٌ
 * تُرسم وأسماءٌ تغيب.**
 *
 * **ووقع في أوّل بناءٍ حقيقيّ** (٢٠٢٦-٠٨-٢١): `build_pbf_glyphs` يسمّي
 * المجلّدَ **باسم ملفّ الخطّ** لا بعائلته الداخليّة، فأنتج
 * `RahalGo-Regular` والنمطُ يطلب `RahalGo Regular`.
 *
 *   الاستعمال:  node check-fontstacks.mjs <مجلّد الموارد> <النمط>
 */
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join } from 'node:path';

const [outDir, stylePath] = process.argv.slice(2);
if (!outDir || !stylePath) {
  console.error('الاستعمال: node check-fontstacks.mjs <مجلّد الموارد> <النمط>');
  process.exit(2);
}

const style = JSON.parse(readFileSync(stylePath, 'utf8'));

/** **كلُّ رصّةٍ يطلبها النمط** — من كلّ طبقةٍ فيها `text-font`. */
const wanted = new Set();
for (const layer of style.layers ?? []) {
  for (const name of layer.layout?.['text-font'] ?? []) wanted.add(name);
}

const glyphDir = join(outDir, 'glyphs');
const built = new Set(
  readdirSync(glyphDir).filter((e) => statSync(join(glyphDir, e)).isDirectory()),
);

let bad = 0;

for (const name of [...wanted].sort()) {
  if (built.has(name)) {
    console.log(`  ✓ ${name}`);
    continue;
  }
  bad += 1;
  console.error(`  ✗ النمطُ يطلب «${name}» ولا مجلّدَ بهذا الاسم`);
  // **والاقتراحُ يوفّر ربعَ ساعة** — الفرقُ حرفٌ في الغالب.
  const near = [...built].find(
    (b) => b.replace(/[-_ ]/g, '').toLowerCase() === name.replace(/[-_ ]/g, '').toLowerCase(),
  );
  if (near) console.error(`    والموجودُ «${near}» — الفرقُ في الفاصل`);
}

/**
 * **والزائدُ يُذكر ولا يُسقط.**
 *
 * **مجلّدٌ لا يطلبه النمطُ ليس عطباً** — لكنّه وزنٌ يُحمل في كلّ حزمةٍ
 * دونَ اتّصال، **فيُرى ليُقرَّر.**
 */
for (const name of [...built].sort()) {
  if (!wanted.has(name)) console.log(`  · «${name}» مبنيٌّ ولا يطلبه النمط`);
}

if (bad > 0) {
  console.error(`  ✗ ${bad} رصّةً لا يجدها النمط — الخريطةُ ستُرسم بلا أسماء`);
  process.exit(6);
}
console.log(`  ✓ ${wanted.size} رصّةً — الاسمُ مطابقٌ لما يطلبه النمط`);
