/**
 * ══════════════════════════════════════════════════════════════════════
 * **تصحيحاتُ النطق — مركزيّةٌ لا مبعثرة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مواصفةُ المالك ٢٠٢٦-٠٨-٢٣، البند ٢٤.)
 *
 * **إن قال المالكُ إنّ كلمةً تُنطق خطأً** فلا تُبدَّل الجملُ كلُّها
 * عشوائيّاً، **ولا يُلجأ إلى التشكيل الكامل لأجل كلمةٍ واحدة.**
 *
 * # وترتيبُ العلاج
 *
 *	١ · تصحيحُ `ttsText` في المدوّنة نفسِها
 *	٢ · تشكيلٌ أدنى في هذا الملفّ
 *	٣ · بديلٌ لفظيٌّ عند الحاجة
 *	٤ · قاموسُ نطق ElevenLabs — **إن ثبت أنّه مدعومٌ للنموذج المستعمل**
 *
 * **ولا يُفترض أنّ الفونيمات تعمل في كلّ النماذج** — تُقاس أوّلاً.
 *
 * # وأثرُه في البصمة
 *
 * **التصحيحُ يبدّل `ttsText` فتتبدّل بصمتُه** — **فتُعاد الملفّاتُ
 * المتأثّرةُ وحدَها** ويبقى الباقي كما هو. وذاك مرادُ البند ١٩.
 */

import { readFileSync, existsSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

export interface Override {
  /** **ما يُكتب في المدوّنة** — ويُطابَق حرفيّاً. */
  readonly from: string;
  /** **ما يُرسَل إلى المحرّك بدلاً منه.** */
  readonly to: string;
  /** **ولماذا** — **وتصحيحٌ بلا سببٍ يُعاد بعد شهرٍ لأنّه بدا خطأً.** */
  readonly why: string;
}

const HERE = dirname(fileURLToPath(import.meta.url));
const FILE = join(HERE, '..', 'data', 'pronunciation-overrides.json');

let cache: readonly Override[] | null = null;

export function loadOverrides(): readonly Override[] {
  if (cache !== null) return cache;
  if (!existsSync(FILE)) {
    cache = [];
    return cache;
  }
  const raw: unknown = JSON.parse(readFileSync(FILE, 'utf8'));
  if (!Array.isArray(raw)) throw new Error('ملفُّ التصحيحات ليس قائمة');
  const list: Override[] = [];
  for (const item of raw) {
    if (
      typeof item !== 'object' ||
      item === null ||
      typeof (item as Override).from !== 'string' ||
      typeof (item as Override).to !== 'string'
    ) {
      throw new Error('تصحيحٌ ناقصُ الحقول');
    }
    const o = item as Override;
    if (o.from.trim() === '') throw new Error('تصحيحٌ بمصدرٍ فارغ');
    list.push({ from: o.from, to: o.to, why: o.why ?? '' });
  }
  cache = list;
  return cache;
}

/** **للاختبار وحدَه** — يُفرغ الذاكرةَ فتُقرأ من جديد. */
export function resetOverridesCache(): void {
  cache = null;
}

/**
 * **يطبّق التصحيحاتِ على نصٍّ منطوق.**
 *
 * **والأطولُ أوّلاً** — **وإلّا ابتلع تصحيحُ «المخرج» تصحيحَ «المخرج
 * الثاني»** فخرج نصفُ الجملة مصحَّحاً ونصفُها لا.
 */
export function applyOverrides(text: string): string {
  const list = [...loadOverrides()].sort((a, b) => b.from.length - a.from.length);
  let out = text;
  for (const o of list) out = out.split(o.from).join(o.to);
  return out;
}
