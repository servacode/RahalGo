#!/usr/bin/env node
/**
 * ══════════════════════════════════════════════════════════════════════
 * **نمطُ الخريطة مصدرٌ واحد — ولا نسختان تنحرفان**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦أ، قرارُ المالك ٢٠٢٦-٠٨-٢١، البند ٨.)
 *
 * # وما كان قبله
 *
 * **ثلاثُ نسخٍ من مصدر البلاطات في المشروع:**
 *
 *	internal/server/map_style.go          ←  نصٌّ مكتوبٌ في Go
 *	app-driver/src/main/assets/…json      ←  نسخةٌ ثانية
 *	web/packages/ui/src/map.tsx           ←  قائمةُ مصادرَ ثالثة
 *
 * **وكانت الأولى والثانيةُ متطابقتين صدفةً** — ولا شيءَ يمنع
 * انحرافَهما. **ووثيقتان تتناقضان صامتتين أسوأُ من لا وثيقة**
 * (`CLAUDE.md`).
 *
 * # وما يفحصه هذا الحارس
 *
 *	١ · نسخةُ الأصول تطابق المرجعيَّ في `maps/style`
 *	٢ · المحرّكُ يُضمّن المرجعيَّ ولا يكتبه نصّاً
 *	٣ · النمطُ المتّجهُ مراسيه معرَّفة
 *	٤ · الويبُ يقرأ مصدرَه من الإعداد لا من ثابتٍ وحدَه
 *	٥ · نسخةُ أندرويد من النمط المتّجه تطابق المرجعيَّ بايتاً ببايت
 *	٦ · نطاقاتُ الحروف في كوتلن هي نفسُها في `glyph-ranges.mjs`
 *	٧ · **لا مسارَ إنتاجٍ في أندرويد يبلغ راستر OSM العموميّ**
 *
 * # وفصلُ الحالتين — المرحلة ٦ب، البند ٥٠
 *
 * **أندرويد وحدَه أُغلق.** والويبُ يبقى `TD-WEB-RASTER-SOURCE`
 * **حاجزَ إصدار** حتّى يُضبط مصدرُ بلاطاته عند النشر. **ولا يُخلط
 * الحالان**: «Android production: zero tile.openstreetmap.org» بينما
 * «Web production fallback: still exists».
 */

import { readFileSync, existsSync, readdirSync, statSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { requiredRanges } from '../../maps/scripts/glyph-ranges.mjs';

const here = dirname(fileURLToPath(import.meta.url));
const ROOT = join(here, '..', '..');

const problems = [];
const notes = [];

function read(p) {
  return existsSync(p) ? readFileSync(p, 'utf8') : null;
}

/**
 * **كلُّ مصدرٍ يدخل حزمةَ الإصدار** — ولا اختبارَ ولا ناتجَ بناء.
 *
 * **و`src/test` و`src/androidTest` تُستثنيان** — أمرُ المالك: يجوز
 * الاحتفاظُ بالراستر «للاختبارات/التاريخ»، **ولا يكون Production
 * fallback.**
 */
function walkAndroid(dir) {
  const out = [];
  for (const name of readdirSync(dir)) {
    if (name === 'build' || name === '.gradle' || name === 'test' ||
        name === 'androidTest' || name === 'debug') {
      continue;
    }
    const p = join(dir, name);
    if (statSync(p).isDirectory()) out.push(...walkAndroid(p));
    else if (/\.(kt|kts|json|xml)$/.test(name)) out.push(p);
  }
  return out;
}

// ── ١ · الراستر بقي مرجعاً ولم يبقَ في التطبيق ─────────────────────
//
// (المرحلة ٦ب، البند ٢.)
//
// **`legacy-raster.style.json` يبقى** — يُضمّنه المحرّكُ لواجهةٍ
// عامّةٍ يستعملها الويب، **وهو تاريخُ ما كان.**
//
// **ونسخةُ أندرويد أُزيلت** — أمرُ المالك: «لا يكون Production
// fallback». **فوجودُها في `assets/` وحدَه احتمالُ عودةٍ**: يكفي أن
// يكتب أحدٌ `asset://map-style.json` فيعود الراستر بلا أن يُلاحَظ.
const canonicalRaw = read(join(ROOT, 'maps', 'style', 'legacy-raster.style.json'));
if (!canonicalRaw) problems.push('المرجعيُّ غائب: maps/style/legacy-raster.style.json');

const androidRasterAsset = join(
  ROOT, 'mobile', 'app-driver', 'src', 'main', 'assets', 'map-style.json',
);
if (existsSync(androidRasterAsset)) {
  problems.push('نمطُ الراستر عاد إلى أصول أندرويد — والمسارُ المتّجهُ هو الإنتاج');
} else if (canonicalRaw) {
  notes.push(
    `النمطُ الراستر: مرجعٌ في maps/style · ${JSON.parse(canonicalRaw).layers.length} طبقة · ` +
    'لا نسخةَ في أندرويد',
  );
}

// ── ٢ · المحرّكُ يُضمّن ولا يكتب ────────────────────────────────────
const goSrc = read(join(ROOT, 'backend', 'internal', 'server', 'map_style.go'));
if (!goSrc) {
  problems.push('map_style.go غائب');
} else {
  if (!goSrc.includes('//go:embed legacy-raster.style.json')) {
    problems.push('map_style.go لا يُضمّن المرجعيَّ — عاد يكتب النمطَ نصّاً');
  }
  if (/"version":\s*8/.test(goSrc)) {
    problems.push('map_style.go يحوي نمطاً مكتوباً نصّاً — والمصدرُ واحد');
  }
  if (!goSrc.includes('MAP_TILE_URL')) {
    problems.push('map_style.go لا يقرأ MAP_TILE_URL — ومصدرُ البلاطات يجب أن يُضبط بالنشر');
  }
}

// ── ٣ · النمطُ المتّجهُ ومراسيه ─────────────────────────────────────
const vectorRaw = read(join(ROOT, 'maps', 'style', 'rahalgo.style.json'));
if (!vectorRaw) {
  problems.push('النمطُ المتّجهُ غائب: maps/style/rahalgo.style.json');
} else {
  const v = JSON.parse(vectorRaw);
  const ids = new Set(v.layers.map((l) => l.id));
  for (const [role, anchor] of Object.entries(v._anchors ?? {})) {
    if (!ids.has(anchor)) problems.push(`المِرساةُ ${role} تشير إلى طبقةٍ غيرِ موجودة: ${anchor}`);
  }
  // **ولا اسمَ عربيٌّ يُقرأ بلا `name:ar`** — سياسةُ التسمية.
  const labels = v.layers.filter((l) => l.type === 'symbol');
  for (const l of labels) {
    const field = JSON.stringify(l.layout?.['text-field'] ?? '');
    if (!field.includes('name:ar')) {
      problems.push(`طبقةُ الأسماء ${l.id} لا تقرأ name:ar`);
    }
  }
  notes.push(`النمطُ المتّجه: ${v.layers.length} طبقة · ${labels.length} طبقةَ أسماء · ` +
    `${Object.keys(v._anchors ?? {}).length} مراسٍ`);
}

// ── ٤ · والويبُ على النمط المرجعيّ نفسِه ────────────────────────────
//
// **كان راستراً بسلسلةِ ارتدادٍ عموميّة** (`TD-WEB-RASTER-SOURCE`)،
// **وصار متّجهاً يقرأ النمطَ المرجعيَّ من مضيفنا** — ٢٠٢٦-٠٨-٢١.
const webMap = read(join(ROOT, 'web', 'packages', 'ui', 'src', 'map.tsx'));
const webCfg = read(join(ROOT, 'web', 'packages', 'ui', 'src', 'mapconfig.ts'));
if (!webMap) {
  problems.push('web/packages/ui/src/map.tsx غائب');
} else if (!webCfg) {
  problems.push('web/packages/ui/src/mapconfig.ts غائب — لا مصدرَ مركزيّ');
} else if (!webCfg.includes('NEXT_PUBLIC_MAP_STYLE_URL')) {
  problems.push('الويبُ لا يقرأ NEXT_PUBLIC_MAP_STYLE_URL — ولا سبيلَ لضبط المصدر بالنشر');
} else if (!webMap.includes('maplibre-gl')) {
  problems.push('الويبُ ليس على المحرّك المتّجه — والنمطُ المرجعيُّ لا يُقرأ براستر');
} else {
  notes.push('الويب: النمطُ المرجعيُّ يُضبط بـNEXT_PUBLIC_MAP_STYLE_URL — ولا ارتدادَ عموميّ');
}

// ── ٥ · نسخةُ أندرويد من النمط المتّجه ──────────────────────────────
//
// **النمطُ يصل التطبيقَ مورداً خامّاً** (`res/raw`) — فوحدةُ المكتبة
// تدمج مواردَها في التطبيقات الثلاثة، **و`assets` تصطدم إن سمّاها
// تطبيقٌ بالاسم نفسِه.**
//
// **والمطابقةُ بايتاً ببايت** — لا دلاليّاً. **فأيُّ فرقٍ انحرافٌ**،
// ولا سببَ يبرّره: الاختلافُ الوحيدُ المشروع (العنوان) **يُحقَن في
// زمن التشغيل** لا في الملفّ.
const androidStyle = read(
  join(ROOT, 'mobile', 'map', 'src', 'main', 'res', 'raw', 'rahalgo_style.json'),
);
if (!androidStyle) {
  problems.push('نسخةُ أندرويد غائبة: mobile/map/…/res/raw/rahalgo_style.json');
} else if (vectorRaw && androidStyle !== vectorRaw) {
  problems.push('نسخةُ أندرويد من النمط المتّجه انحرفت عن المرجعيّ');
} else if (vectorRaw) {
  notes.push('النمطُ المتّجه: نسخةُ أندرويد مطابقةٌ بايتاً ببايت');
}

// ── ٦ · نطاقاتُ الحروف في لغتين ─────────────────────────────────────
//
// **كوتلن لا تستورد من JavaScript** — فالقائمةُ مكرَّرةٌ ضرورةً.
// **والتكرارُ بلا حارسٍ هو ما أنتج ثلاثَ قوائمَ متناقضةٍ في ٦أ.**
const kotlinStore = read(
  join(ROOT, 'mobile', 'map', 'src', 'main', 'kotlin', 'com', 'rahalgo', 'map',
    'data', 'MapPackageStore.kt'),
);
if (!kotlinStore) {
  problems.push('MapPackageStore.kt غائب');
} else {
  const block = /object MapGlyphRanges \{[\s\S]*?\n\}/.exec(kotlinStore);
  if (!block) {
    problems.push('MapGlyphRanges غيرُ موجودٍ في MapPackageStore.kt');
  } else {
    const inKotlin = [...block[0].matchAll(/"(\d+-\d+)"/g)].map((m) => m[1]);
    const inSource = requiredRanges();
    if (JSON.stringify(inKotlin) !== JSON.stringify(inSource)) {
      problems.push(
        `نطاقاتُ الحروف في كوتلن تخالف glyph-ranges.mjs — ` +
        `كوتلن ${inKotlin.join(' ')} · المصدر ${inSource.join(' ')}`,
      );
    } else {
      notes.push(`نطاقاتُ الحروف: ${inSource.length} في كوتلن وفي المصدر — متطابقة`);
    }
  }
}

// ── ٧ · لا راستر OSM في مسار إنتاج أندرويد ──────────────────────────
//
// (البند ٢: «لا أريد أي Android Production path يصل إلى
//  tile.openstreetmap.org».)
//
// **ويُبحث في كلّ ما يُبنى في الإصدار** — لا في `test/` ولا في
// تعليق. **فالتعليقُ يشرح ولا يتّصل.**
const androidProd = walkAndroid(join(ROOT, 'mobile'));
const osmHits = [];
for (const f of androidProd) {
  const raw = readFileSync(f, 'utf8');
  const code = raw
    .replace(/\/\*[\s\S]*?\*\//g, '')
    .replace(/^\s*\/\/.*$/gm, '');
  if (/tile\.openstreetmap\.org|[ab-c]\.tile\.openstreetmap/.test(code)) {
    osmHits.push(f.slice(ROOT.length + 1).replace(/\\/g, '/'));
  }
}
if (osmHits.length) {
  problems.push('مسارُ إنتاجٍ في أندرويد يبلغ راستر OSM العموميّ:');
  for (const h of osmHits) problems.push(`    ${h}`);
} else {
  notes.push(
    `أندرويد الإنتاج: صفرُ نداءٍ إلى tile.openstreetmap.org · ` +
    `${androidProd.length} ملفّاً فُحص`,
  );
  // **والويبُ حالٌ أخرى تُذكر ولا تُخلط** (البند ٥٠).
  notes.push('الويب: TD-WEB-RASTER-SOURCE باقٍ حاجزَ إصدارٍ حتّى يُضبط مصدرُ النشر');
}

// ── ٨ · ألوانُ خطوط المسار مركزيّة ─────────────────────────────────
//
// (إغلاقُ واجهة ٧، قرارُ المالك ٢٠٢٦-٠٨-٢١، البند ٤٨.)
//
// **أمرُ المالك نصّاً**: «لا تنشر Hex values جديدة في عدة ملفات…
// اجعل Route visualization palette مركزية».
//
// **وكانت متفرّقةً**: `#02678F` و`#1E88E5` في `TripMarkers`،
// و`#8A94A6` في `AltRouteLayer`. **فمن بدّل لوناً بدّل نصفَه.**
const NAV_DIR = join(ROOT, 'mobile', 'driver-navigation', 'src', 'main',
  'kotlin', 'com', 'rahalgo', 'navigation');
const PALETTE = join(NAV_DIR, 'MapRoutePalette.kt');

if (!existsSync(PALETTE)) {
  problems.push('MapRoutePalette.kt غائب — وألوانُ المسار تتفرّق');
} else {
  const palette = readFileSync(PALETTE, 'utf8');
  const declared = new Set(
    [...palette.matchAll(/"(#[0-9A-Fa-f]{6,8})"/g)].map((m) => m[1].toUpperCase()),
  );

  // **وكلُّ لونٍ في ملفّات رسم المسار يجب أن يكون منها.**
  const drawFiles = ['AltRouteLayer.kt', 'TripMarkers.kt', 'TripMap.kt'];
  const stray = [];
  for (const name of drawFiles) {
    const p = join(NAV_DIR, name);
    if (!existsSync(p)) continue;
    const src = readFileSync(p, 'utf8')
      .replace(/\/\*[\s\S]*?\*\//g, '')
      .replace(/^\s*\/\/.*$/gm, '');
    for (const m of src.matchAll(/"(#[0-9A-Fa-f]{6,8})"/g)) {
      const hex = m[1].toUpperCase();
      if (!declared.has(hex)) stray.push(`${name} → ${hex}`);
    }
  }
  if (stray.length) {
    problems.push('ألوانٌ خارجَ MapRoutePalette:');
    for (const x of stray) problems.push(`    ${x}`);
  } else {
    notes.push(`ألوانُ المسار: ${declared.size} في MapRoutePalette · ولا لونَ شارد`);
  }
}

for (const n of notes) console.log(`  · ${n}`);
if (problems.length) {
  console.error('نمطُ الخريطة — خللٌ:');
  for (const p of problems) console.error(`  ✗ ${p}`);
  process.exit(1);
}
console.log('نمطُ الخريطة مصدرٌ واحد — ولا نسخةَ تنحرف.');
