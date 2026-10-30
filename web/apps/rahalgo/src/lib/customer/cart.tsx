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
import { isDrinksGroup, type Item, type ModifierGroup, type ModifierOption } from "./types";
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

/**
 * **سطرُ مشروبٍ في أسفل السلّة** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — المشروبُ نفسُه (باسمه) من كلّ الأسطر
 * سطرٌ واحدٌ بعددٍ ومجموعٍ مجموعين، و`refs` أماكنُه بترتيب الأسطر: «+» و«−» لأوّلها.
 */
export interface DrinkRow {
  name: string;
  thumb: string | null;
  qty: number;
  total: number;
  refs: { key: string; optionId: string; qty: number }[];
}

const drinkName = (o: ModifierOption) => o.name.trim();

export function drinkRows(lines: CartLine[]): DrinkRow[] {
  const out = new Map<string, DrinkRow>();
  for (const l of lines) {
    const key = lineKey(l);
    for (const d of l.drinks ?? []) {
      if (d.qty <= 0) continue;
      const name = drinkName(d.option);
      const row = out.get(name) ?? { name: d.option.name, thumb: null, qty: 0, total: 0, refs: [] };
      row.qty += d.qty;
      row.total += d.option.price_delta * d.qty;
      row.thumb = row.thumb ?? d.option.image_thumb_url ?? null;
      row.refs.push({ key, optionId: d.option.id, qty: d.qty });
      out.set(name, row);
    }
  }
  return [...out.values()];
}

/** **مشروباتُ سطرين يُدمجان** — وسطرا ما قبل العدّادات (بلا `drinks`) يبقيان بلاها. */
function joinDrinks(a: { drinks?: DrinkPick[] }, b: { drinks?: DrinkPick[] }): DrinkPick[] | undefined {
  return a.drinks || b.drinks ? mergeDrinks(a.drinks ?? [], b.drinks ?? []) : undefined;
}

/** **أسطرٌ صارت متطابقةً تُدمج** — بعد حذف مشروبٍ منها. */
function mergeSame(lines: CartLine[]): CartLine[] {
  const out: CartLine[] = [];
  for (const l of lines) {
    const k = lineKey(l);
    const at = out.findIndex((x) => lineKey(x) === k);
    // **ومشروباتُ الاثنين تُجمع** — وإلّا ضاع مشروبُ أحدهما.
    const was = at >= 0 ? out[at] : undefined;
    if (was) out[at] = { ...was, qty: was.qty + l.qty, drinks: joinDrinks(was, l) };
    else out.push(l);
  }
  return out;
}

/** **سطرٌ تُخرجه نافذةُ الصنف** — مرآةُ `Cart.Pick`. */
export interface Pick {
  options: ModifierOption[];
  qty: number;
  drinks: DrinkPick[];
}

/** **سقفُ الخيار الواحد في العدّادات** — مرآةُ `OptionSplit.MAX_PER_OPTION`. */
export const MAX_PER_OPTION = 20;

/**
 * **مجموعةُ العدّادات** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — مرآةُ `OptionSplit.group`.
 * مجموعةٌ إلزاميّةٌ واحدةٌ بخيارٍ واحد (الحجم، الوزن، النمرة)، بلا «مشروبات» ولا الاختياريّة؛
 * **ويُختار منها بعدّادٍ جنب كلّ خيار**، وكلُّ خيارٍ سطرٌ في السلّة. و`null` = كما كانت.
 */
export function splitGroup(groups: ModifierGroup[]): ModifierGroup | null {
  const req = groups.filter((g) => !isDrinksGroup(g) && g.min_select > 0);
  const g = req.length === 1 ? req[0] : undefined;
  if (!g) return null;
  return g.min_select === 1 && g.max_select === 1 ? g : null;
}

/** **سطرٌ لكلّ خيارٍ عددُه فوق الصفر** — الإضافاتُ لكلّها، والمشروباتُ لأوّلها وحدَه. */
export function splitPicks(
  g: ModifierGroup,
  counts: Record<string, number>,
  extras: ModifierOption[],
  drinks: DrinkPick[],
): Pick[] {
  const out: Pick[] = [];
  for (const o of g.options) {
    const n = counts[o.id] ?? 0;
    if (n > 0) out.push({ options: [o, ...extras], qty: n, drinks: out.length === 0 ? drinks.filter((d) => d.qty > 0) : [] });
  }
  return out;
}

/** **ثمنُ السطور كلّها** — مرآةُ `OptionSplit.total`. */
export function picksTotal(item: Item, picks: Pick[]): number {
  return picks.reduce((s, p) => s + unitPrice({ item, options: p.options }) * p.qty + drinksTotal(p), 0);
}

/** **ثمنُها بالدولار** لمتجرٍ يعرض به. */
export function picksTotalUsd(baseUsd: number, picks: Pick[]): number {
  return picks.reduce(
    (s, p) =>
      s +
      (baseUsd + p.options.reduce((a, o) => a + (o.price_delta_usd ?? 0), 0)) * p.qty +
      p.drinks.reduce((a, d) => a + (d.option.price_delta_usd ?? 0) * d.qty, 0),
    0,
  );
}

interface CartState {
  lines: CartLine[];
  count: number;
  subtotal: number;
  ready: boolean;
  add: (item: Item, qty?: number, options?: ModifierOption[], drinks?: DrinkPick[]) => void;
  setQty: (key: string, qty: number) => void;
  /** **عددُ مشروبٍ في سطره** — والصفرُ يرفعه وحدَه (قرارُ المالك ٢٠٢٦-١٠-٣٠). */
  setDrinkQty: (key: string, optionId: string, qty: number) => void;
  /** **حذفُ مشروبٍ (باسمه) من كلّ الأسطر** — سلّةُ المهملات في سطر المشروبات أسفلَ السلّة. */
  removeDrink: (name: string) => void;
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

  /* **وسطرٌ صار مطابقاً لآخر يُدمج فيه** — مرآةُ `Cart.setDrinkQty`. */
  const setDrinkQty = useCallback((key: string, optionId: string, qty: number) => {
    setLines((cur) => {
      const at = cur.findIndex((l) => lineKey(l) === key);
      const line = cur[at];
      if (!line) return cur;
      const drinks = (line.drinks ?? [])
        .map((d) => (d.option.id === optionId ? { ...d, qty: Math.min(MAX_DRINK, qty) } : d))
        .filter((d) => d.qty > 0);
      const updated: CartLine = { ...line, drinks };
      const rest = cur.filter((_, i) => i !== at);
      const k = lineKey(updated);
      const same = rest.find((l) => lineKey(l) === k);
      const merged = same
        ? rest.map((l) =>
            l === same ? { ...l, qty: l.qty + updated.qty, drinks: joinDrinks(l, updated) } : l,
          )
        : [...rest.slice(0, at), updated, ...rest.slice(at)];
      writeJSON(KEY, merged);
      return merged;
    });
  }, []);

  const removeDrink = useCallback((name: string) => {
    setLines((cur) => {
      const next = mergeSame(
        cur.map((l) =>
          l.drinks?.some((d) => drinkName(d.option) === name.trim())
            ? { ...l, drinks: l.drinks.filter((d) => drinkName(d.option) !== name.trim()) }
            : l,
        ),
      );
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
      setDrinkQty,
      removeDrink,
      clear,
    }),
    [lines, ready, add, setQty, setDrinkQty, removeDrink, clear],
  );

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useCart(): CartState {
  const c = useContext(Ctx);
  if (!c) throw new Error("useCart must be used inside CartProvider");
  return c;
}
