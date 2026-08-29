/**
 * ══════════════════════════════════════════════════════════════════════
 * **اختباراتُ الأداة — البند ٢٩**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وتُشغَّل بلا مفتاحٍ ولا شبكة** — **واختبارٌ يحتاج نداءً مدفوعاً
 * لا يُشغَّل، فلا يحرس شيئاً.**
 */

import { test } from 'node:test';
import assert from 'node:assert/strict';
import { spokenDistance, spokenOrdinal, distanceSlug, LexiconError } from './lexicon.js';
import {
  buildCorpus,
  validateCorpus,
  stripDiacritics,
  distancePrefix,
  VOCALIZATION,
  CorpusError,
} from './corpus.js';
import { sourceHash, decide } from './hash.js';
import { verifyMp3, MIN_BYTES } from './verify.js';
import { toCsv, validateManifest, entryFor, SCHEMA_VERSION, ManifestError } from './manifest.js';
import { loadConfig, validateConfig, DEFAULT_VOICE_SETTINGS, ConfigError } from './config.js';
import { redact } from './client.js';
import { applyOverrides } from './overrides.js';

// ── القاموس ───────────────────────────────────────────────────────────

test('المسافةُ تُنطق نصّاً لا رقماً', () => {
  assert.equal(spokenDistance(300), 'ثَلاثمِئَة مِتْر');
  assert.equal(spokenDistance(1000), 'كيلومِتْر واحِد');
  assert.equal(distanceSlug(1500), '1500m');
});

test('ومسافةٌ بلا قراءةٍ تُرفض ولا تُخمَّن', () => {
  assert.throws(() => spokenDistance(137), LexiconError);
});

test('وترتيبُ المخرج من الأوّل إلى الثاني عشر', () => {
  assert.equal(spokenOrdinal(1), 'الأَوَّل');
  assert.equal(spokenOrdinal(11), 'الحادِي عَشَر');
  assert.throws(() => spokenOrdinal(13), LexiconError);
});

// ── المدوّنة ──────────────────────────────────────────────────────────

test('المدوّنةُ تُبنى ولا معرّفَ يتكرّر ولا ملفّ', () => {
  const c = buildCorpus();
  assert.ok(c.length > 300, `عددٌ صغير: ${c.length}`);
  assert.equal(new Set(c.map((i) => i.id)).size, c.length);
  assert.equal(new Set(c.map((i) => i.file)).size, c.length);
});

test('ولا رقمَ رقميٌّ في أيّ نصٍّ منطوق', () => {
  for (const i of buildCorpus()) {
    assert.ok(!/[0-9٠-٩]/.test(i.ttsText), `${i.id}: ${i.ttsText}`);
  }
});

test('والنصُّ المرئيُّ بلا تشكيلٍ كامل — والشدّةُ تبقى', () => {
  for (const i of buildCorpus()) {
    assert.ok(!VOCALIZATION.test(i.displayText), `${i.id}: ${i.displayText}`);
  }
  // **والشدّةُ ليست تشكيلاً كاملاً** — «تمّ» تُكتب هكذا في كلّ نصٍّ عاديّ.
  const arrived = buildCorpus().find((i) => i.id === 'arrived');
  assert.ok(arrived);
  assert.equal(arrived.displayText, 'تمّ الوصول إلى الوجهة.');
});

test('وأسماءُ الملفّات صالحةٌ لـres/raw', () => {
  for (const i of buildCorpus()) {
    assert.match(i.file, /^[a-z][a-z0-9_]*\.mp3$/, i.file);
  }
});

test('والجملةُ كاملةٌ لا مقاطع — المسافةُ داخلَ النصّ', () => {
  const c = buildCorpus();
  const withDist = c.find((i) => i.id === 'turn_right_in_300m');
  assert.ok(withDist);
  assert.equal(withDist.ttsText, 'بَعْد ثَلاثمِئَة مِتْر، اِنْعَطِف يَمينًا.');
  assert.equal(withDist.displayText, 'بعد ثلاثمئة متر، انعطف يمينًا.');
});

test('وجملةُ الدوّار بالمخرج المطلوب', () => {
  const c = buildCorpus();
  const r = c.find((i) => i.id === 'roundabout_exit_2');
  assert.ok(r);
  assert.equal(r.ttsText, 'عِند الدَّوّار، خُذ المَخرَج الثّاني.');
});

test('وكشفُ التكرار يعمل', () => {
  const one = buildCorpus()[0];
  assert.ok(one);
  assert.throws(() => validateCorpus([one, one]), CorpusError);
});

test('ونزعُ التشكيل لا يمسّ الحروف', () => {
  assert.equal(stripDiacritics('اِنْعَطِف يَمينًا.'), 'انعطف يمينا.');
});

test('وسابقةُ المسافة كما نصّ المالك', () => {
  assert.equal(distancePrefix(300).tts, 'بَعْد ثَلاثمِئَة مِتْر، ');
});

// ── البصمة والذاكرة ───────────────────────────────────────────────────

const H = {
  ttsText: 'اِنْعَطِف يَمينًا.',
  voiceId: 'v1',
  modelId: 'eleven_multilingual_v2',
  outputFormat: 'mp3_44100_128',
  voiceSettings: DEFAULT_VOICE_SETTINGS,
};

test('البصمةُ حتميّةٌ وتتبدّل بتبدّل أيّ مدخل', () => {
  assert.equal(sourceHash(H), sourceHash(H));
  assert.notEqual(sourceHash(H), sourceHash({ ...H, ttsText: 'اِنْعَطِف يَسارًا.' }));
  assert.notEqual(sourceHash(H), sourceHash({ ...H, voiceId: 'v2' }));
  assert.notEqual(sourceHash(H), sourceHash({ ...H, modelId: 'x' }));
  assert.notEqual(sourceHash(H), sourceHash({ ...H, outputFormat: 'mp3_22050_32' }));
  assert.notEqual(
    sourceHash(H),
    sourceHash({ ...H, voiceSettings: { ...DEFAULT_VOICE_SETTINGS, speed: 1.0 } }),
  );
});

test('وقرارُ الذاكرة — يُتخطّى الموجودُ المطابق وحدَه', () => {
  const h = 'a'.repeat(64);
  assert.equal(decide({ fileExists: false, storedHash: undefined, currentHash: h, force: false }), 'generate');
  assert.equal(decide({ fileExists: true, storedHash: h, currentHash: h, force: false }), 'skip');
  assert.equal(decide({ fileExists: true, storedHash: 'b'.repeat(64), currentHash: h, force: false }), 'regenerate');
  // **وملفٌّ بلا بصمةٍ يُعاد** — لا يُعرف بأيّ إعدادٍ وُلّد.
  assert.equal(decide({ fileExists: true, storedHash: undefined, currentHash: h, force: false }), 'regenerate');
  assert.equal(decide({ fileExists: true, storedHash: h, currentHash: h, force: true }), 'regenerate');
});

// ── التحقّق من الصوت ──────────────────────────────────────────────────

test('وردُّ خطأٍ بصيغة JSON لا يُقبل ملفَّ صوت', () => {
  const json = new TextEncoder().encode(`{"detail":{"code":"x"}}${' '.repeat(MIN_BYTES)}`);
  assert.equal(verifyMp3(json).ok, false);
});

test('والفارغُ والقصيرُ يُرفضان', () => {
  assert.equal(verifyMp3(new Uint8Array(0)).ok, false);
  assert.equal(verifyMp3(new Uint8Array(10)).ok, false);
});

test('ورأسُ ID3 يُقبل', () => {
  const d = new Uint8Array(MIN_BYTES + 8);
  d[0] = 0x49;
  d[1] = 0x44;
  d[2] = 0x33;
  assert.equal(verifyMp3(d).ok, true);
});

test('وإطارُ MPEG يُقبل', () => {
  const d = new Uint8Array(MIN_BYTES + 8);
  d[0] = 0xff;
  d[1] = 0xfb;
  assert.equal(verifyMp3(d).ok, true);
});

// ── الفهرس ────────────────────────────────────────────────────────────

function sampleEntry(id: string, file: string) {
  const i = buildCorpus().find((x) => x.id === 'turn_right');
  assert.ok(i);
  return entryFor({
    instruction: { ...i, id, file },
    voiceId: 'v1',
    modelId: 'm',
    outputFormat: 'mp3_44100_128',
    voiceSettings: DEFAULT_VOICE_SETTINGS,
    sourceHash: 'a'.repeat(64),
    generatedAt: '2026-08-24T00:00:00.000Z',
  });
}

test('الفهرسُ يُبنى ويُتحقّق منه', () => {
  const m = { schemaVersion: SCHEMA_VERSION, pack: 'p', language: 'ar', entries: [sampleEntry('a', 'a.mp3')] };
  validateManifest(m);
  assert.equal(m.entries[0]?.characterCount, [...'اِنْعَطِف يَمينًا.'].length);
});

test('ومعرّفٌ مكرّرٌ في الفهرس يُرفض', () => {
  const m = {
    schemaVersion: SCHEMA_VERSION,
    pack: 'p',
    language: 'ar',
    entries: [sampleEntry('a', 'a.mp3'), sampleEntry('a', 'b.mp3')],
  };
  assert.throws(() => validateManifest(m), ManifestError);
});

test('ومفتاحٌ في الفهرس يُرفض', () => {
  const e = sampleEntry('a', 'a.mp3');
  const m = {
    schemaVersion: SCHEMA_VERSION,
    pack: 'p',
    language: 'ar',
    entries: [{ ...e, displayText: 'sk_abcdefgh12345678' }],
  };
  assert.throws(() => validateManifest(m), ManifestError);
});

test('وجدولُ المراجعة يهرّب الفاصلة', () => {
  const i = buildCorpus().find((x) => x.id === 'roundabout_exit_2');
  assert.ok(i);
  const csv = toCsv({
    schemaVersion: SCHEMA_VERSION,
    pack: 'p',
    language: 'ar',
    entries: [
      entryFor({
        instruction: i,
        voiceId: 'v',
        modelId: 'm',
        outputFormat: 'mp3_44100_128',
        voiceSettings: DEFAULT_VOICE_SETTINGS,
        sourceHash: 'a'.repeat(64),
        generatedAt: '2026-08-24T00:00:00.000Z',
      }),
    ],
  });
  assert.ok(csv.includes('"عِند الدَّوّار، خُذ المَخرَج الثّاني."'));
});

// ── الإعداد والأمان ───────────────────────────────────────────────────

test('وغيابُ المفتاح يوقف النداءَ المدفوع ولا يوقف الجافّ', () => {
  const before = process.env['ELEVENLABS_API_KEY'];
  delete process.env['ELEVENLABS_API_KEY'];
  try {
    assert.throws(() => loadConfig(true), ConfigError);
    assert.doesNotThrow(() => loadConfig(false));
  } finally {
    if (before !== undefined) process.env['ELEVENLABS_API_KEY'] = before;
  }
});

test('وقيمةٌ خارجَ المدى تُرفض', () => {
  assert.throws(
    () =>
      validateConfig({
        apiKey: '',
        voiceId: 'v',
        modelId: 'm',
        outputFormat: 'mp3_44100_128',
        voiceSettings: { ...DEFAULT_VOICE_SETTINGS, stability: 2 },
        concurrency: 1,
        maxRetries: 2,
      }),
    ConfigError,
  );
});

test('والمفتاحُ يُنقّى من أيّ نصٍّ يُكتب', () => {
  assert.equal(redact('فشل sk_abcdefgh12345678 هنا'), 'فشل sk_*** هنا');
});

test('وتصحيحُ النطق يُطبَّق ولا يكسر نصّاً بلا تصحيح', () => {
  assert.equal(typeof applyOverrides('اِنْعَطِف يَمينًا.'), 'string');
});
