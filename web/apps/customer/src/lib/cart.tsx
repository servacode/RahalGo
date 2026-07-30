"use client";

/** سلة الزبون: متجر واحد في المرة (المعيار في تطبيقات التوصيل) — تُحفظ محلياً. */

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
  merchant_id: string;
  merchant_name: string;
  lines: CartLine[];
}

interface CartState {
  cart: Cart | null;
  count: number;
  add: (merchantID: string, merchantName: string, line: CartLine) => boolean;
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

  // يعيد false إذا كانت السلة لمتجر آخر (الواجهة تسأل قبل الإفراغ)
  function add(merchantID: string, merchantName: string, line: CartLine): boolean {
    if (cart && cart.merchant_id !== merchantID && cart.lines.length > 0) return false;
    const base = cart && cart.merchant_id === merchantID ? cart : null;
    persist({
      merchant_id: merchantID,
      merchant_name: merchantName,
      lines: [...(base?.lines ?? []), line],
    });
    return true;
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
