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

/** صنفٌ محفوظ — **ومعه متجرُه وسعرُه الآن.** */
export interface FavoriteItem {
  id: string;
  name: string;
  price: number;
  image_thumb_url: string | null;
  merchant_id: string;
  merchant_name: string;
  category_icon: string;
  /** **لا يُطلب اليوم** — أوقفه المتجر أو أُغلق طارئاً. */
  unavailable: boolean;
}

type Api = <T>(path: string, init?: RequestInit) => Promise<T>;

/**
 * useFavorites قائمةُ المفضّلة وقلبُها — **مصدرٌ واحدٌ للصفحة وللزرّ.**
 *
 * **ولا يُنادى الخادمُ لمن لم يدخل**: `‎/my/favorites` يردّ «غيرَ مخوَّل»،
 * **وخطأٌ في السجلّ عند كلّ زائرٍ يُغرق ما يُقرأ.**
 */
export function useFavorites(api: Api, enabled = true) {
  const [ids, setIds] = useState<Set<string> | null>(null);
  const [rows, setRows] = useState<FavoriteItem[] | null>(null);

  const load = useCallback(() => {
    if (!enabled) {
      setIds(new Set());
      setRows([]);
      return;
    }
    api<FavoriteItem[]>("/api/v1/my/favorites")
      .then((r) => {
        const list = r ?? [];
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
      className={`inline-flex shrink-0 items-center justify-center rounded-control border transition-colors ${box} ${
        on
          ? "border-danger/30 bg-danger/10 text-danger"
          : "border-line text-ink-muted hover:border-danger/40 hover:text-danger"
      } ${className}`}
    >
      {/* **والممتلئُ يُقرأ بلمحة** — وقلبان بالحدّ نفسِه لا يفترقان في العين. */}
      <IconHeart size={size === "sm" ? 15 : 18} className={on ? "fill-current" : ""} />
    </button>
  );
}

/**
 * FavoritesPage صفحةُ المفضّلة — **صحونٌ تُضغط فتُفتح في مطبخها.**
 *
 * @param Link رابطُ الإطار — يختلف بين `next/link` وغيره.
 */
export function FavoritesPage({
  api,
  Link,
  enabled = true,
  empty,
}: {
  api: Api;
  Link: ComponentType<{ href: string; className?: string; children: ReactNode }>;
  /** هل دخل صاحبُ الشاشة — **ومن لم يدخل لا مفضّلةَ له.** */
  enabled?: boolean;
  /** ما يُعرض حين لا مفضّلة — **زرُّ تصفّحٍ خيرٌ من جملةٍ يائسة.** */
  empty?: ReactNode;
}) {
  const { rows, has, toggle } = useFavorites(api, enabled);

  if (rows === null) return <LoadingState />;

  return (
    <PageContainer>
      <PageHeader icon={IconHeart} title={F.title} subtitle={F.subtitle} />

      {rows.length === 0 ? (
        <EmptyState icon={IconHeart} title={F.empty} action={empty} />
      ) : (
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {rows.map((f) => (
            <div
              key={f.id}
              className="flex items-center gap-3 rounded-card border border-line bg-surface p-3"
            >
              {/* **والوجهةُ مطبخُه لا الصنفُ وحدَه**: الطلبُ يبدأ من شاشة
                  المتجر — وهناك خياراتُه وكمّيتُه. */}
              <Link
                href={`/m/${f.merchant_id}`}
                className="flex min-w-0 flex-1 items-center gap-3"
              >
                {f.image_thumb_url ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={f.image_thumb_url}
                    alt=""
                    className="h-14 w-14 shrink-0 rounded-control object-cover"
                  />
                ) : (
                  <span className="flex h-14 w-14 shrink-0 items-center justify-center rounded-control bg-primary-light">
                    <CategoryIcon name={f.category_icon} size={18} />
                  </span>
                )}
                <span className="min-w-0 flex-1">
                  <span className="block truncate font-bold">{f.name}</span>
                  {/* **واسمُ المطعم تحته**: «شاورما دجاج» في ثلاثة مطاعم،
                      **ومن رأى اسماً بلا مطعمٍ لا يعرف أيَّها حفظ.** */}
                  <span className="block truncate text-xs text-ink-muted">
                    {f.merchant_name}
                  </span>
                  <span className="mt-1 flex items-center gap-2">
                    {/* **والسعرُ الآن لا يومَ الحفظ** — ومن اكتشف الفرقَ في
                        السلّة اكتشفه في أسوأ لحظة. */}
                    <span className="font-bold text-primary-dark">
                      {fmtNum(f.price)} {m.common.currency}
                    </span>
                    {f.unavailable && <Badge variant="warning">{F.unavailable}</Badge>}
                  </span>
                </span>
              </Link>
              <FavoriteButton itemID={f.id} on={has(f.id)} onToggle={toggle} />
            </div>
          ))}
        </div>
      )}
    </PageContainer>
  );
}
