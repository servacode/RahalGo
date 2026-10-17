"use client";

/**
 * **خبرٌ قصيرٌ واحدٌ للسوق كلِّه** — «أُضيف إلى السلّة»، «وصل طلبك».
 *
 * **ومكدّسُ الإخبار من العُدّة** (`useToast`) — يُرفع هنا إلى السياق
 * فتناديه أيُّ شاشةٍ بلا أن تحمل مكدّسها.
 */

import { createContext, useContext, type ReactNode } from "react";
import { useToast, type AlertTone } from "@rahalgo/ui";

type Push = (text: string, tone?: AlertTone) => void;

const Ctx = createContext<Push>(() => undefined);

export function ShopToastProvider({ children }: { children: ReactNode }) {
  const { push, Toaster } = useToast();
  return (
    <Ctx.Provider value={push}>
      {children}
      <Toaster />
    </Ctx.Provider>
  );
}

export function useShopToast(): Push {
  return useContext(Ctx);
}
