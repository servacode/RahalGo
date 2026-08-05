"use client";

/**
 * **المفضّلة** — متاجرُ يعود إليها صاحبُها.
 *
 * # لماذا الآن
 *
 * كانت مبنيّةً في الخادم بالكامل (`GET /my/favorites` و`POST
 * /my/favorites/{id}`) **ولا بابَ لها في الواجهة** — صفرُ نداءٍ من التطبيق
 * كلِّه. **وميزةٌ لا يصل إليها أحدٌ ميزةٌ غيرُ موجودة.**
 *
 * # ولماذا مكوّنٌ واحدٌ لا ثلاثة
 *
 * القلبُ في صفحة المتجر، والقائمةُ في صفحتها، **والاثنان يقرآن الشيءَ نفسَه.**
 * ولو قرأ كلٌّ منهما وحدَه **لَافترقا**: يُضاف متجرٌ من صفحته فلا يظهر في
 * القائمة حتّى تُحدَّث الصفحةُ كلُّها.
 *
 * # والقلبُ ينقلب قبل أن يردّ الخادم
 *
 * **ضغطةٌ لا يظهر أثرُها فوراً تُضغط مرّتين** — فيُضاف المتجرُ ثمّ يُحذف،
 * **ويظنّ صاحبُه أنّ الزرّ لا يعمل.** فيُقلَب في الشاشة أوّلاً، **ويُردّ إن
 * فشل النداء** — لا يُترك على كذبة.
 */

import { useCallback, useEffect, useState } from "react";
import type { ComponentType, ReactNode } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { CategoryIcon } from "./CategoryIcon";
import { PageContainer, PageHeader, EmptyState, LoadingState } from "./layout";
import { IconHeart, IconStore } from "./icons";

const m = getMessages(defaultLocale);
const F = m.customer.favorites;

export interface FavoriteMerchant {
  id: string;
  name: string;
  category_icon: string;
  logo_thumb_url: string | null;
  emergency_closed: boolean;
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
  const [rows, setRows] = useState<FavoriteMerchant[] | null>(null);

  const load = useCallback(() => {
    if (!enabled) {
      setIds(new Set());
      setRows([]);
      return;
    }
    api<FavoriteMerchant[]>("/api/v1/my/favorites")
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

  /** toggle يقلب الحالَ في الشاشة ثمّ يُثبته — **ويردّه إن تعذّر.** */
  const toggle = useCallback(
    async (mer: FavoriteMerchant) => {
      const was = ids?.has(mer.id) ?? false;
      setIds((s) => {
        const next = new Set(s ?? []);
        if (was) next.delete(mer.id);
        else next.add(mer.id);
        return next;
      });
      setRows((s) => (was ? (s ?? []).filter((x) => x.id !== mer.id) : [mer, ...(s ?? [])]));
      try {
        // **والخادمُ يقول ما استقرّ عليه** — لا ما ظنّته الشاشة.
        const r = await api<{ favorite: boolean }>(`/api/v1/my/favorites/${mer.id}`, {
          method: "POST",
        });
        if (r.favorite === was) load();
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
 */
export function FavoriteButton({
  merchant,
  on,
  onToggle,
  onRequireLogin,
  className = "",
}: {
  merchant: FavoriteMerchant;
  on: boolean;
  onToggle: (mer: FavoriteMerchant) => void;
  onRequireLogin?: () => void;
  className?: string;
}) {
  const label = on ? F.remove : F.add;
  return (
    <button
      type="button"
      title={label}
      aria-label={label}
      aria-pressed={on}
      onClick={() => (onRequireLogin ? onRequireLogin() : onToggle(merchant))}
      className={`inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-control border transition-colors ${
        on
          ? "border-danger/30 bg-danger/10 text-danger"
          : "border-line text-ink-muted hover:border-danger/40 hover:text-danger"
      } ${className}`}
    >
      {/* **والممتلئُ يُقرأ بلمحة** — وقلبان بالحدّ نفسِه لا يفترقان في العين. */}
      <IconHeart size={18} className={on ? "fill-current" : ""} />
    </button>
  );
}

/**
 * FavoritesPage صفحةُ المفضّلة — **قائمةٌ تُضغط لا عرضٌ للأسماء.**
 *
 * @param LinkComp رابطُ الإطار — يختلف بين `next/link` وغيره.
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
              <Link href={`/m/${f.id}`} className="flex min-w-0 flex-1 items-center gap-3">
                {f.logo_thumb_url ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={f.logo_thumb_url}
                    alt=""
                    className="h-12 w-12 shrink-0 rounded-control object-cover"
                  />
                ) : (
                  <span className="flex h-12 w-12 shrink-0 items-center justify-center rounded-control bg-primary-light">
                    <CategoryIcon name={f.category_icon} size={18} />
                  </span>
                )}
                <span className="min-w-0 flex-1">
                  <span className="block truncate font-medium">{f.name}</span>
                  {/* **ومغلقٌ يُقال هنا لا بعد الضغط** — من فتح متجراً مغلقاً
                      وعاد يعدّها عطباً في المنصة. */}
                  {f.emergency_closed && (
                    <span className="block text-xs text-warning">{m.site.closed}</span>
                  )}
                </span>
              </Link>
              <FavoriteButton merchant={f} on={has(f.id)} onToggle={toggle} />
            </div>
          ))}
        </div>
      )}
    </PageContainer>
  );
}
