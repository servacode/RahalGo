/**
 * ══════════════════════════════════════════════════════════════════════
 * **إعدادُ الحزمة — رقمٌ واحدٌ في موضعٍ واحد**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مواصفةُ المالك ٢٠٢٦-٠٨-٢٣، البند ٣: «اجعل جميعها Configuration
 *  وليست أرقاماً موزّعةً داخل الكود».)
 *
 * # والمفتاحُ لا يسكن هنا ولا في أيّ ملفّ
 *
 * **يُقرأ من البيئة وحدَها** — ولا يُطبع ولا يُكتب في تقريرٍ ولا
 * فهرس. **ومفتاحٌ يمرّ في سجلٍّ مفتاحٌ ضاع.**
 */

export interface VoiceSettings {
  readonly stability: number;
  readonly similarity_boost: number;
  readonly style: number;
  readonly use_speaker_boost: boolean;
  readonly speed: number;
}

export interface Config {
  readonly apiKey: string;
  readonly voiceId: string;
  readonly modelId: string;
  readonly outputFormat: string;
  readonly voiceSettings: VoiceSettings;
  readonly concurrency: number;
  readonly maxRetries: number;
}

/** **الإعدادُ المبدئيّ** — مواصفةُ المالك حرفاً بحرف. */
export const DEFAULT_VOICE_SETTINGS: VoiceSettings = {
  stability: 0.82,
  similarity_boost: 0.78,
  style: 0.0,
  use_speaker_boost: true,
  speed: 1.03,
};

export const DEFAULT_MODEL_ID = 'eleven_multilingual_v2';
export const DEFAULT_OUTPUT_FORMAT = 'mp3_44100_128';

/** **اسمُ الصوت الذي طلبه المالك** — يُبحث عنه إن لم يُعطَ معرّف. */
export const PREFERRED_VOICE_NAME = 'rahalgo2';

export class ConfigError extends Error {}

/**
 * **يقرأ الإعدادَ من البيئة ويتحقّق منه.**
 *
 * **ولا يُقبل مفتاحٌ فارغ** — **وتشغيلٌ يمضي بلا مفتاحٍ يسقط عند أوّل
 * نداءٍ بعد أن يكون قد بنى خطّةً كاملة**، فيُظنّ العطبُ في الخطّة.
 *
 * @param requireKey **يُطلب المفتاحُ للنداءات المدفوعة وحدَها** —
 *   والتشغيلُ الجافُّ يعمل بلاه.
 */
export function loadConfig(requireKey: boolean): Config {
  const apiKey = (process.env['ELEVENLABS_API_KEY'] ?? '').trim();
  if (requireKey && apiKey === '') {
    throw new ConfigError(
      'ELEVENLABS_API_KEY غيرُ مضبوط.\n' +
        '  PowerShell:  $env:ELEVENLABS_API_KEY = "…"\n' +
        '  bash:        export ELEVENLABS_API_KEY="…"\n' +
        'ولا يُكتب في ملفٍّ داخل المستودع.',
    );
  }
  const cfg: Config = {
    apiKey,
    voiceId: (process.env['ELEVENLABS_VOICE_ID'] ?? '').trim(),
    modelId: (process.env['ELEVENLABS_MODEL_ID'] ?? '').trim() || DEFAULT_MODEL_ID,
    outputFormat: (process.env['ELEVENLABS_OUTPUT_FORMAT'] ?? '').trim() || DEFAULT_OUTPUT_FORMAT,
    voiceSettings: DEFAULT_VOICE_SETTINGS,
    concurrency: Number(process.env['NAVVOICE_CONCURRENCY'] ?? '1'),
    maxRetries: Number(process.env['NAVVOICE_MAX_RETRIES'] ?? '4'),
  };
  validateConfig(cfg);
  return cfg;
}

/** **ويُتحقّق من المدى** — **وقيمةٌ خارجَ الحدّ تردّها الخدمةُ بخطأٍ غامض.** */
export function validateConfig(c: Config): void {
  const s = c.voiceSettings;
  const range = (n: number, lo: number, hi: number, name: string): void => {
    if (!Number.isFinite(n) || n < lo || n > hi) {
      throw new ConfigError(`${name} خارجَ المدى [${lo}, ${hi}]: ${n}`);
    }
  };
  range(s.stability, 0, 1, 'stability');
  range(s.similarity_boost, 0, 1, 'similarity_boost');
  range(s.style, 0, 1, 'style');
  range(s.speed, 0.7, 1.2, 'speed');
  if (!Number.isInteger(c.concurrency) || c.concurrency < 1 || c.concurrency > 4) {
    throw new ConfigError(`concurrency محافظٌ بين ١ و٤: ${c.concurrency}`);
  }
  if (!Number.isInteger(c.maxRetries) || c.maxRetries < 0 || c.maxRetries > 8) {
    throw new ConfigError(`maxRetries بين ٠ و٨: ${c.maxRetries}`);
  }
  if (!/^mp3_\d+_\d+$/.test(c.outputFormat)) {
    throw new ConfigError(`صيغةٌ غيرُ معروفة: ${c.outputFormat}`);
  }
}

/** **ويُخفى المعرّفُ في التقارير** — **لا سرّاً بل نظافةً.** */
export function shortId(id: string): string {
  return id.length <= 10 ? id : `${id.slice(0, 6)}…${id.slice(-4)}`;
}
