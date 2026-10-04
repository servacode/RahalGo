"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **صلاحيّةُ الزرّ من جدول المحرّك نفسِه** (قرارُ المالك ٢٠٢٦-١٠-٠٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كانت أزرارُ ملفّ الحساب تفحص قدرةً والمحرّكُ يطلب أخرى** — «الثقة والسلامة»
 * لا ترى زرَّ الإيقاف وهي صاحبتُه، ومن معه «إدارة الأدوار» يراه ويُردّ.
 *
 * **فالسؤالُ هنا سؤالُ المحرّك**: «أيُّ قدرةٍ تفتح هذا الباب؟» — من الملفّ المولَّد
 * (`adminPolicy.gen.ts`، يُسقط البناءَ إن شاخ). **والمطابقةُ مطابقتُه**: الأكثرُ
 * حروفاً ثابتةً يغلب، والطريقةُ المحدّدةُ تغلب الفارغة.
 */

import { useCallback } from "react";
import { ADMIN_POLICY } from "./adminPolicy.gen";
import { useAuth } from "./auth";

function segs(p: string): string[] {
  return p.replace(/^\/+|\/+$/g, "").split("/");
}

const isParam = (s: string) => s.startsWith("{") && s.endsWith("}");

/** capFor **القدرةُ التي يطلبها المحرّكُ لهذا الباب** — بنمطه (`/users/{id}`). */
export function capFor(method: string, pattern: string): string | null {
  const want = segs(pattern);
  let best: string | null = null;
  let bestScore = -1;
  for (const [m, p, cap] of ADMIN_POLICY) {
    if (m !== "" && m !== method) continue;
    const have = segs(p);
    if (have.length !== want.length) continue;
    let ok = true;
    let literal = 0;
    for (let i = 0; i < have.length; i++) {
      const h = have[i] ?? "";
      const w = want[i] ?? "";
      if (isParam(h)) continue;
      if (isParam(w) || h !== w) {
        ok = false;
        break;
      }
      literal++;
    }
    if (!ok) continue;
    const score = literal * 2 + (m !== "" ? 1 : 0);
    if (score > bestScore) {
      best = cap;
      bestScore = score;
    }
  }
  return best;
}

/** useCanCall **أيملك المستخدمُ هذا الباب؟** — `canCall("PATCH", "/users/{id}")`. */
export function useCanCall(): (method: string, pattern: string) => boolean {
  const { can } = useAuth();
  return useCallback(
    (method: string, pattern: string) => {
      const cap = capFor(method, pattern);
      return cap !== null && can(cap);
    },
    [can],
  );
}
