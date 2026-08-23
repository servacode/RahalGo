#!/usr/bin/env node
/**
 * ══════════════════════════════════════════════════════════════════
 * **تغطيةُ العربيّة — تُقرأ من داخل `.pbf` لا من وجود الملفّ**
 * ══════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦أ، قرارُ المالك ٢٠٢٦-٠٨-٢١، البند ٥.)
 *
 * **أمرُ المالك نصّاً**: «اختبر أنّ نصّاً عربيّاً حقيقيّاً يجد glyphs.
 * لا أريد أن نكتشف على الجهاز أنّ الأسماء مربّعات».
 *
 * # **ولماذا لا يكفي أن يوجد الملفّ**
 *
 * **`64768-65023.pbf` موجودٌ بمئتي كيلوبايت** ولا يعني أنّ فيه الشكلَ
 * الذي يحتاجه اسمُ «شارع الجمهوريّة». **الملفُّ نطاقٌ، والحاجةُ حرف.**
 *
 * **فيُفتح الملفُّ وتُقرأ مُعرِّفاتُ حروفه** — والمُعرِّفُ هو نقطةُ
 * الترميز نفسُها.
 *
 * # **وأشكالُ العرض تُشتقّ ولا تُفترض**
 *
 * **MapLibre يستدعي `u_shapeArabic_61`** (ثبت بمسح `libmaplibre.so` في
 * PREFLIGHT) **فيحوّل «ش» إلى شكلها بحسب موضعها** ثمّ يبحث عن ذلك
 * الشكل. **فالتغطيةُ تُقاس على الأشكال لا على الحرف الأساس.**
 *
 * **والأشكالُ تُستخرج من Unicode نفسِه**: كلُّ محرفٍ في كتل أشكال
 * العرض **تفكيكُه `NFKD` يردّه إلى أصله.** فالانعكاسُ يعطي أشكالَ
 * كلِّ حرف **بلا جدولٍ مكتوبٍ باليد يشيخ.**
 *
 * **والمسحُ على كتل العربيّة وحدَها** — `U+FE00–U+FE6F` علاماتُ
 * توافقٍ صينيّة، **وتفكيكُ أربعةٍ منها يعطي مسافة**، فكان كلُّ اسمٍ
 * فيه فراغٌ يطلبها.
 *
 *   الاستعمال:  node check-glyph-coverage.mjs <مجلّد الموارد>
 */
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join } from 'node:path';
import { requiredRanges, rangeIndex, SHAPING_BLOCKS } from './glyph-ranges.mjs';
import { ARABIC_CORPUS } from './corpus.mjs';

const outDir = process.argv[2];
if (!outDir) {
  console.error('الاستعمال: node check-glyph-coverage.mjs <مجلّد الموارد>');
  process.exit(2);
}

// ══════════════════════════════════════════════════════════════════
// **قارئُ `.pbf` — أدنى ما يلزم**
// ══════════════════════════════════════════════════════════════════
//
//   glyphs { fontstack = 1 { name = 1, range = 2, glyphs = 3 { id = 1 } } }
//
// **ولا مكتبةَ protobuf** — الحاجةُ حقلٌ واحدٌ من ثلاثةِ مستويات،
// **وتبعيّةٌ جديدةٌ لأجله وزنٌ لا يُبرَّر.**

function readVarint(buf, pos) {
  let result = 0;
  let shift = 0;
  for (;;) {
    const b = buf[pos];
    pos += 1;
    result += (b & 0x7f) * 2 ** shift;
    if ((b & 0x80) === 0) break;
    shift += 7;
  }
  return [result, pos];
}

/** **يمرّ على حقول رسالةٍ ويسلّم كلَّ حقلٍ لمن يعرفه.** */
function eachField(buf, start, end, fn) {
  let pos = start;
  while (pos < end) {
    let key;
    [key, pos] = readVarint(buf, pos);
    const field = key >> 3;
    const wire = key & 7;
    if (wire === 2) {
      let len;
      [len, pos] = readVarint(buf, pos);
      fn(field, wire, pos, pos + len);
      pos += len;
    } else if (wire === 0) {
      let val;
      const at = pos;
      [val, pos] = readVarint(buf, pos);
      fn(field, wire, at, pos, val);
    } else if (wire === 5) {
      fn(field, wire, pos, pos + 4);
      pos += 4;
    } else if (wire === 1) {
      fn(field, wire, pos, pos + 8);
      pos += 8;
    } else {
      throw new Error(`نوعُ سلكٍ غيرُ متوقَّع: ${wire}`);
    }
  }
}

/** **مُعرِّفاتُ الحروف في ملفّ نطاق** — والمُعرِّفُ نقطةُ الترميز. */
function glyphIds(file) {
  const buf = readFileSync(file);
  const ids = new Set();
  eachField(buf, 0, buf.length, (f1, _w1, s1, e1) => {
    if (f1 !== 1) return; // fontstack
    eachField(buf, s1, e1, (f2, _w2, s2, e2) => {
      if (f2 !== 3) return; // glyphs
      eachField(buf, s2, e2, (f3, w3, _s3, _e3, val) => {
        if (f3 === 1 && w3 === 0) ids.add(val); // id
      });
    });
  });
  return ids;
}

// ══════════════════════════════════════════════════════════════════
// **أشكالُ العرض — مشتقّةٌ بالانعكاس**
// ══════════════════════════════════════════════════════════════════
//
// **يُمرّ على `U+FB50–U+FEFF` مرّةً**، ويُفكَّك كلُّ محرفٍ إلى أصله،
// **فيُبنى: حرفٌ أساسٌ ← أشكالُه.**
//
// **والتفكيكُ قد يعطي حرفين** (لام-ألف مثلاً) — فيُنسب الشكلُ إلى
// كليهما، **لأنّ ظهورَه يحتاج وجودَهما معاً في النصّ.**
function buildForms() {
  const forms = new Map();
  for (const block of SHAPING_BLOCKS) {
    for (let cp = block.from; cp <= block.to; cp += 1) {
      const ch = String.fromCodePoint(cp);
      const base = ch.normalize('NFKD');
      if (base === ch) continue;
      for (const b of base) {
        const key = b.codePointAt(0);
        // **ولا يُنسب شكلٌ إلى غيرِ حرفٍ عربيّ.**
        //
        // **تفكيكُ بعض المحارف يعطي مسافةً وعلامةً تركيبيّة**، فلو
        // نُسب الشكلُ إلى المسافة **لطالب كلُّ نصٍّ فيه فراغٌ بأشكالٍ
        // لا علاقةَ له بها.** (وقع فعلاً ٢٠٢٦-٠٨-٢١: المسافةُ جرّت
        // `U+FE49`–`U+FE4C` وهي علاماتُ توافقٍ صينيّة.)
        if (key < 0x0600 || key > 0x06ff) continue;
        if (!forms.has(key)) forms.set(key, new Set());
        forms.get(key).add(cp);
      }
    }
  }
  return forms;
}
const FORMS = buildForms();

// ══════════════════════════════════════════════════════════════════
// **القياس**
// ══════════════════════════════════════════════════════════════════
const glyphDir = join(outDir, 'glyphs');
const stacks = readdirSync(glyphDir).filter((e) =>
  statSync(join(glyphDir, e)).isDirectory(),
);
if (stacks.length === 0) {
  console.error('  ✗ لا رصّةَ حروفٍ في الناتج');
  process.exit(4);
}

/** **كلُّ نقاط الترميز التي يحتاجها المتنُ** — أصولاً وأشكالاً. */
function neededCodepoints(texts) {
  const need = new Set();
  for (const t of texts) {
    for (const ch of t) {
      const cp = ch.codePointAt(0);
      need.add(cp);
      for (const form of FORMS.get(cp) ?? []) need.add(form);
    }
  }
  return need;
}

const needed = [...neededCodepoints(ARABIC_CORPUS)].sort((a, b) => a - b);
const hex = (cp) => `U+${cp.toString(16).toUpperCase().padStart(4, '0')}`;

let failed = 0;

for (const stack of stacks.sort()) {
  const ids = new Set();
  for (const range of requiredRanges()) {
    const file = join(glyphDir, stack, `${range}.pbf`);
    try {
      for (const id of glyphIds(file)) ids.add(id);
    } catch (err) {
      console.error(`  ✗ ${stack}/${range}.pbf غيرُ مقروء: ${err.message}`);
      failed += 1;
    }
  }

  const missing = needed.filter((cp) => !ids.has(cp));

  /**
   * **والغائبُ خارجَ النطاقات المطلوبة عطبُ نطاقٍ لا عطبُ خطّ.**
   *
   * **يُفصلان** — فالأوّلُ يُصلَح في `glyph-ranges.mjs` والثاني يعني
   * أنّ الخطَّ نفسَه لا يحوي الشكل. **والخلطُ بينهما يرسل المصلحَ إلى
   * الملفّ الخطأ.**
   */
  const inRange = new Set(requiredRanges());
  const outsideRanges = missing.filter((cp) => !inRange.has(rangeIndexName(cp)));
  const insideRanges = missing.filter((cp) => inRange.has(rangeIndexName(cp)));

  console.log(
    `  · ${stack}: ${ids.size} حرفاً مبنيّاً — يحتاج المتنُ ${needed.length}`,
  );

  if (outsideRanges.length > 0) {
    console.error(
      `  ✗ ${stack}: ${outsideRanges.length} نقطةً خارجَ النطاقات المبنيّة — عطبُ نطاق`,
    );
    console.error(`    ${outsideRanges.slice(0, 12).map(hex).join(' ')}`);
    failed += 1;
  }
  if (insideRanges.length > 0) {
    console.error(
      `  ✗ ${stack}: ${insideRanges.length} نقطةً داخلَ النطاقات ولا حرفَ لها — عطبُ خطّ`,
    );
    console.error(`    ${insideRanges.slice(0, 12).map(hex).join(' ')}`);
    failed += 1;
  }
  if (missing.length === 0) console.log(`  ✓ ${stack} — لا مربّعَ في المتن`);
}

function rangeIndexName(cp) {
  const i = rangeIndex(cp);
  return `${i * 256}-${i * 256 + 255}`;
}

if (failed > 0) {
  console.error(`  ✗ التغطيةُ ناقصة — الأسماءُ ستظهر مربّعاتٍ على الجهاز`);
  process.exit(7);
}
console.log(
  `  ✓ التغطية — ${stacks.length} رصّةً × ${needed.length} نقطةَ ترميزٍ من ${ARABIC_CORPUS.length} اسماً مقيساً`,
);
