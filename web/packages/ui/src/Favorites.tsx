"use client";

/**
 * **المفضّلة** — أصنافٌ يعود إليها صاحبُها.
 *
 * # على الصنف لا على المتجر
 *
 * **الزبونُ لا يشتهي متجراً، يشتهي صحناً.** ومن أحبّ شاورما مطعمٍ بعينه لا
 * يريد قائمتَه كلَّها — يريد ذلك الصحن، **وبينه وبينه ثلاثُ ضغطاتٍ في كلّ
 * مرّة**: المتجرُ ثمّ القسمُ ثمّ الصنف. **ومفضّلةُ المتاجر تختصر واحدةً من
 * ثلاث.** (قرارُ المالك ٢٠٢٦-٠٨-٠٥.)
 *
 * # ولماذا مكوّنٌ واحدٌ لا ثلاثة
 *
 * القلبُ في بطاقة الصنف، والقائمةُ في صفحتها، **والاثنان يقرآن الشيءَ نفسَه.**
 * ولو قرأ كلٌّ منهما وحدَه **لَافترقا**: يُحفظ صنفٌ من شاشة المتجر فلا يظهر في
 * القائمة حتّى تُحدَّث الصفحةُ كلُّها.
 *
 * # والقلبُ ينقلب قبل أن يردّ الخادم
 *
 * **ضغطةٌ لا يظهر أثرُها فوراً تُضغط مرّتين** — فيُحفظ الصنفُ ثمّ يُحذف،
 * **ويظنّ صاحبُه أنّ الزرّ لا يعمل.** فيُقلَب في الشاشة أوّلاً، **ويُردّ إن
 * فشل النداء** — لا يُترك على كذبة.
 */

import { useCallback, useEffect, useState } from "react";
import type { ComponentType, ReactNode } from "react";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import { CategoryIcon } from "./CategoryIcon";
import { Badge } from "./components";
import { PageContainer, PageHeader, EmptyState, LoadingState } from "./layout";
import { IconHeart } from "./icons";

const m = getMessages(defaultLocale);
const F = m.customer.favorites;

/**
 * معرّفاتُ ما حُفظ — **والبطاقةُ تُرسم في التطبيق لا هنا.**
 *
 * **صفحةُ المفضّلة صارت شبكةَ التصفّح نفسَها**: بطاقةٌ تُضغط فتُفتح في نافذةٍ
 * منبثقة، **لا رابطٌ ينقل إلى صفحةٍ ثانية.** (قرارُ المالك ٢٠٢٦-٠٨-٠٥:
 * «اتّفقنا نافذةٌ منبثقةٌ نفسَ نظام الموقع الصفحة الرئيسيّة».)
 *
 * **وبطاقةُ الصنف تعيش في تطبيق الزبون** — تفتح `ItemClient` وتنادي `api` —
 * **فلا تصعد إلى الحزمة المشتركة.** وهذا الملفُّ يحمل الحالَ وحدَها.
 */
type Api = <T>(path: string, init?: RequestInit) => Promise<T>;

/**
 * useFavorites قائمةُ المفضّلة وقلبُها — **مصدرٌ واحدٌ للصفحة وللزرّ.**
 *
 * **ولا يُنادى الخادمُ لمن لم يدخل**: `‎/my/favorites` يردّ «غيرَ مخوَّل»،
 * **وخطأٌ في السجلّ عند كلّ زائرٍ يُغرق ما يُقرأ.**
 */
export function useFavorites<T extends { id: string }>(api: Api, enabled = true) {
  const [ids, setIds] = useState<Set<string> | null>(null);
  const [rows, setRows] = useState<T[] | null>(null);

  const load = useCallback(() => {
    if (!enabled) {
      setIds(new Set());
      setRows([]);
      return;
    }
    // **والردُّ ملفوفٌ في `items`** — لأنّه استعلامُ التصفّح نفسُه
    // (`favoritesSelect`)، **وشكلُ الصنف واحدٌ في كلّ شاشة.**
    api<{ items: T[] }>("/api/v1/my/favorites")
      .then((r) => {
        const list = r.items ?? [];
        setRows(list);
        setIds(new Set(list.map((x) => x.id)));
      })
      .catch(() => {
        setRows([]);
        setIds(new Set());
      });
  }, [api, enabled]);

  useEffect(load, [load]);

  const has = useCallback((id: string) => ids?.has(id) ?? false, [ids]);

  /**
   * toggle يقلب الحالَ في الشاشة ثمّ يُثبته — **ويردّه إن تعذّر.**
   *
   * **ويكفيه المعرّف**: من ضغط القلبَ في بطاقةٍ لا يملك تفاصيلَ الصنف كلَّها،
   * **والقائمةُ تُعاد قراءتُها بعد الإضافة** — أرخصُ من حمل البطاقة كلِّها
   * في كلّ نداء.
   */
  const toggle = useCallback(
    async (itemID: string) => {
      const was = ids?.has(itemID) ?? false;
      setIds((s) => {
        const next = new Set(s ?? []);
        if (was) next.delete(itemID);
        else next.add(itemID);
        return next;
      });
      if (was) setRows((s) => (s ?? []).filter((x) => x.id !== itemID));
      try {
        const r = await api<{ favorite: boolean }>(`/api/v1/my/favorites/${itemID}`, {
          method: "POST",
        });
        // **والخادمُ يقول ما استقرّ عليه** — لا ما ظنّته الشاشة.
        if (r.favorite === was || r.favorite) load();
      } catch {
        load();
      }
    },
    [api, ids, load],
  );

  return { ids, rows, has, toggle, reload: load };
}

/**
 * FavoriteButton قلبٌ يُضغط — **ومن لم يدخل يُساق إلى الدخول لا يُمنع صامتاً.**
 *
 * **وزرٌّ لا يفعل شيئاً أسوأُ من زرٍّ غائب**: من ضغطه ولم يقع شيءٌ يظنّ العطبَ
 * في المنصة، **ومن سيق إلى الدخول فهم أنّ الميزةَ له إن دخل.**
 *
 * **ويوقف الضغطةَ عن أن تصعد**: القلبُ داخل بطاقةٍ كلُّها زرّ، **فمن أراد
 * الحفظَ لَفُتحت له نافذةُ الطلب** — ضغطةٌ تفعل شيئين ليست ضغطةً واحدة.
 */
export function FavoriteButton({
  itemID,
  on,
  onToggle,
  onRequireLogin,
  size = "md",
  className = "",
}: {
  itemID: string;
  on: boolean;
  onToggle: (itemID: string) => void;
  onRequireLogin?: () => void;
  /** `sm` لبطاقة القائمة و`md` لشاشة الصنف. */
  size?: "sm" | "md";
  className?: string;
}) {
  const label = on ? F.remove : F.add;
  const box = size === "sm" ? "h-8 w-8" : "h-10 w-10";
  return (
    <button
      type="button"
      title={label}
      aria-label={label}
      aria-pressed={on}
      onClick={(e) => {
        e.preventDefault();
        e.stopPropagation();
        if (onRequireLogin) onRequireLogin();
        else onToggle(itemID);
      }}
      /* **ومساحةُ اللمس تتّسع ولا يكبر الرسم** — قلبٌ بأربعةٍ وأربعين في
         بطاقةٍ صغيرةٍ يزاحم اسمَ الصنف. (انظر `taparea` في الثيم.) */
      className={`taparea inline-flex shrink-0 items-center justify-center rounded-control border transition-colors ${box} ${
        on
          ? "border-danger-edge bg-danger-tint text-danger"
          : "border-line text-ink-muted hover:border-danger-edge hover:text-danger"
      } ${className}`}
    >
      {/* **والممتلئُ يُقرأ بلمحة** — وقلبان بالحدّ نفسِه لا يفترقان في العين. */}
      <IconHeart size={size === "sm" ? 15 : 18} className={on ? "fill-current" : ""} />
    </button>
  );
}
