#!/usr/bin/env node
/**
 * ══════════════════════════════════════════════════════════════════════
 * **الفهرس — عقدٌ مؤرَّخٌ لا جدولُ قاعدة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦أ، قرارُ المالك ٢٠٢٦-٠٨-٢١، البند ١١: «لا Database
 *  migration».)
 *
 * # ولماذا لا جدول
 *
 * **الحزمُ آثارٌ ثابتةٌ يولّدها خطُّ الأنابيب** — لا صفوفٌ تُحرَّر.
 * **وتحمل نسختَها في ترويستها**: قِيس في PREFLIGHT أنّ Planetiler
 * يكتب داخلَ الأرشيف:
 *
 *	planetiler:osm:osmosisreplicationtime = 2026-08-20T20:20:51Z
 *	planetiler:version · planetiler:githash · planetiler:buildtime
 *
 * **فجدولٌ في القاعدة يكرّر ما في الملفّ** — ويفترق عنه يوماً.
 *
 * # ويُكتب آخرَ شيء
 *
 * **ولا يشير إلى أثرٍ لم يُرفَع** (البند ١٠) — فالنشرُ: ارفع الثابت،
 * **ثمّ انشر الفهرس.**
 */

import { readFileSync, readdirSync, statSync } from 'node:fs';
import { createHash } from 'node:crypto';
import { join } from 'node:path';
import { readArchiveMeta } from './pmtiles.mjs';

/** **نسخةُ عقد الفهرس** — تُرفع إن تبدّل شكلُه. */
export const MANIFEST_SCHEMA_VERSION = 1;

/**
 * **أسماءُ المناطق من وصفة البناء.**
 *
 * **ولا تُقرأ من الأثر** — Planetiler يكتب فيه اسمَ المخطّط.
 */
function regionNames() {
  const env = readFileSync(join(import.meta.dirname, '..', 'config', 'build.env'), 'utf8');
  const out = {};
  const RE = /^REGION_([A-Z_]+)="([^|]+)\|/;
  for (const line of env.split(String.fromCharCode(10))) {
    const m = RE.exec(line.trim());
    if (m) out[m[1].toLowerCase()] = m[2];
  }
  return out;
}

function sha256(path) {
  return createHash('sha256').update(readFileSync(path)).digest('hex');
}

/**
 * **يبني الفهرسَ من مجلّد الآثار.**
 *
 * **ولا يخترع رقماً** — كلُّ حقلٍ مقروءٌ من الأثر نفسِه أو من القرص.
 */
export function buildManifest(outDir) {
  const files = readdirSync(outDir);
  const base = files.find((f) => f === 'syria.pmtiles');
  if (!base) throw new Error('لا أرشيفَ أساسيّ: syria.pmtiles');

  const baseMeta = readArchiveMeta(join(outDir, base));
  const regions = files
    .filter((f) => f.startsWith('region-') && f.endsWith('.pmtiles'))
    .map((f) => {
      const id = f.slice('region-'.length, -'.pmtiles'.length);
      const m = readArchiveMeta(join(outDir, f));
      return {
        id,
        // **والاسمُ من الوصفة لا من الأثر** — Planetiler يكتب اسمَ
        // المخطّط (`OpenMapTiles`) لا اسمَ المنطقة.
        name: regionNames()[id] ?? id,
        bbox: m.bounds,
        url: `regions/${m.dataVersion}/${f}`,
        bytes: statSync(join(outDir, f)).size,
        sha256: sha256(join(outDir, f)),
        minZoom: m.minZoom,
        maxZoom: m.maxZoom,
        dataVersion: m.dataVersion,
      };
    });

  const resources = JSON.parse(
    readFileSync(join(outDir, 'map-resources', 'resources.json'), 'utf8'),
  );

  return {
    schemaVersion: MANIFEST_SCHEMA_VERSION,
    dataVersion: baseMeta.dataVersion,
    tileSchema: baseMeta.tileSchema,
    generator: baseMeta.generator,
    generatorCommit: baseMeta.generatorCommit,
    buildTime: baseMeta.buildTime,
    resourcesVersion: resources.resourcesVersion,
    base: {
      url: `base/${baseMeta.dataVersion}/syria.pmtiles`,
      bytes: statSync(join(outDir, base)).size,
      sha256: sha256(join(outDir, base)),
      minZoom: baseMeta.minZoom,
      maxZoom: baseMeta.maxZoom,
      bbox: baseMeta.bounds,
    },
    regions,
    resources: {
      url: `map-resources/${resources.resourcesVersion}/`,
      styleVersion: resources.styleVersion,
      glyphVersion: resources.glyphVersion,
      spriteVersion: resources.spriteVersion,
      fontstacks: resources.fontstacks,
    },
    // **والإسنادُ عقدٌ مركزيٌّ لا نصوصٌ متفرّقة** (البند ٢٥).
    attribution: baseMeta.attribution,
  };
}

if (process.argv[1] && process.argv[1].endsWith('manifest.mjs')) {
  const dir = process.argv[2];
  if (!dir) {
    console.error('الاستعمال: node manifest.mjs <مجلّد الآثار>');
    process.exit(2);
  }
  process.stdout.write(`${JSON.stringify(buildManifest(dir), null, 2)}\n`);
}
