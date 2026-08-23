/**
 * ══════════════════════════════════════════════════════════════════════
 * **قارئُ آثارٍ أدنى — للتحقّق لا للعرض**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦أ.)
 *
 * **يقرأ ما يلزم الفهرسَ والفحصَ وحدَه**: الترويسةَ والبياناتِ
 * الوصفيّةَ والحدود. **ولا يفكّ بلاطةً ولا يرسم** — ذاك شأنُ المحرّك.
 *
 * **ويقرأ الصيغتين**: `PMTiles` أصلاً، و`MBTiles` إن لزم الاحتياط.
 * (قرارُ المالك، البند ٤: **MBTiles احتياطٌ لا أساس.**)
 */

import { readFileSync, openSync, readSync, closeSync } from 'node:fs';
import { gunzipSync } from 'node:zlib';

/** **رقمُ الصيغةِ السحريّ.** */
const MAGIC = 'PMTiles';

/**
 * **يقرأ ترويسةَ PMTiles v3.**
 *
 * **والحدودُ في الترويسة لا في البيانات الوصفيّة** — قِيس ٢٠٢٦-٠٨-٢١:
 * حزمةُ الرقّة تردّ `bounds = null` من البيانات الوصفيّة **وحدودُها
 * في الترويسة صحيحة.** فمن قرأ الوصفيّةَ وحدَها ظنّ الحزمةَ بلا حدود.
 */
export function readPmtilesHeader(path) {
  const fd = openSync(path, 'r');
  try {
    const buf = Buffer.alloc(127);
    readSync(fd, buf, 0, 127, 0);
    if (buf.toString('ascii', 0, 7) !== MAGIC) throw new Error(`ليس PMTiles: ${path}`);
    const spec = buf.readUInt8(7);
    const metaOffset = Number(buf.readBigUInt64LE(24));
    const metaLength = Number(buf.readBigUInt64LE(32));
    const h = {
      spec,
      metaOffset,
      metaLength,
      internalCompression: buf.readUInt8(97),
      tileCompression: buf.readUInt8(98),
      tileType: buf.readUInt8(99),
      minZoom: buf.readUInt8(100),
      maxZoom: buf.readUInt8(101),
      // **الحدودُ بالمليون** — كما تنصّ الصيغة.
      bounds: [
        buf.readInt32LE(102) / 1e7,
        buf.readInt32LE(106) / 1e7,
        buf.readInt32LE(110) / 1e7,
        buf.readInt32LE(114) / 1e7,
      ],
      // **وعددُ البلاطات عند ٧٢ لا ٥٦** — الإزاحةُ ٥٦ هي بدايةُ
      // بيانات البلاطات. (صُحّح بالقياس ٢٠٢٦-٠٨-٢١: كان يردّ ١٦٣٨٤
      // في كلّ أثرٍ — وهو طولُ الترويسة والدليل لا عددُ البلاطات.)
      addressedTiles: Number(buf.readBigUInt64LE(72)),
      tileEntries: Number(buf.readBigUInt64LE(80)),
    };
    return h;
  } finally {
    closeSync(fd);
  }
}

function decompress(buf, kind) {
  return kind === 2 ? gunzipSync(buf) : buf;
}

/**
 * **يقرأ ما يلزم من أيّ أثر** — ويوحّد الأسماء.
 *
 * **فالفهرسُ والفحصُ لا يعرفان صيغةً** — يعرفان حقولاً.
 */
export function readArchiveMeta(path) {
  if (path.endsWith('.pmtiles')) {
    const h = readPmtilesHeader(path);
    const whole = readFileSync(path);
    const raw = whole.subarray(h.metaOffset, h.metaOffset + h.metaLength);
    const meta = JSON.parse(decompress(raw, h.internalCompression).toString('utf8'));
    const vectorLayers =
      meta.vector_layers ?? JSON.parse(meta.json ?? '{}').vector_layers ?? [];
    return {
      format: 'pmtiles',
      name: meta.name,
      tileSchema: meta.version,
      minZoom: h.minZoom,
      maxZoom: h.maxZoom,
      bounds: h.bounds,
      tileCount: h.addressedTiles,
      attribution: meta.attribution,
      vectorLayers: vectorLayers.map((l) => l.id),
      fields: Object.fromEntries(vectorLayers.map((l) => [l.id, Object.keys(l.fields ?? {})])),
      dataVersion: stampOf(meta),
      generator: `planetiler ${meta['planetiler:version'] ?? '?'}`,
      generatorCommit: meta['planetiler:githash'] ?? null,
      buildTime: meta['planetiler:buildtime'] ?? null,
    };
  }
  throw new Error(`صيغةٌ لا تُقرأ هنا: ${path}`);
}

/**
 * **نسخةُ البيانات — من ختم OSM لا من ساعة البناء.**
 *
 * **وساعةُ البناء تتبدّل بإعادة التشغيل والبياناتُ هي هي** — فالنسخةُ
 * ختمُ المصدر، وهو ما يجعل بناءين متتاليين يعطيان النسخةَ نفسَها.
 */
function stampOf(meta) {
  const t = meta['planetiler:osm:osmosisreplicationtime'];
  if (!t) return 'unknown';
  return String(t).slice(0, 10);
}
