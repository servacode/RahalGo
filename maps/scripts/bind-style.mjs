#!/usr/bin/env node
/**
 * ══════════════════════════════════════════════════════════════════════
 * **نمطٌ دلاليٌّ واحد — ومصادرُ تتبدّل**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦أ، قرارُ المالك ٢٠٢٦-٠٨-٢١، البند ٨.)
 *
 *     rahalgo.style.json  ←  الطبقاتُ والألوانُ والعقود
 *            ↓  bind
 *     online   ·   offline   ·   fallback
 *
 * # ولماذا لا تُقارَن النصوصُ
 *
 * **العناوينُ تختلف بالضرورة**: الأونلاين `pmtiles://https://…`
 * والمحلّيُّ `pmtiles://file://…`. **فمقارنةُ JSON حرفاً بحرفٍ تسقط
 * أبداً** — والمقارنةُ الصحيحةُ **دلاليّة**: الطبقاتُ والرسمُ والتخطيطُ
 * وعقودُ `source-layer` هي هي، **والمصادرُ وحدَها تتبدّل.**
 *
 * # وما كان قبل هذا
 *
 * **نسختان من النمط في المشروع** — واحدةٌ مكتوبةٌ نصّاً في
 * `internal/server/map_style.go` وأخرى في أصول تطبيق السائق. **وكانتا
 * متطابقتين صدفةً، ولا شيءَ يمنع انحرافَهما.**
 *
 *   الاستعمال:
 *     node bind-style.mjs online  --tiles=https://cdn/…/syria.pmtiles \
 *                                 --resources=https://cdn/map-resources/1
 *     node bind-style.mjs offline --tiles=raqqa.pmtiles --resources=.
 */

import { readFileSync, writeFileSync, mkdirSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const CANONICAL = join(here, '..', 'style', 'rahalgo.style.json');

/** **الارتباطاتُ الثلاثة** — ولا رابعَ لها. */
export const BINDINGS = ['online', 'offline', 'fallback'];

/**
 * **يبني نمطاً نافذاً من المرجعيّ.**
 *
 * **و`tiles` يُلفّ بـ`pmtiles://`** — المحرّكُ يفهمه أصلاً منذ 13.4.1
 * (قِيس في PREFLIGHT: `mbgl::PMTilesFileSource` في الثنائيّ).
 */
export function bind(canonical, { binding, tiles, resources }) {
  const style = JSON.parse(JSON.stringify(canonical));
  delete style._note;
  delete style._contract;

  const tileUrl = tiles.startsWith('pmtiles://') || tiles.startsWith('mbtiles://')
    ? tiles
    : `pmtiles://${tiles}`;

  style.sources.base.url = tileUrl;
  style.glyphs = `${resources}/glyphs/{fontstack}/{range}.pbf`;
  style.sprite = `${resources}/sprite`;
  style.metadata = { ...style.metadata, 'rahalgo:binding': binding };
  return style;
}

/**
 * **المقارنةُ الدلاليّة** — تُستعمل في الحارس.
 *
 * **وتُهمل المصادرَ وحدَها** — انظر أعلى الملفّ.
 */
export function semanticShape(style) {
  return {
    version: style.version,
    layers: style.layers.map((l) => ({
      id: l.id,
      type: l.type,
      source: l.source ?? null,
      sourceLayer: l['source-layer'] ?? null,
      minzoom: l.minzoom ?? null,
      maxzoom: l.maxzoom ?? null,
      filter: l.filter ?? null,
      layout: l.layout ?? null,
      paint: l.paint ?? null,
    })),
  };
}

function main() {
  const [binding, ...rest] = process.argv.slice(2);
  if (!BINDINGS.includes(binding)) {
    console.error(`الارتباطُ غيرُ معروف: ${binding} — المتاح: ${BINDINGS.join(' · ')}`);
    process.exit(2);
  }
  const arg = (name, fallback) => {
    const hit = rest.find((a) => a.startsWith(`--${name}=`));
    return hit ? hit.slice(name.length + 3) : fallback;
  };
  const canonical = JSON.parse(readFileSync(CANONICAL, 'utf8'));

  // ══════════════════════════════════════════════════════════════════
  //  **ولا افتراضَ إلى `latest`** — إغلاقُ `TD-MAP-LATEST-PIN`
  // ══════════════════════════════════════════════════════════════════
  //
  // (قرارُ المالك ٢٠٢٦-٠٨-٢٢، البند ٦: «لا أريد Production يعتمد على
  //  base/latest/syria.pmtiles».)
  //
  // **كان الافتراضُ `base/latest/syria.pmtiles`** — ومن نسي الوسيطَ
  // خرج بنمطٍ يشير إلى ملفٍّ **يتبدّل من تحت العميل**: الهاتفُ يخزّن
  // بلاطاتِ نسخةٍ ويطلب ما بقي من نسخةٍ أخرى، **فتختلط طبقتان لا
  // يُعرف أيُّهما.** ولا رجوعَ (`rollback`) إلى ما لا اسمَ له.
  //
  // **والصمتُ هو العطب**: خطأٌ يُرى خيرٌ من نمطٍ يبدو صحيحاً ويشير إلى
  // متحرّك. **فيُرفض الآن ولا يُخمَّن.**
  const tiles = arg('tiles', '');
  if (!tiles) {
    console.error([
      'لزِم --tiles=<عنوانُ أرشيفٍ مؤرَّخ> — مثل',
      '  https://maps.rahalgo.com/base/2026-08-20/syria.pmtiles',
      'ولا يُقبل `latest`: ملفٌّ يتبدّل تحت العميل لا يُرجَع إليه.',
    ].join('\n'));
    process.exit(2);
  }
  if (/\/latest\//.test(tiles)) {
    console.error(`عنوانٌ متحرّكٌ مرفوض: ${tiles}`);
    process.exit(2);
  }

  const style = bind(canonical, {
    binding,
    tiles,
    resources: arg('resources', 'https://maps.rahalgo.com/map-resources/1'),
  });
  const out = arg('out', '');
  const text = `${JSON.stringify(style, null, 2)}\n`;
  if (out) {
    mkdirSync(dirname(out), { recursive: true });
    writeFileSync(out, text);
    console.error(`كُتب: ${out}`);
  } else {
    process.stdout.write(text);
  }
}

if (process.argv[1] && process.argv[1].endsWith('bind-style.mjs')) main();
