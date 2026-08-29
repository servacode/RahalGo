/**
 * ══════════════════════════════════════════════════════════════════════
 * **واجهةُ الأداة — جافٌّ ثمّ تأهيلٌ ثمّ توليدٌ كامل**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مواصفةُ المالك ٢٠٢٦-٠٨-٢٣.)
 *
 *	dry-run   لا ينادي الخدمةَ إطلاقاً — البند ٢٠
 *	voices    يعرض أصواتَ الحساب — البند ٢
 *	qualify   عيّناتُ الاعتماد وحدَها — البند ٢٣
 *	generate  الحزمةُ الكاملةُ بعد «اعتمد الصوت» — البند ٢٥
 *	pack      التقريرُ والأرشيف — البندان ٢٦ و٢٧
 *
 * **ولا يُطبع المفتاحُ في أيّ منها.**
 */

import { existsSync, mkdirSync, readFileSync, writeFileSync, statSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';
import {
  loadConfig,
  ConfigError,
  PREFERRED_VOICE_NAME,
  shortId,
  type Config,
} from './config.js';
import { buildCorpus, type Instruction } from './corpus.js';
import { sourceHash, decide } from './hash.js';
import { listVoices, synthesize, withRetry, redact } from './client.js';
import { verifyMp3 } from './verify.js';
import {
  entryFor,
  toCsv,
  validateManifest,
  SCHEMA_VERSION,
  type Manifest,
  type ManifestEntry,
} from './manifest.js';

const HERE = dirname(fileURLToPath(import.meta.url));
const ROOT = join(HERE, '..');
const OUT = join(ROOT, 'output');
const AUDIO = join(OUT, 'audio');
const RAW = join(OUT, 'android_raw');
const QUAL = join(OUT, 'qualification');
const REPORTS = join(ROOT, 'reports');
const STATE = join(OUT, 'state.json');

/** **عيّناتُ الاعتماد** — البند ٢٣، وفيها ما نصّ عليه المالك بالحرف. */
const QUALIFICATION_IDS: readonly string[] = [
  'turn_right_in_300m',
  'turn_left_in_250m',
  'roundabout_exit_2',
  'roundabout_exit_11',
  'keep_right',
  'keep_left',
  'continue_straight',
  'recalculating_route',
  'arrived',
  'gps_weak',
  'turn_right',
  'turn_left',
  'destination_right',
  'uturn',
  'turn_right_in_1000m',
];

interface State {
  readonly hashes: Record<string, string>;
}

function readState(): State {
  if (!existsSync(STATE)) return { hashes: {} };
  try {
    const raw: unknown = JSON.parse(readFileSync(STATE, 'utf8'));
    const h = (raw as State).hashes;
    return { hashes: typeof h === 'object' && h !== null ? h : {} };
  } catch {
    // **وحالٌ تالفةٌ تُهمَل ولا تُسقط التشغيل** — **وأسوأُ ما تفعله
    // أن تُعيد توليدَ ما وُلّد، ولا تفسد ملفّاً.**
    return { hashes: {} };
  }
}

function writeState(s: State): void {
  mkdirSync(OUT, { recursive: true });
  writeFileSync(STATE, JSON.stringify(s, null, 2), 'utf8');
}

function ensureDirs(): void {
  for (const d of [OUT, AUDIO, RAW, QUAL, REPORTS]) mkdirSync(d, { recursive: true });
}

interface Planned {
  readonly instruction: Instruction;
  readonly hash: string;
  readonly decision: 'generate' | 'skip' | 'regenerate';
  readonly target: string;
}

function plan(cfg: Config, list: readonly Instruction[], dir: string, force: boolean): Planned[] {
  const state = readState();
  return list.map((i) => {
    const hash = sourceHash({
      ttsText: i.ttsText,
      voiceId: cfg.voiceId,
      modelId: cfg.modelId,
      outputFormat: cfg.outputFormat,
      voiceSettings: cfg.voiceSettings,
    });
    const target = join(dir, i.file);
    return {
      instruction: i,
      hash,
      target,
      decision: decide({
        fileExists: existsSync(target),
        storedHash: state.hashes[i.id],
        currentHash: hash,
        force,
      }),
    };
  });
}

function summarize(planned: readonly Planned[]): {
  generate: number;
  skip: number;
  regenerate: number;
  chars: number;
} {
  let generate = 0;
  let skip = 0;
  let regenerate = 0;
  let chars = 0;
  for (const p of planned) {
    if (p.decision === 'skip') {
      skip++;
      continue;
    }
    if (p.decision === 'generate') generate++;
    else regenerate++;
    chars += [...p.instruction.ttsText].length;
  }
  return { generate, skip, regenerate, chars };
}

function printPlan(cfg: Config, planned: readonly Planned[]): void {
  const s = summarize(planned);
  console.log('── الخطّة ──');
  console.log(`  التعليمات        ${planned.length}`);
  console.log(`  ستُولَّد          ${s.generate}`);
  console.log(`  ستُعاد            ${s.regenerate}`);
  console.log(`  ستُتخطّى         ${s.skip}`);
  console.log(`  أحرفٌ سترسَل      ${s.chars}`);
  console.log(`  الصوت            ${cfg.voiceId ? shortId(cfg.voiceId) : '(غيرُ محدَّد)'}`);
  console.log(`  النموذج          ${cfg.modelId}`);
  console.log(`  الصيغة           ${cfg.outputFormat}`);
  console.log(`  الإعداد          ${JSON.stringify(cfg.voiceSettings)}`);
}

/**
 * **ويُحسم الصوتُ قبل أيّ نداءٍ مدفوع** — البند ٢.
 *
 * **ولا يُختار واحدٌ من عدّةٍ بلا إذن** — **وصوتٌ خطأٌ يعني الحزمةَ
 * كلَّها تُعاد**، وثمنُها مرّتان.
 */
async function resolveVoice(cfg: Config): Promise<Config> {
  if (cfg.voiceId !== '') return cfg;
  const voices = await withRetry(cfg.maxRetries, () => listVoices(cfg));
  const named = voices.filter((v) => v.name.trim().toLowerCase() === PREFERRED_VOICE_NAME);
  if (named.length === 1) {
    const v = named[0];
    if (v !== undefined) {
      console.log(`الصوت: ${v.name} · ${v.voiceId} · ${v.category}`);
      return { ...cfg, voiceId: v.voiceId };
    }
  }
  const mine = voices.filter((v) => v.category !== 'premade');
  console.log('أصواتُ الحساب غيرُ الجاهزة:');
  for (const v of mine) console.log(`  ${v.name} · ${v.voiceId} · ${v.category}`);
  throw new ConfigError(
    `لم يُحسم الصوت. اضبط ELEVENLABS_VOICE_ID، أو سمِّ صوتاً باسم «${PREFERRED_VOICE_NAME}».`,
  );
}

async function generateInto(
  cfg: Config,
  list: readonly Instruction[],
  dir: string,
  force: boolean,
): Promise<{ entries: ManifestEntry[]; failed: string[]; retries: number }> {
  ensureDirs();
  mkdirSync(dir, { recursive: true });
  const planned = plan(cfg, list, dir, force);
  printPlan(cfg, planned);

  const state = readState();
  const hashes: Record<string, string> = { ...state.hashes };
  const entries: ManifestEntry[] = [];
  const failed: string[] = [];
  let retries = 0;
  let done = 0;

  for (const p of planned) {
    const i = p.instruction;
    if (p.decision === 'skip') {
      entries.push(mkEntry(cfg, i, p.hash, statSync(p.target).mtime.toISOString()));
      continue;
    }
    try {
      const data = await withRetry(
        cfg.maxRetries,
        async () => {
          const bytes = await synthesize(cfg, i.ttsText);
          const v = verifyMp3(bytes);
          // **وفشلُ التحقّق يُعاد كما يُعاد خطأُ شبكة** — البند ٢٢:
          // **ملفٌّ تالفٌ ليس نجاحاً.**
          if (!v.ok) throw new Error(`تحقّقٌ فاشل: ${v.why}`);
          return bytes;
        },
        (n, why) => {
          retries++;
          console.log(`  إعادةٌ ${n} — ${i.id}: ${why}`);
        },
      );
      writeFileSync(p.target, data);
      hashes[i.id] = p.hash;
      writeState({ hashes });
      entries.push(mkEntry(cfg, i, p.hash, new Date().toISOString()));
      done++;
      console.log(`  ✔ ${i.file}  (${data.byteLength} بايت)`);
    } catch (e) {
      failed.push(`${i.id}: ${redact(e instanceof Error ? e.message : String(e))}`);
      console.log(`  ✘ ${i.file}`);
    }
  }
  console.log(`تمّ: ${done} مولَّداً · ${failed.length} ساقطاً`);
  return { entries, failed, retries };
}

function mkEntry(cfg: Config, i: Instruction, hash: string, at: string): ManifestEntry {
  return entryFor({
    instruction: i,
    voiceId: cfg.voiceId,
    modelId: cfg.modelId,
    outputFormat: cfg.outputFormat,
    voiceSettings: cfg.voiceSettings,
    sourceHash: hash,
    generatedAt: at,
  });
}

function writeManifest(entries: readonly ManifestEntry[], dir: string): Manifest {
  const m: Manifest = {
    schemaVersion: SCHEMA_VERSION,
    pack: 'rahalgo-navigation-ar-elevenlabs',
    language: 'ar',
    entries,
  };
  validateManifest(m);
  writeFileSync(join(dir, 'manifest.json'), `${JSON.stringify(m, null, 2)}\n`, 'utf8');
  writeFileSync(join(dir, 'manifest.csv'), toCsv(m), 'utf8');
  return m;
}

async function main(): Promise<number> {
  const cmd = process.argv[2] ?? 'dry-run';
  const force = process.argv.includes('--force');
  const corpus = buildCorpus();

  if (cmd === 'dry-run') {
    // **ولا مفتاحَ يُطلب هنا** — البند ٢٠: لا نداءَ إطلاقاً.
    const cfg = loadConfig(false);
    ensureDirs();
    printPlan(cfg, plan(cfg, corpus, AUDIO, force));
    return 0;
  }

  if (cmd === 'voices') {
    const cfg = loadConfig(true);
    for (const v of await withRetry(cfg.maxRetries, () => listVoices(cfg))) {
      console.log(`${v.name} · ${v.voiceId} · ${v.category}`);
    }
    return 0;
  }

  if (cmd === 'sample') {
    // **عيّنةٌ واحدةٌ لا غير** — طلبُ المالك: «تولّد صوتاً واحداً، إن
    // طابق قلتُ أكمل». **وخمس عشرة عيّنةً قبل الحكم مالٌ يُصرف على
    // صوتٍ قد يُرفض.**
    const cfg = await resolveVoice(loadConfig(true));
    const ids = (process.argv[3] ?? 'turn_right_in_300m').split(',').map((s) => s.trim());
    const picked = ids.map((id) => {
      const one = corpus.find((i) => i.id === id);
      if (one === undefined) throw new Error(`لا تعليمةَ بهذا المعرّف: ${id}`);
      return one;
    });
    // **وتُعاد دائماً وإن لم تتبدّل بصمتُها** — **عيّنةٌ يُطلب سماعُها
    // ثمّ تُتخطّى لأنّها موجودةٌ عبثٌ**، والمالكُ ينتظر ملفّاً جديداً.
    const r = await generateInto(cfg, picked, QUAL, true);
    for (const one of picked) console.log(`  ${one.file}  ←  ${one.ttsText}`);
    return r.failed.length === 0 ? 0 : 1;
  }

  if (cmd === 'qualify') {
    const cfg = await resolveVoice(loadConfig(true));
    const picked = corpus.filter((i) => QUALIFICATION_IDS.includes(i.id));
    const missing = QUALIFICATION_IDS.filter((id) => !corpus.some((i) => i.id === id));
    if (missing.length > 0) throw new Error(`عيّناتٌ غيرُ موجودةٍ في المدوّنة: ${missing.join(', ')}`);
    const r = await generateInto(cfg, picked, QUAL, force);
    writeManifest(r.entries, QUAL);
    console.log(`\nالعيّنات في: ${QUAL}`);
    console.log('استمع إليها — ولا تُولَّد الحزمةُ الكاملةُ قبل «اعتمد الصوت».');
    return r.failed.length === 0 ? 0 : 1;
  }

  if (cmd === 'generate') {
    const cfg = await resolveVoice(loadConfig(true));
    const r = await generateInto(cfg, corpus, AUDIO, force);
    writeManifest(r.entries, OUT);
    // **ونسخةٌ مسطّحةٌ لأندرويد** — البند ١٧: **`res/raw` لا يعرف
    // مجلّداتٍ داخليّة.**
    mkdirSync(RAW, { recursive: true });
    for (const e of r.entries) {
      const src = join(AUDIO, e.file);
      if (existsSync(src)) writeFileSync(join(RAW, e.file), readFileSync(src));
    }
    return r.failed.length === 0 ? 0 : 1;
  }

  console.error(`أمرٌ غيرُ معروف: ${cmd}`);
  return 2;
}

main()
  .then((code) => process.exit(code))
  .catch((e: unknown) => {
    console.error(redact(e instanceof Error ? e.message : String(e)));
    process.exit(1);
  });
