"use client";

/**
 * سلّةُ الزبون — **أصنافٌ بلا مصدر.**
 *
 * # لماذا لم تعد تحمل المتجر
 *
 * الزبونُ لا يرى المتاجر: **يطلب أصنافاً ونحن نعرف من أين نشتريها.** ومعرّفُ
 * متجرٍ محفوظٌ في `localStorage` **يُقرأ بسطرٍ واحدٍ في أدوات المتصفّح** —
 * فالإخفاءُ الذي حرسناه في الشبكة يسقط في ذاكرة الجهاز.
 *
 * # وسقفُ المصادر يُفرض في الخادم
 *
 * كانت السلّةُ ترفض صنفاً من متجرٍ آخر **وهي تعرف المتجرين**. واليومَ لا
 * تعرفهما، **والخادمُ يعرف**: يستنتج المصدرَ من الأصناف ويردّ إن تعدّدت.
 *
 * **وهو الموضعُ الذي لا يُلتفّ عليه**: حارسٌ في المتصفّح يتجاوزه كلُّ من يعرف
 * النقطة، **وحارسٌ في الخادم لا يتجاوزه أحد.**
 */

import { createContext, useContext, useEffect, useState } from "react";

export interface CartLine {
  menu_item_id: string;
  name: string;
  price: number; // للعرض فقط — التسعير الفعلي خادمي
  qty: number;
  note: string;
  option_ids: string[];
  option_names: string[];
  options_delta: number;
}

export interface Cart {
  lines: CartLine[];
}

interface CartState {
  cart: Cart | null;
  count: number;
  add: (line: CartLine) => void;
  setQty: (index: number, qty: number) => void;
  clear: () => void;
}

const CartContext = createContext<CartState | null>(null);
const KEY = "rahalgo_cart";

export function CartProvider({ children }: { children: React.ReactNode }) {
  const [cart, setCart] = useState<Cart | null>(null);

  useEffect(() => {
    try {
      const raw = localStorage.getItem(KEY);
      if (raw) setCart(JSON.parse(raw) as Cart);
    } catch {
      /* تجاهل سلة تالفة */
    }
  }, []);

  function persist(c: Cart | null) {
    setCart(c);
    if (c) localStorage.setItem(KEY, JSON.stringify(c));
    else localStorage.removeItem(KEY);
  }

  function add(line: CartLine) {
    persist({ lines: [...(cart?.lines ?? []), line] });
  }

  function setQty(index: number, qty: number) {
    if (!cart) return;
    const lines = qty <= 0
      ? cart.lines.filter((_, i) => i !== index)
      : cart.lines.map((l, i) => (i === index ? { ...l, qty } : l));
    persist(lines.length ? { ...cart, lines } : null);
  }

  const count = cart?.lines.reduce((n, l) => n + l.qty, 0) ?? 0;

  return (
    <CartContext.Provider value={{ cart, count, add, setQty, clear: () => persist(null) }}>
      {children}
    </CartContext.Provider>
  );
}

export function useCart(): CartState {
  const ctx = useContext(CartContext);
  if (!ctx) throw new Error("useCart must be used within CartProvider");
  return ctx;
}
