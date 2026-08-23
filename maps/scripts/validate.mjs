#!/usr/bin/env node
/**
 * ══════════════════════════════════════════════════════════════════════
 * **فحصُ الآثار — ولا يُنشَر ما لم يُفحص**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦أ، قرارُ المالك ٢٠٢٦-٠٨-٢١، البند ٢٧.)
 *
 * # وتحذيرٌ كشفه القياسُ في PREFLIGHT
 *
 * (قرارُ المالك نصّاً: «**لا تقارن قائمة `vector_layers` بين Syria
 *  وRaqqa حرفيّاً**… قارن ضدّ schema capability/definition».)
 *
 * **وقِيس الفرقُ فعلاً** (٢٠٢٦-٠٨-٢١):
 *
 *	سوريا ١٦ طبقة  ·  الرقّة ١٢
 *	الناقص: aeroway · aerodrome_label · park · water_name
 *	transportation: ١٦ حقلاً مقابل ١٠
 *
 * **والسببُ ليس انحرافَ مخطّط** — نسختُه واحدةٌ في الاثنين
 * (`3.16.0`). **بل أنّ `vector_layers` تُشتَقّ من البيانات**: لا مطارَ
 * في صندوق الرقّة فلا تُعلَن طبقتُه.
 *
 * **فالحارسُ يقارن بالمقدرة لا بالحضور**: كلُّ طبقةٍ يعلنها الأثرُ
 * **يجب أن تكون معروفةً في المخطّط** — ولا يُشترط حضورُها.
 */

import { readFileSync, existsSync, readdirSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { readArchiveMeta } from './pmtiles.mjs';
import { bind, semanticShape } from './bind-style.mjs';
import { requiredRanges, rangesForText, BLOCKS } from './glyph-ranges.mjs';
import { ARABIC_CORPUS } from './corpus.mjs';

const here = dirname(fileURLToPath(import.meta.url));

/** **نسخةُ المخطّط المعتمدة** — تُرفع بقرارٍ لا بصمت. */
export const TILE_SCHEMA = '3.16.0';

/**
 * **الطبقاتُ التي يعرفها المخطّط** — مقدرةٌ لا شرط.
 *
 * **قِيست من أرشيف سوريا الكامل** (٢٠٢٦-٠٨-٢١).
 */
export const SCHEMA_LAYERS = new Set([
  'aerodrome_label', 'aeroway', 'boundary', 'building', 'housenumber',
  'landcover', 'landuse', 'mountain_peak', 'park', 'place', 'poi',
  'transportation', 'transportation_name', 'water', 'water_name', 'waterway',
]);

/**
 * **وما لا تقوم الملاحةُ بدونه** — يُشترط حضورُه في كلّ أثر.
 *
 * **وثلاثٌ لا ستَّ عشرةَ**: خريطةٌ بلا مطارٍ تعمل، **وبلا طرقٍ لا.**
 */
export const REQUIRED_LAYERS = ['transportation', 'transportation_name', 'place'];

/** **صندوقُ سوريا** — وما خرج عنه ليس بياناتِنا. */
export const SYRIA_BBOX = [35.5, 32.0, 42.5, 37.5];

const problems = [];
const notes = [];
const fail = (m) => problems.push(m);
const note = (m) => notes.push(m);

function insideSyria(b) {
  return b[0] >= SYRIA_BBOX[0] - 0.5 && b[1] >= SYRIA_BBOX[1] - 0.5 &&
    b[2] <= SYRIA_BBOX[2] + 0.5 && b[3] <= SYRIA_BBOX[3] + 0.5;
}

function checkArchive(path, label, { region }) {
  let m;
  try {
    m = readArchiveMeta(path);
  } catch (e) {
    fail(`${label}: لا يُفتح — ${e.message}`);
    return null;
  }
  if (m.tileSchema !== TILE_SCHEMA) {
    fail(`${label}: نسخةُ المخطّط ${m.tileSchema} — والمعتمدةُ ${TILE_SCHEMA}`);
  }
  if (!m.tileCount || m.tileCount < 1) fail(`${label}: لا بلاطاتِ فيه`);
  if (m.maxZoom <= m.minZoom) fail(`${label}: مدى تقريبٍ فاسد z${m.minZoom}–z${m.maxZoom}`);

  // **المقدرةُ لا الحضور** — انظر أعلى الملفّ.
  for (const id of m.vectorLayers) {
    if (!SCHEMA_LAYERS.has(id)) fail(`${label}: طبقةٌ خارجَ المخطّط: ${id}`);
  }
  for (const id of REQUIRED_LAYERS) {
    if (!m.vectorLayers.includes(id)) fail(`${label}: طبقةٌ لازمةٌ غائبة: ${id}`);
  }
  // **والعربيّةُ شرطُ قبول** — لا زينة.
  const names = m.fields['transportation_name'] ?? [];
  if (!names.includes('name:ar')) fail(`${label}: لا حقلَ name:ar في أسماء الطرق`);
  if (!m.attribution || !/OpenStreetMap/i.test(m.attribution)) {
    fail(`${label}: الإسنادُ ناقصٌ أو بلا OpenStreetMap`);
  }
  if (!insideSyria(m.bounds)) fail(`${label}: حدودٌ خارجَ سوريا ${m.bounds.join(',')}`);
  if (region && !insideSyria(m.bounds)) fail(`${label}: حزمةٌ خارجَ صندوقها`);
  note(`${label}: ${m.tileCount} بلاطة · z${m.minZoom}–z${m.maxZoom} · ` +
    `${m.vectorLayers.length} طبقة · نسخةُ البيانات ${m.dataVersion}`);
  return m;
}

/** **والنمطُ يُفحص ضدّ ما يعلنه الأثرُ فعلاً.** */
function checkStyle(baseMeta) {
  const canonical = JSON.parse(
    readFileSync(join(here, '..', 'style', 'rahalgo.style.json'), 'utf8'),
  );
  const known = new Set(baseMeta ? baseMeta.vectorLayers : SCHEMA_LAYERS);
  for (const l of canonical.layers) {
    const sl = l['source-layer'];
    if (!sl) continue;
    if (!SCHEMA_LAYERS.has(sl)) fail(`النمط: الطبقةُ ${l.id} تشير إلى مصدرٍ مجهول: ${sl}`);
    else if (baseMeta && !known.has(sl)) note(`النمط: ${sl} غيرُ موجودةٍ في الأساس (مقبول)`);
  }
  for (const anchor of Object.values(canonical._anchors ?? {})) {
    if (!canonical.layers.some((l) => l.id === anchor)) {
      fail(`النمط: المِرساةُ ${anchor} غيرُ معرَّفة`);
    }
  }
  // **والارتباطاتُ الثلاثةُ دلاليّاً واحدة** — وإلّا انحرفت نسختان.
  const online = bind(canonical, { binding: 'online', tiles: 'https://a/x.pmtiles', resources: 'https://a/r' });
  const offline = bind(canonical, { binding: 'offline', tiles: 'file:///x.pmtiles', resources: '.' });
  const a = JSON.stringify(semanticShape(online));
  const b = JSON.stringify(semanticShape(offline));
  if (a !== b) fail('النمط: الارتباطان أونلاين ودونَ اتّصالٍ يختلفان دلاليّاً');
  note(`النمط: ${canonical.layers.length} طبقة · مراسٍ ${Object.keys(canonical._anchors).length}`);
}

/** **وموارِدُ الخريطة تُفحص كما تُفحص البلاطات.** */
function checkResources(outDir) {
  const dir = join(outDir, 'map-resources');
  if (!existsSync(dir)) { fail('الموارد: المجلّد غائب'); return; }
  const spec = join(dir, 'resources.json');
  if (!existsSync(spec)) { fail('الموارد: resources.json غائب'); return; }
  const r = JSON.parse(readFileSync(spec, 'utf8'));
  for (const k of ['resourcesVersion', 'styleVersion', 'glyphVersion', 'spriteVersion', 'fontstacks']) {
    if (r[k] === undefined) fail(`الموارد: الحقلُ ${k} غائب`);
  }
  // ══════════════════════════════════════════════════════════════════
  // **والنطاقاتُ تُقرأ من مصدرها الواحد لا تُكتب هنا**
  // ══════════════════════════════════════════════════════════════════
  //
  // (إغلاقُ أدوات الموارد ٢٠٢٦-٠٨-٢١.)
  //
  // **كانت ثلاثُ قوائمَ متناقضةٍ في المشروع**: ستٌّ في
  // `glyph-ranges.mjs`، وثلاثٌ يشترطها هذا الفاحص، وخمسٌ في رفيدة
  // الاختبار. **فمرَّ فحصٌ وملفٌّ لازمٌ غيرُ موجود.**
  const needed = requiredRanges();
  for (const entry of r.fontstacks ?? []) {
    // **والعقدُ صار مقيساً لا مكتوباً** — `fontstacks` كانت أسماءً
    // فصارت سجلّاتٍ فيها النسخةُ والترخيصُ والبصمةُ والنطاقاتُ
    // المبنيّة. **فيُقبل الشكلان ويُشترط الاسم.**
    const stack = typeof entry === 'string' ? entry : entry.name;
    if (!stack) { fail('الموارد: سجلُّ رصّةٍ بلا اسم'); continue; }
    if (typeof entry === 'object') {
      for (const k of ['family', 'version', 'license', 'source', 'glyphRanges']) {
        if (entry[k] === undefined || entry[k] === null) {
          fail(`الموارد: ${stack} — الحقلُ ${k} غائبٌ من العقد`);
        }
      }
      if ((entry.missingRanges ?? []).length) {
        fail(`الموارد: ${stack} — العقدُ يُقرّ بنطاقاتٍ ناقصة: ${entry.missingRanges.join(' · ')}`);
      }
    }
    const gdir = join(dir, 'glyphs', stack);
    if (!existsSync(gdir)) { fail(`الموارد: حروفُ ${stack} غائبة`); continue; }
    const have = new Set(readdirSync(gdir).filter((f) => f.endsWith('.pbf')));
    const missing = needed.filter((x) => !have.has(`${x}.pbf`));
    if (missing.length) {
      fail(`الموارد: ${stack} — نطاقاتٌ لازمةٌ مفقودة: ${missing.join(' · ')}`);
    }
    // **والمُدوَّنةُ الحقيقيّةُ تُسأل أيضاً** — فإن طلبت نطاقاً خارجَ
    // الكتل المعلنة **فثمّة كتلةٌ ناقصةٌ في `BLOCKS`.**
    const byCorpus = rangesForText(ARABIC_CORPUS);
    const uncovered = byCorpus.filter((x) => !needed.includes(x));
    if (uncovered.length) {
      fail(`الموارد: المُدوَّنةُ تطلب نطاقاً غيرَ معلنٍ في BLOCKS: ${uncovered.join(' · ')}`);
    }
    note(`الموارد: ${stack} — ${have.size} نطاقاً · لازمٌ ${needed.length} · ` +
      `كتلٌ ${BLOCKS.length}`);
  }

  // **وأسماءُ الرصّات تطابق ما يطلبه النمطُ حرفاً بحرف** (البند ٤).
  const canonical = JSON.parse(
    readFileSync(join(here, '..', 'style', 'rahalgo.style.json'), 'utf8'),
  );
  const wanted = new Set();
  for (const l of canonical.layers) {
    for (const f of l.layout?.['text-font'] ?? []) wanted.add(f);
  }
  for (const f of wanted) {
    if (!existsSync(join(dir, 'glyphs', f))) {
      fail(`الموارد: النمطُ يطلب الرصّةَ «${f}» ولا مجلّدَ لها`);
    }
  }
  if (wanted.size) note(`الموارد: رصّاتُ النمط — ${[...wanted].join(' · ')}`);

  // **ونصُّ الرخصة يُوزَّع مع الخطّ** — شرطُ OFL، والبند ٦.
  // **ولا يكفي أن يُذكر اسمُ الترخيص في العقد.**
  if (r.fontLicense) {
    if (!existsSync(join(dir, r.fontLicense.file))) {
      fail(`الموارد: العقدُ يذكر ${r.fontLicense.file} ولا ملفَّ له`);
    } else {
      note(`الموارد: الترخيص — ${r.fontLicense.spdx} · ${r.fontLicense.bytes} بايت`);
    }
  } else {
    fail('الموارد: لا ترخيصَ مرفَقٌ بالخطّ — توزيعٌ مخالفٌ لـOFL');
  }

  // **وعقدُ عنوان الحروف مُعلَن** (البند ١١).
  if (r.glyphUrlTemplate !== '{fontstack}/{range}.pbf') {
    fail(`الموارد: عقدُ العنوان غيرُ متوقَّع: ${r.glyphUrlTemplate}`);
  }

  // **والأيقوناتُ التي يطلبها النمطُ موجودةٌ في الفهرس** (البند ٨).
  checkSprite(dir, canonical);
}

/** **فهرسُ الأيقونات — والنمطُ لا يطلب ما ليس فيه.** */
function checkSprite(dir, canonical) {
  const idx = join(dir, 'sprite.json');
  const png = join(dir, 'sprite.png');
  const idx2 = join(dir, 'sprite@2x.json');
  const png2 = join(dir, 'sprite@2x.png');
  for (const f of [idx, png, idx2, png2]) {
    if (!existsSync(f)) { fail(`الأيقونات: ملفٌّ مفقود ${f.split(/[\\/]/).pop()}`); }
  }
  if (!existsSync(idx) || !existsSync(idx2)) return;
  const one = JSON.parse(readFileSync(idx, 'utf8'));
  const two = JSON.parse(readFileSync(idx2, 'utf8'));

  const wanted = new Set();
  for (const l of canonical.layers) {
    const img = JSON.stringify(l.layout?.['icon-image'] ?? '');
    for (const m of img.matchAll(/"(poi-[a-z-]+)"/g)) wanted.add(m[1]);
  }
  for (const id of wanted) {
    if (!(id in one)) fail(`الأيقونات: النمطُ يطلب «${id}» وليست في sprite.json`);
    if (!(id in two)) fail(`الأيقونات: «${id}» مفقودةٌ في sprite@2x.json`);
  }
  // **و2x ضعفُ 1x بالضبط** — وإلّا ظهرت مشوّهةً على شاشةٍ كثيفة.
  for (const id of Object.keys(one)) {
    if (!(id in two)) { fail(`الأيقونات: «${id}» في 1x وليست في 2x`); continue; }
    if (two[id].pixelRatio !== 2) fail(`الأيقونات: «${id}» في 2x بنسبةٍ ${two[id].pixelRatio}`);
    if (two[id].width !== one[id].width * 2 || two[id].height !== one[id].height * 2) {
      fail(`الأيقونات: «${id}» أبعادُ 2x ليست ضعفَ 1x`);
    }
    if (!one[id].width || !one[id].height) fail(`الأيقونات: «${id}» بأبعادٍ صفريّة`);
  }
  note(`الأيقونات: ${Object.keys(one).length} أيقونةً · 1x و2x متّسقتان · ` +
    `يطلب النمطُ ${wanted.size}`);
}

// ══════════════════════════════════════════════════════════════════════
// **وأنواعُ الفحص تُفصل — والنجاحُ لا يستعير معناه**
// ══════════════════════════════════════════════════════════════════════
//
// (المرحلة ٦ب، قرارُ المالك ٢٠٢٦-٠٨-٢١، البند ٠.)
//
// **كان أمرٌ واحدٌ يفحص كلَّ شيء** — فإذا نجحت الموارُد وغاب أرشيفُ
// البلاطات **سقط الأمرُ كلُّه.** وذلك خللٌ زائف: **الموارُد لا تحتاج
// ثلاثَ مئةِ ميغابايت لتكون سليمة.**
//
// **والعكسُ أخطر**: لو تسامح الأمرُ مع غياب الأثر **لصار «الأثرُ
// سليم» يعني «لم يُفحص أثر».**
//
// **فلكلِّ فحصٍ اسمُه وشرطُه**:
//
//	resources <مجلّد>   موارُد وحدَها      — لا يطلب أرشيفاً
//	style               النمطُ القانونيّ    — لا يطلب شيئاً
//	package <ملفّ>      أرشيفٌ واحد        — يطلب الملفَّ نفسَه
//	artifact <مجلّد>    الأثرُ كامل        — **يسقط إن غاب الأساس**
//	all <مجلّد>         الأثرُ والموارد    — **يسقط إن غاب الأساس**

const COMMANDS = ['resources', 'style', 'package', 'artifact', 'all'];

/** **الأثرُ وحدَه** — الأساسُ والحزمُ والنمطُ في سياقها. */
function validateArtifact(outDir) {
  const basePath = join(outDir, 'syria.pmtiles');
  if (!existsSync(basePath)) {
    // **ولا يُتسامح** — طُلب فحصُ أثرٍ ولا أثرَ يُفحص.
    fail('الأساس: syria.pmtiles غائب — طُلب فحصُ أثرٍ ولا أثر');
    checkStyle(null);
    return null;
  }
  const baseMeta = checkArchive(basePath, 'الأساس', { region: false });

  for (const f of readdirSync(outDir)) {
    if (f.startsWith('region-') && f.endsWith('.pmtiles')) {
      const m = checkArchive(join(outDir, f), `حزمة ${f}`, { region: true });
      // **والمخطّطُ واحدٌ في الأونلاين ودونَ اتّصال** (البند ٢٧).
      if (m && baseMeta && m.tileSchema !== baseMeta.tileSchema) {
        fail(`${f}: نسخةُ مخطّطٍ تخالف الأساس`);
      }
    }
  }
  checkStyle(baseMeta);
  return baseMeta;
}

function report(what, subject) {
  for (const n of notes) console.log(`  · ${n}`);
  if (problems.length) {
    console.error(`\n✗ ${subject} — ${problems.length} خللاً:`);
    for (const p of problems) console.error(`  ✗ ${p}`);
    process.exit(1);
  }
  console.log(`\n✓ ${what}`);
}

function main() {
  const [cmd, arg] = process.argv.slice(2);

  // **والأمرُ القديمُ يبقى يعمل** — `validate.mjs <مجلّد>` تعني `all`.
  if (cmd && !COMMANDS.includes(cmd)) {
    return run('all', cmd);
  }
  if (!cmd) {
    console.error('الاستعمال: node validate.mjs <أمر> [مسار]');
    console.error(`  الأوامر: ${COMMANDS.join(' · ')}`);
    console.error('  resources <مجلّد>  · style  · package <ملفّ>');
    console.error('  artifact <مجلّد>   · all <مجلّد>');
    process.exit(2);
  }
  return run(cmd, arg);
}

function needPath(cmd, arg) {
  if (!arg) {
    console.error(`الأمرُ «${cmd}» يحتاج مساراً`);
    process.exit(2);
  }
  if (!existsSync(arg)) {
    console.error(`المسارُ غيرُ موجود: ${arg}`);
    process.exit(2);
  }
  return arg;
}

function run(cmd, arg) {
  switch (cmd) {
    case 'style':
      checkStyle(null);
      return report('النمطُ القانونيّ سليم', 'النمطُ القانونيّ');

    case 'resources':
      checkResources(needPath(cmd, arg));
      return report('الموارُد سليمة', 'الموارُد');

    case 'package':
      checkArchive(needPath(cmd, arg), 'الحزمة', { region: true });
      return report('الحزمةُ سليمة', 'الحزمة');

    case 'artifact':
      validateArtifact(needPath(cmd, arg));
      return report('الأثرُ سليم', 'الأثر');

    case 'all': {
      const dir = needPath(cmd, arg);
      validateArtifact(dir);
      checkResources(dir);
      return report('الأثرُ والموارُد سليمة', 'الأثرُ والموارُد');
    }

    default:
      console.error(`أمرٌ غيرُ معروف: ${cmd}`);
      process.exit(2);
  }
}

if (process.argv[1] && process.argv[1].endsWith('validate.mjs')) main();
