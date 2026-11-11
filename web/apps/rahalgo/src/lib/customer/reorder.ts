/**
 * **«اطلب مرة تانية»** (قرارُ المالك ٢٠٢٦-١١-١١) — توأمُ `Reorder.kt` في التطبيق.
 *
 * أصنافُ الطلب المُسلَّم تعود إلى السلّة بخياراتها **من كتالوج اليوم لا من لقطة الأمس**: السعرُ والتوفّرُ يحكمهما
 * الحاضر. **وصنفٌ ما عاد متوفراً (أو اختفى أحدُ خياراته) يُترك ويُقال ذلك**، ولا يُخمَّن له بديل.
 */

import { api } from "@/lib/api";
import type { DrinkPick } from "./cart";
import type { Item, ItemDetail, ModifierOption, MyOrder, MyOrderLine } from "./types";

export interface Planned {
  item: Item;
  qty: number;
  options: ModifierOption[];
  drinks: DrinkPick[];
  /** سقط مشروبُه وحدَه. */
  lostDrink: boolean;
}

/** يطابق سطراً قديماً على صنفٍ حاليّ — و`null` إن لم يعد صالحاً. */
export function planLine(line: MyOrderLine, drinkLines: MyOrderLine[], detail: ItemDetail): Planned | null {
  if (!detail.item.available) return null;
  const all = detail.modifiers.flatMap((g) => g.options);
  const options: ModifierOption[] = [];
  for (const o of line.options ?? []) {
    const hit = o.id ? all.find((x) => x.id === o.id) : undefined;
    if (!hit || !hit.available) return null;
    options.push(hit);
  }
  let lostDrink = false;
  const drinks: DrinkPick[] = [];
  for (const d of drinkLines) {
    const hit = d.drink_option_id ? all.find((x) => x.id === d.drink_option_id && x.available) : undefined;
    if (hit) drinks.push({ option: hit, qty: d.qty });
    else lostDrink = true;
  }
  return { item: detail.item, qty: line.qty, options, drinks, lostDrink };
}

export type AddToCart = (item: Item, qty: number, options: ModifierOption[], drinks: DrinkPick[]) => void;

export async function reorderInto(order: MyOrder, add: AddToCart): Promise<{ added: number; skipped: number }> {
  const all = order.items ?? [];
  const lines = all.filter((l) => !l.drink_for && l.menu_item_id && l.qty > 0);
  const details = new Map<string, ItemDetail | null>();
  let added = 0;
  let skipped = 0;
  for (const l of lines) {
    const id = l.menu_item_id as string;
    if (!details.has(id)) {
      details.set(id, await api<ItemDetail>(`/api/v1/public/items/${id}`).catch(() => null));
    }
    const detail = details.get(id);
    const planned = detail ? planLine(l, all.filter((d) => d.drink_for === l.id), detail) : null;
    if (!planned) {
      skipped++;
      continue;
    }
    add(planned.item, planned.qty, planned.options, planned.drinks);
    added++;
    if (planned.lostDrink) skipped++;
  }
  return { added, skipped };
}
