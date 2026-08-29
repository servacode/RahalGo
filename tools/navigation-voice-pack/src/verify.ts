/**
 * ══════════════════════════════════════════════════════════════════════
 * **التحقّقُ من الملفّ الصوتيّ — ولا يُعدّ ناجحاً بلا فحص**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مواصفةُ المالك ٢٠٢٦-٠٨-٢٣، البند ٢٢.)
 *
 * **ونجاحُ النداء وحدَه ليس نجاحاً** — نصُّ المالك. **وردُّ خطأٍ بصيغة
 * JSON يُحفظ باسم `.mp3` يمرّ من كلّ فحصٍ يسأل «أموجودٌ الملفّ؟».**
 *
 * # وكيف يُعرف الـMP3
 *
 * **إمّا رأسُ `ID3`** (وسمٌ في أوّله)، **أو إطارُ MPEG** الذي يبدأ
 * بـ`0xFF` وثلاثِ بتّاتٍ مضبوطةٍ بعده. **وأحدُهما يكفي.**
 *
 * **ولا يُكتفى بالامتداد** — **والامتدادُ اسمٌ نكتبه نحن.**
 */

export interface VerifyResult {
  readonly ok: boolean;
  readonly why: string;
  readonly bytes: number;
}

/** **أدنى حجمٍ معقول** — **وملفٌّ من مئتي بايتٍ ليس كلاماً منطوقاً.** */
export const MIN_BYTES = 1024;

export function verifyMp3(data: Uint8Array): VerifyResult {
  const bytes = data.byteLength;
  if (bytes === 0) return { ok: false, why: 'ملفٌّ فارغ', bytes };
  if (bytes < MIN_BYTES) return { ok: false, why: `أصغرُ من الحدّ (${bytes} بايت)`, bytes };

  // **وردُّ خطأٍ نصّيٌّ يبدأ بقوسٍ أو بمسافة** — يُمسك قبل أيّ شيء.
  const head = new TextDecoder('utf8', { fatal: false }).decode(data.subarray(0, 16)).trim();
  if (head.startsWith('{') || head.startsWith('<')) {
    return { ok: false, why: 'ردُّ خطأٍ لا صوت', bytes };
  }

  const b0 = data[0] ?? 0;
  const b1 = data[1] ?? 0;
  const b2 = data[2] ?? 0;
  const hasId3 = b0 === 0x49 && b1 === 0x44 && b2 === 0x33; // "ID3"
  if (hasId3) return { ok: true, why: 'رأسُ ID3', bytes };

  // **وإطارُ MPEG**: أحدَ عشرَ بتّاً مضبوطةً ثمّ نسخةٌ وطبقةٌ صالحتان.
  for (let i = 0; i < Math.min(data.byteLength - 1, 2048); i++) {
    const a = data[i] ?? 0;
    const b = data[i + 1] ?? 0;
    if (a === 0xff && (b & 0xe0) === 0xe0) {
      const version = (b >> 3) & 0x03;
      const layer = (b >> 1) & 0x03;
      // **والنسخةُ ١ والطبقةُ ٠ محجوزتان** — فوجودُهما يعني أنّها ضجيج.
      if (version !== 0x01 && layer !== 0x00) {
        return { ok: true, why: 'إطارُ MPEG', bytes };
      }
    }
  }
  return { ok: false, why: 'لا رأسَ MP3 صالحاً', bytes };
}
