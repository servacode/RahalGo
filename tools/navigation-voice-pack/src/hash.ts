/**
 * ══════════════════════════════════════════════════════════════════════
 * **بصمةُ الطلب — تمنع دفعَ الثمن مرّتين**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مواصفةُ المالك ٢٠٢٦-٠٨-٢٣، البند ١٩.)
 *
 * **والخدمةُ مدفوعةٌ بالحرف** — **فتوليدٌ يُعاد بلا سببٍ مالٌ يُحرق.**
 *
 * # وما يدخل البصمة
 *
 *	ttsText · voiceId · modelId · voiceSettings · outputFormat
 *
 * **وكلُّها تُبدّل الصوتَ الناتج** — فتبديلُ أيٍّ منها يوجب إعادة
 * التوليد، **وثباتُها كلِّها يوجب التخطّي.**
 *
 * **ولا يدخلها اسمُ الملفّ** — **وإعادةُ تسميةٍ لا تُبدّل صوتاً**،
 * ولو دخلت لَأُعيد توليدُ الحزمة كلِّها يومَ نرتّب الأسماء.
 *
 * **ولا يدخلها المفتاح** — لا لأنّه لا يؤثّر فحسب، **بل لأنّ بصمةً
 * تُكتب في فهرسٍ لا يجوز أن تُشتقّ من سرّ.**
 */

import { createHash } from 'node:crypto';
import type { VoiceSettings } from './config.js';

export interface HashInput {
  readonly ttsText: string;
  readonly voiceId: string;
  readonly modelId: string;
  readonly outputFormat: string;
  readonly voiceSettings: VoiceSettings;
}

/**
 * **بصمةٌ حتميّة** — **وترتيبُ الحقول ثابتٌ لا من `Object.keys`**:
 * **ترتيبٌ يتبدّل يبدّل البصمةَ فيُعاد توليدُ ما لم يتغيّر.**
 */
export function sourceHash(input: HashInput): string {
  const s = input.voiceSettings;
  const canonical = [
    `text=${input.ttsText}`,
    `voice=${input.voiceId}`,
    `model=${input.modelId}`,
    `format=${input.outputFormat}`,
    `stability=${s.stability}`,
    `similarity=${s.similarity_boost}`,
    `style=${s.style}`,
    `boost=${s.use_speaker_boost ? '1' : '0'}`,
    `speed=${s.speed}`,
  ].join('\n');
  return createHash('sha256').update(canonical, 'utf8').digest('hex');
}

/** **وقرارُ التخطّي** — يُقرأ في التشغيل الجافّ وفي التوليد سواء. */
export type Decision = 'generate' | 'skip' | 'regenerate';

export function decide(opts: {
  readonly fileExists: boolean;
  readonly storedHash: string | undefined;
  readonly currentHash: string;
  readonly force: boolean;
}): Decision {
  if (opts.force) return 'regenerate';
  if (!opts.fileExists) return 'generate';
  // **وملفٌّ بلا بصمةٍ محفوظةٍ يُعاد** — **لا نعرف بأيّ إعدادٍ وُلّد**،
  // وصوتٌ من إعدادٍ مجهولٍ في حزمةٍ متجانسةٍ يُسمع نشازا.
  if (opts.storedHash === undefined) return 'regenerate';
  return opts.storedHash === opts.currentHash ? 'skip' : 'regenerate';
}
