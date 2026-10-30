"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **سلّةُ الزبون — في المتصفّح، ومرآةُ `Cart.kt`**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **سطرٌ = صنفٌ بخياراته** — وصنفٌ واحدٌ بحجمين سطران لا سطر.
 *
 * **والسعرُ هنا للعرض وحدَه**: التسعيرةُ تُطلب من المحرّك قبل الإرسال
 * (`/public/quote`)، **ويُقارَن ما عُرض بما قيل** فيُسأل الزبونُ إن تبدّل.
 */

import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import type { Item, ModifierOption } from "./types";
import { readJSON, writeJSON } from "./storage";

/** **مشروبٌ وعددُه** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — لا يُضرب بعدد الصنف. مرآةُ `Cart.DrinkPick`. */
export interface DrinkPick {
  option: ModifierOption;
  qty: number;
}

export interface CartLine {
  item: Item;
  qty: number;
  options: ModifierOption[];
  /**
   * **المشروباتُ بأعدادها.** وغيابُه = سطرٌ حُفظ قبل عدّاد المشروبات: مشروبُه (إن كان)
   * داخل `options` ويُضرب بعدد الصنف، **فلا يُرسَل له `option_qty`** ليبقى سعرُه كما رآه.
   */
  drinks?: DrinkPick[];
}

/** **سقفُ المشروب الواحد في السطر** — مرآةُ `maxDrinkQty` في المحرّك. */
export const MAX_DRINK = 20;

const KEY = "rahalgo_cart_v1";

export function lineKey(l: { item: Item; options: ModifierOption[]; drinks?: DrinkPick[] }): string {
  const base = l.item.id + "|" + l.options.map((o) => o.id).sort().join(",");
  const d = l.drinks ?? [];
  return d.length === 0 ? base : base + "|" + d.map((x) => x.option.id + "x" + x.qty).sort().join(",");
}

export function unitPrice(l: { item: Item; options: ModifierOption[] }): number {
  return l.item.price + l.options.reduce((s, o) => s + o.price_delta, 0);
}

/** **ثمنُ المشروبات** — كلٌّ بسعره وعدده. */
export function drinksTotal(l: { drinks?: DrinkPick[] }): number {
  return (l.drinks ?? []).reduce((s, d) => s + d.option.price_delta * d.qty, 0);
}

/** **ثمنُ السطر كاملاً** — الصنفُ بعدده وخياراته، ثمّ المشروباتُ بأعدادها. */
export function lineTotal(l: CartLine): number {
  return unitPrice(l) * l.qty + drinksTotal(l);
}

/**
 * **ما يُرسَل للمحرّك** — الصنفُ وعددُه وخياراتُه، ولا سعر.
 * **و`option_qty` دائماً ولو `{}`** إلّا لسطرٍ قديم (انظر `CartLine.drinks`).
 */
export function linePayload(l: CartLine) {
  const base = { menu_item_id: l.item.id, qty: l.qty, option_ids: l.options.map((o) => o.id) };
  if (!l.drinks) return base;
  const option_qty: Record<string, number> = {};
  for (const d of l.drinks) if (d.qty > 0) option_qty[d.option.id] = d.qty;
  return { ...base, option_qty };
}

/** **يجمع مشروبين بالمعرّف نفسِه** — والسقفُ ٢٠. */
function mergeDrinks(a: DrinkPick[], b: DrinkPick[]): DrinkPick[] {
  const out = new Map<string, DrinkPick>();
  for (const d of [...a, ...b]) {
    const was = out.get(d.option.id);
    out.set(d.option.id, was ? { ...was, qty: Math.min(MAX_DRINK, was.qty + d.qty) } : d);
  }
  return [...out.values()];
}

interface CartState {
  lines: CartLine[];
  count: number;
  subtotal: number;
  ready: boolean;
  add: (item: Item, qty?: number, options?: ModifierOption[], drinks?: DrinkPick[]) => void;
  setQty: (key: string, qty: number) => void;
  clear: () => void;
  /** **متجرُ السلّة** — ومنه يُعرف أنّ صنفاً جديداً من متجرٍ آخر. */
  merchantId: string;
}

const Ctx = createContext<CartState | null>(null);

export function CartProvider({ children }: { children: ReactNode }) {
  const [lines, setLines] = useState<CartLine[]>([]);
  const [ready, setReady] = useState(false);

  /* **وتُقرأ بعد التركيب** — الخادمُ لا تخزينَ له، **ورسمٌ أوّلُ يختلف
     بين الخادم والمتصفّح يُسقط الإحياء.** */
  useEffect(() => {
    const saved = readJSON<CartLine[]>(KEY, []);
    setLines(Array.isArray(saved) ? saved.filter((l) => l && l.item && l.qty > 0) : []);
    setReady(true);
  }, []);

  const save = useCallback((next: CartLine[]) => {
    setLines(next);
    writeJSON(KEY, next);
  }, []);

  const add = useCallback(
    (item: Item, qty = 1, options: ModifierOption[] = [], drinks: DrinkPick[] = []) => {
      setLines((cur) => {
        const picked = drinks.filter((d) => d.qty > 0);
        const k = lineKey({ item, options, drinks: picked });
        const at = cur.findIndex((l) => lineKey(l) === k);
        const next =
          at >= 0
            ? cur.map((l, i) =>
                i === at ? { ...l, qty: l.qty + qty, drinks: mergeDrinks(l.drinks ?? [], picked) } : l,
              )
            : [...cur, { item, qty, options, drinks: picked }];
        writeJSON(KEY, next);
        return next;
      });
    },
    [],
  );

  const setQty = useCallback((key: string, qty: number) => {
    setLines((cur) => {
      const next =
        qty <= 0 ? cur.filter((l) => lineKey(l) !== key) : cur.map((l) => (lineKey(l) === key ? { ...l, qty } : l));
      writeJSON(KEY, next);
      return next;
    });
  }, []);

  const clear = useCallback(() => save([]), [save]);

  const value = useMemo<CartState>(
    () => ({
      lines,
      ready,
      count: lines.reduce((s, l) => s + l.qty, 0),
      subtotal: lines.reduce((s, l) => s + lineTotal(l), 0),
      merchantId: lines[0]?.item.merchant_id ?? "",
      add,
      setQty,
      clear,
    }),
    [lines, ready, add, setQty, clear],
  );

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useCart(): CartState {
  const c = useContext(Ctx);
  if (!c) throw new Error("useCart must be used inside CartProvider");
  return c;
}
