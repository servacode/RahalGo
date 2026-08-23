#!/usr/bin/env node
/**
 * ══════════════════════════════════════════════════════════════════════
 * **نطاقاتُ الحروف — تُشتَقّ ولا تُكتب بيد**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦أ، إغلاقُ أدوات الموارد ٢٠٢٦-٠٨-٢١.)
 *
 * # وخطأٌ وقع لأنّها كُتبت بيد
 *
 * **كانت القائمةُ مكتوبةً حرفاً حرفاً في نسختي الأولى، فوقع فيها
 * خطآن كشفهما المالك:**
 *
 *	٦٤٧٦٨–٦٥٠٢٣  (U+FD00–FDFF)  ←  **سقطت**، وهي من أشكال العرض أ
 *	٦٥٢٨٠–٦٥٥٣٥  (U+FF00–FFFF)  ←  **زائدة**، وهي أشكالٌ عريضةٌ
 *	                                لاتينيّةٌ ويابانيّة، لا عربيّةَ فيها
 *
 * **وثلاثُ قوائمَ متناقضةٍ كانت في المشروع**: ستٌّ هنا، وثلاثٌ يشترطها
 * الفاحص، وخمسٌ في رفيدة الاختبار. **وذاك بعينه ما يمنعه مصدرُ حقيقةٍ
 * واحد.**
 *
 * # فصارت تُحسَب من حدود Unicode المسمّاة
 *
 * **ولا رقمَ سحريٌّ يُدقَّق بالعين** — الكتلُ تُعلَن بأسمائها وحدودها،
 * **والنطاقاتُ تُشتَقّ منها بالقسمة على ٢٥٦.**
 *
 * # ولماذا أشكالُ العرض أصلاً
 *
 * **مسحُ `libmaplibre.so` في PREFLIGHT وجد `u_shapeArabic_61`** —
 * يحوّل الحرفَ إلى شكل عرضه بحسب موضعه:
 *
 *	ش (U+0634) → ﺷ (U+FEB7) في الوسط
 *	ا (U+0627) → ﺎ (U+FE8E) في الآخر
 *
 * **ثمّ يُبحث عن ذلك الشكل في نطاقات الحروف** — فمجموعةٌ فيها العربيّةُ
 * الأساسيّةُ وحدَها **تُظهر النصَّ مربّعاتٍ والخطُّ سليم.**
 *
 *   الاستعمال:
 *     node glyph-ranges.mjs           ← يطبع النطاقات وسببَ كلٍّ منها
 *     node glyph-ranges.mjs --json    ← للفاحص والبناء
 */

/** **حجمُ النطاق كما يقسّمه MapLibre** — `{range}.pbf`. */
export const RANGE_SIZE = 256;

/**
 * ══════════════════════════════════════════════════════════════════════
 * **كتلُ Unicode اللازمة — بأسمائها وحدودها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وهذه وحدَها ما يُدقَّق بالعين** — وما بعدها حساب.
 */
export const BLOCKS = [
  {
    name: 'Basic Latin + Latin-1',
    from: 0x0000,
    to: 0x00ff,
    why: 'الأرقامُ والمسافةُ وعلاماتُ الترقيم — «طريق 4» و«الرقة 12»',
  },
  {
    name: 'Arabic',
    from: 0x0600,
    to: 0x06ff,
    why: 'الحروفُ الأساسيّةُ والتشكيلُ والأرقامُ العربيّة-الهنديّة',
  },
  {
    name: 'Arabic Presentation Forms-A',
    from: 0xfb50,
    to: 0xfdff,
    why: 'ما يولّده u_shapeArabic من مركّبات — لام-ألف والتراكيب',
  },
  {
    name: 'Arabic Presentation Forms-B',
    from: 0xfe70,
    to: 0xfeff,
    why: 'أشكالُ الحرف الأربعة: مفردٌ وأوّلٌ ووسطٌ وآخر — وهي الأكثر',
  },
];

/**
 * **ما ينتجه التشكيلُ من كتل** — تُستعمل في اشتقاق النطاقات وفي الشرح.
 */
export const SHAPING_BLOCKS = BLOCKS.filter((b) => b.name.includes('Presentation'));

export const rangeIndex = (cp) => Math.floor(cp / RANGE_SIZE);
export const rangeName = (i) => `${i * RANGE_SIZE}-${i * RANGE_SIZE + RANGE_SIZE - 1}`;

/**
 * **النطاقاتُ اللازمة — مشتقّةٌ من الكتل.**
 *
 * **وكتلةٌ تعبر حدَّ ٢٥٦ تُنتج أكثرَ من نطاق**: `FB50–FDFF` تقع في
 * ثلاثةٍ (`FB00` · `FC00` · `FD00`) — **وهذا بعينه ما سقط في نسختي
 * الأولى.**
 */
export function requiredRanges(extraCodepoints = []) {
  const need = new Set();
  for (const b of BLOCKS) {
    for (let i = rangeIndex(b.from); i <= rangeIndex(b.to); i++) need.add(i);
  }
  for (const cp of extraCodepoints) need.add(rangeIndex(cp));
  return [...need].sort((a, b) => a - b).map(rangeName);
}

/** **ولماذا كلُّ نطاق** — للتقرير والتدقيق. */
export function explain() {
  const rows = [];
  for (const b of BLOCKS) {
    for (let i = rangeIndex(b.from); i <= rangeIndex(b.to); i++) {
      rows.push({
        range: rangeName(i),
        block: b.name,
        codepoints: `U+${(i * RANGE_SIZE).toString(16).toUpperCase().padStart(4, '0')}–` +
          `U+${(i * RANGE_SIZE + 255).toString(16).toUpperCase().padStart(4, '0')}`,
        why: b.why,
      });
    }
  }
  return rows;
}

/**
 * **النطاقاتُ التي تحتاجها مُدوَّنةٌ فعليّاً** — قبل التشكيل.
 *
 * **وتُقارَن بالمشتقّة**: إن طلبت المُدوَّنةُ نطاقاً خارجَها **فثمّة
 * كتلةٌ ناقصةٌ في `BLOCKS`** — والفاحصُ يُسقط البناء.
 */
export function rangesForText(texts) {
  const need = new Set();
  for (const t of texts) {
    for (const ch of t) need.add(rangeIndex(ch.codePointAt(0)));
  }
  return [...need].sort((a, b) => a - b).map(rangeName);
}

if (process.argv[1] && process.argv[1].endsWith('glyph-ranges.mjs')) {
  if (process.argv.includes('--list')) {
    // **سطرٌ لكلّ نطاق** — يقرؤه `build-glyphs.sh`، فلا تُدوَّن
    // النطاقاتُ في مكانين ثمّ تتناقضان.
    process.stdout.write(`${requiredRanges().join('\n')}\n`);
  } else if (process.argv.includes('--json')) {
    process.stdout.write(`${JSON.stringify({ ranges: requiredRanges(), blocks: BLOCKS }, null, 2)}\n`);
  } else {
    console.log('نطاقاتُ الحروف — مشتقّةٌ من كتل Unicode:\n');
    for (const r of explain()) {
      console.log(`  ${r.range.padEnd(14)} ${r.codepoints.padEnd(16)} ${r.block}`);
      console.log(`  ${''.padEnd(14)} ${''.padEnd(16)} ${r.why}`);
    }
    console.log(`\nالمجموع: ${requiredRanges().length} نطاقاً`);
  }
}
