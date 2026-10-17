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

export interface CartLine {
  item: Item;
  qty: number;
  options: ModifierOption[];
}

const KEY = "rahalgo_cart_v1";

export function lineKey(l: { item: Item; options: ModifierOption[] }): string {
  return l.item.id + "|" + l.options.map((o) => o.id).sort().join(",");
}

export function unitPrice(l: { item: Item; options: ModifierOption[] }): number {
  return l.item.price + l.options.reduce((s, o) => s + o.price_delta, 0);
}

/** **ما يُرسَل للمحرّك** — الصنفُ وعددُه وخياراتُه، ولا سعر. */
export function linePayload(l: CartLine) {
  return { menu_item_id: l.item.id, qty: l.qty, option_ids: l.options.map((o) => o.id) };
}

interface CartState {
  lines: CartLine[];
  count: number;
  subtotal: number;
  ready: boolean;
  add: (item: Item, qty?: number, options?: ModifierOption[]) => void;
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
    (item: Item, qty = 1, options: ModifierOption[] = []) => {
      setLines((cur) => {
        const k = lineKey({ item, options });
        const at = cur.findIndex((l) => lineKey(l) === k);
        const next =
          at >= 0
            ? cur.map((l, i) => (i === at ? { ...l, qty: l.qty + qty } : l))
            : [...cur, { item, qty, options }];
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
      subtotal: lines.reduce((s, l) => s + unitPrice(l) * l.qty, 0),
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
