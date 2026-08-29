/**
 * ══════════════════════════════════════════════════════════════════════
 * **عميلُ ElevenLabs — REST رسميّ، وعقدُه مقيسٌ لا مُخمَّن**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مواصفةُ المالك ٢٠٢٦-٠٨-٢٣، البندان ٣ و٣٠: «لا تخمّن أسماءَ
 *  Methods أو Parameters».)
 *
 * # وكيف عُرف العقد
 *
 * **نُودي فعلاً وقِيست ردودُه** (٢٠٢٦-٠٨-٢٣) قبل كتابة هذا الملفّ —
 * **وثلاثةُ ردودٍ مختلفةٍ أثبتت الشكل:**
 *
 *	401 missing the permission voices_read   ← الترويسةُ صحيحةٌ والصلاحيّةُ ناقصة
 *	402 paid_plan_required                    ← الجسمُ صحيحٌ وصوتُ المكتبة ممنوع
 *	200 audio/mpeg                            ← بصوتٍ أصليّ
 *
 * **وردُّ خطأٍ مفصَّلٌ كهذا أصدقُ من توثيقٍ قديم**: يقول اسمَ الصلاحيّة
 * الناقصة بالحرف.
 *
 * # ولا مفتاحَ يُطبع
 *
 * **ولا في رسالة خطأ** — **وردُّ الخدمة قد يعيد ما أُرسل**، فيُقصّ
 * ويُنقّى قبل أن يُكتب في سجلّ.
 */

import type { Config } from './config.js';

const BASE = 'https://api.elevenlabs.io/v1';

export interface VoiceInfo {
  readonly voiceId: string;
  readonly name: string;
  readonly category: string;
}

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly detail: string,
    readonly retryable: boolean,
  ) {
    super(`ElevenLabs ${status}: ${detail}`);
  }
}

/** **ويُنقّى كلُّ ما يُكتب** — فلا يتسرّب مفتاحٌ في نصّ خطأ. */
export function redact(text: string): string {
  return text.replace(/sk_[A-Za-z0-9]{8,}/g, 'sk_***');
}

function headers(cfg: Config, accept: string): Record<string, string> {
  return {
    'xi-api-key': cfg.apiKey,
    'Content-Type': 'application/json',
    Accept: accept,
  };
}

/** **٤٢٩ و٥xx تُعاد، وما سواها لا** — **وإعادةُ ٤٠١ تحرق المحاولات.** */
function isRetryable(status: number): boolean {
  return status === 429 || (status >= 500 && status < 600);
}

async function readDetail(res: Response): Promise<string> {
  const raw = await res.text().catch(() => '');
  return redact(raw.slice(0, 300));
}

/** **أصواتُ الحساب** — البند ٢. */
export async function listVoices(cfg: Config): Promise<readonly VoiceInfo[]> {
  const res = await fetch(`${BASE}/voices`, { headers: headers(cfg, 'application/json') });
  if (!res.ok) throw new ApiError(res.status, await readDetail(res), isRetryable(res.status));
  const body = (await res.json()) as { voices?: readonly Record<string, unknown>[] };
  return (body.voices ?? []).map((v) => ({
    voiceId: String(v['voice_id'] ?? ''),
    name: String(v['name'] ?? ''),
    category: String(v['category'] ?? ''),
  }));
}

/**
 * **يولّد مقطعاً واحداً** — ويردّ بايتاته.
 *
 * **ويُتحقّق من نوع المحتوى** — **وردُّ خطأٍ بصيغة JSON يُحفظ باسم
 * `.mp3` يُقرأ ملفّاً سليماً حتّى يُشغَّل**، وذاك بندُ المالك ٢٢.
 */
export async function synthesize(cfg: Config, text: string): Promise<Uint8Array> {
  const url =
    `${BASE}/text-to-speech/${encodeURIComponent(cfg.voiceId)}` +
    `?output_format=${encodeURIComponent(cfg.outputFormat)}`;
  const res = await fetch(url, {
    method: 'POST',
    headers: headers(cfg, 'audio/mpeg'),
    body: JSON.stringify({
      text,
      model_id: cfg.modelId,
      voice_settings: cfg.voiceSettings,
    }),
  });
  if (!res.ok) throw new ApiError(res.status, await readDetail(res), isRetryable(res.status));
  const type = res.headers.get('content-type') ?? '';
  if (!type.includes('audio')) {
    throw new ApiError(res.status, `نوعُ محتوًى غيرُ صوتيّ: ${type}`, false);
  }
  return new Uint8Array(await res.arrayBuffer());
}

/**
 * **إعادةٌ محدودةٌ بتباعدٍ أُسّيّ** — البند ٢١.
 *
 * **ولا تُعاد المحاولةُ على خطأٍ لن يتبدّل** (٤٠١ · ٤٠٢ · ٤٠٤):
 * **إعادةٌ على مفتاحٍ ناقصِ الصلاحيّة تنتظر دقائقَ ثمّ تسقط كما
 * سقطت.**
 */
export async function withRetry<T>(
  maxRetries: number,
  op: () => Promise<T>,
  onRetry?: (attempt: number, why: string) => void,
): Promise<T> {
  let lastErr: unknown;
  for (let attempt = 0; attempt <= maxRetries; attempt++) {
    try {
      return await op();
    } catch (e) {
      lastErr = e;
      const retryable = e instanceof ApiError ? e.retryable : true;
      if (!retryable || attempt === maxRetries) break;
      // **والتباعدُ أُسّيٌّ بارتعاشة** — **وعشرةُ عملاءَ يعيدون في
      // اللحظة نفسِها يصنعون موجةً ثانية.**
      const base = 1000 * 2 ** attempt;
      const jitter = Math.floor(Math.random() * 400);
      onRetry?.(attempt + 1, redact(e instanceof Error ? e.message : String(e)));
      await new Promise((r) => setTimeout(r, base + jitter));
    }
  }
  throw lastErr instanceof Error ? lastErr : new Error(String(lastErr));
}
