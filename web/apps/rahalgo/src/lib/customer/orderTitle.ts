/**
 * **عنوانُ الطلب حين لا اسمَ متجر** (قرارُ المالك ٢٠٢٦-١١-١١) — دالّةٌ صافيةٌ توأمُها `ItemsTitle.kt` في التطبيق.
 *
 * **«+N» عددُ الأصناف الأخرى لا عددُ القطع**: سطرٌ واحدٌ بكميّة ٢ يُقال «برغر × ٢» لا «برغر +١»،
 * وصنفان مختلفان يُقال «برغر +١». **والمشروبُ (`drink_for`) تابعٌ لصنفه فلا يُعدّ صنفاً.**
 */

export interface TitleLine {
  name: string;
  qty: number;
  menu_item_id?: string | null;
  drink_for?: string | null;
}

export interface ItemsTitleParts {
  first: string;
  /** عددُ الأصناف المختلفة الأخرى. */
  others: number;
  /** كميّةُ الصنف الأوّل. */
  qty: number;
}

/** `preview`: معاينةُ المحرّك («أ، ب، ج») حين لا بنود — وفيها أسماءٌ فقط فلا كميّات. */
export function itemsTitleParts(items: TitleLine[] | null | undefined, preview: string | null | undefined, sep: string): ItemsTitleParts {
  const lines = (items ?? []).filter((l) => !l.drink_for && l.name.trim());
  const head = lines[0];
  if (head) {
    const key = (l: TitleLine) => l.menu_item_id || l.name.trim();
    const kinds = new Set(lines.map(key));
    const qty = lines.filter((l) => key(l) === key(head)).reduce((s, l) => s + l.qty, 0);
    return { first: head.name.trim(), others: kinds.size - 1, qty };
  }
  const parts = (preview ?? "").split(sep).map((p) => p.trim()).filter(Boolean);
  return { first: parts[0] ?? "", others: Math.max(parts.length - 1, 0), qty: 1 };
}

export function itemsTitleText(p: ItemsTitleParts, fmt: (n: number) => string): string {
  if (!p.first) return "";
  if (p.others > 0) return `${p.first} +${fmt(p.others)}`;
  if (p.qty > 1) return `${p.first} × ${fmt(p.qty)}`;
  return p.first;
}
