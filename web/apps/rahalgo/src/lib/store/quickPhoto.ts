/**
 * **ترتيب التصوير السريع** (قرار المالك ٢٠٢٦-١١-٠٦) — مرآة `QuickPhotoQueue.of` في
 * `QuickPhotoQueue.kt`: ما تخطّاه المتجر في هذه الجلسة يذهب إلى الآخر، وما صورته قيد
 * الرفع لا يظهر ثانيةً، والباقي بترتيب الخادم كما جاء (القسم ثم الاسم ثم المعرّف).
 */
export interface NoPhotoItem {
  id: string;
  name: string;
  section_name: string;
  merchant_price: number;
  merchant_price_usd: number | null;
  available: boolean;
}

export function quickPhotoOrder<T extends { id: string }>(
  items: T[],
  skipped: ReadonlySet<string> = new Set(),
  queuedForUpload: ReadonlySet<string> = new Set(),
): T[] {
  const open = items.filter((i) => !queuedForUpload.has(i.id));
  return [...open.filter((i) => !skipped.has(i.id)), ...open.filter((i) => skipped.has(i.id))];
}
