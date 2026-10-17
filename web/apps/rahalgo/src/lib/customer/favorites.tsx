"use client";

/**
 * **المفضّلةُ مرّةً واحدةً للسوق كلِّه** — القلبُ في البطاقة وفي نافذة
 * الصنف وصفحةُ «المفضلة» تقرأ مصدراً واحداً (`useFavorites` من العُدّة).
 *
 * **والضيفُ يُساق إلى الدخول** — قلبٌ يُضغط فلا يقع شيءٌ يُقرأ عطباً
 * (مرآةُ `onLike` في `MainActivity.kt`).
 */

import { createContext, useCallback, useContext, type ReactNode } from "react";
import { usePathname, useRouter } from "next/navigation";
import { useFavorites } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import type { Item } from "./types";

interface FavState {
  rows: Item[] | null;
  has: (id: string) => boolean;
  toggle: (id: string) => void;
  reload: () => void;
  /** **للضيف** — يُساق إلى الدخول ثمّ يعود. */
  needLogin: (() => void) | undefined;
}

const Ctx = createContext<FavState | null>(null);

export function FavoritesProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth();
  const router = useRouter();
  const path = usePathname() ?? "/shop";
  const fav = useFavorites<Item>(api, !!user);
  const login = useCallback(() => router.push(`/login?next=${encodeURIComponent(path)}`), [router, path]);
  const value: FavState = {
    rows: fav.rows,
    has: fav.has,
    toggle: (id: string) => void fav.toggle(id),
    reload: fav.reload,
    needLogin: user ? undefined : login,
  };
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useFav(): FavState {
  const c = useContext(Ctx);
  if (!c) throw new Error("useFav must be used inside FavoritesProvider");
  return c;
}
