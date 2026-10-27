"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/stores/[id] — صفحةُ متجر** (قرارُ المالك ٢٠٢٦-١٠-١٨)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **رأسٌ** (الشعارُ والاسمُ ودوامُ اليوم) **ثمّ دوائرُ الأقسام كالسوق،
 * وتحتها أصنافُ القسم المختار وحدَه** — بالبطاقة والنافذة والسلّة نفسِها.
 *
 * **والمغلقُ للتصفّح وحدَه**: أصنافُه تصل `source_closed` فلا «+» عليها،
 * **والنافذةُ تقول السببَ مكانَ زرّ الإضافة.**
 *
 * **و`/store` لوحةُ صاحب المتجر** — وهذه صفحةُ الزبون، فاسمُها `/stores`.
 */

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Alert, Badge, EmptyState, LoadingState, ReloadState, IconPrev, IconStore, SectionRail } from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useBrowseQuery } from "@/lib/customer/context";
import type { Item, StorePage } from "@/lib/customer/types";
import { clock12, fill } from "@/lib/customer/texts";
import { ItemGrid, ItemSheet } from "@/components/customer/items";
import { StoreLogo } from "@/components/customer/stores";

const m = getMessages(defaultLocale);
const ST = m.shop.stores;

/**
 * **دوامُ اليوم بلفظ الزبون** (ملاحظةُ المالك ٢٠٢٦-١٠-١٨) — «مفتوح ٢٤ ساعة» أو «اليوم من … لـ …»
 * أو «مغلق اليوم»، **لا «مفتوح هلّق».**
 */
function todayText(today: StorePage["store"]["today"]): string {
  if (!today || today.open_24h) return ST.open24;
  if (today.closed || !today.open) return ST.closedToday;
  return fill(ST.todayHours, { from: clock12(today.open), to: clock12(today.close) });
}

export default function StorePageView() {
  const params = useParams<{ id: string }>();
  const id = params?.id ?? "";
  const scope = useBrowseQuery("?");
  const [page, setPage] = useState<StorePage | null>(null);
  const [error, setError] = useState("");
  const [open, setOpen] = useState<Item | null>(null);
  const [active, setActive] = useState("");

  const load = useCallback(() => {
    if (!id) return;
    setError("");
    setPage(null);
    api<StorePage>(`/api/v1/public/stores/${id}${scope}`)
      .then((p) => {
        setPage(p);
        setActive(p.sections[0]?.id ?? "");
      })
      .catch((err) => setError(errorText(err)));
  }, [id, scope]);

  useEffect(load, [load]);

  const shown = page?.sections.find((sec) => sec.id === active) ?? page?.sections[0];

  return (
    <div className="flex flex-col gap-4">
      {/* **زرٌّ واضحٌ بسهم رجوع** (ملاحظةُ المالك ٢٠٢٦-١٠-١٨) */}
      <Link
        href="/shop"
        className="inline-flex w-fit items-center gap-2 rounded-full border border-accent px-4 py-2 text-sm font-bold text-accent-text transition-colors hover:bg-accent-tint"
      >
        <IconPrev size={18} />
        {ST.back}
      </Link>

      {error ? (
        <ReloadState label={error} onRetry={load} />
      ) : !page ? (
        <LoadingState variant="list" rows={4} />
      ) : (
        <>
          {/* ── الرأس ── */}
          <header className="flex items-center gap-3">
            <StoreLogo name={page.store.name} logo={page.store.logo_thumb_url ?? page.store.logo_url} className="h-20 w-20" />
            <div className="min-w-0 flex-1 space-y-1">
              <h1 className="heading-page truncate">{page.store.name}</h1>
              {/* **«مفتوح / مسكّر» حبّةً، ودوامُ اليوم سطرٌ تحتها** (مرآةُ `StoreScreen.kt` ٢٠٢٦-١٠-٢٧). */}
              <Badge variant={page.store.open_now ? "success" : "danger"} className="font-bold">
                {page.store.open_now ? ST.pillOpen : ST.pillClosed}
              </Badge>
              <p className="text-xs text-ink-muted">{todayText(page.store.today)}</p>
            </div>
          </header>

          {!page.store.open_now && <Alert tone="warning">{ST.browseOnly}</Alert>}

          {page.sections.length === 0 ? (
            <EmptyState icon={IconStore} title={ST.empty} action={<span className="text-xs text-ink-muted">{ST.emptyHint}</span>} />
          ) : (
            <>
              {/* ── دوائرُ الأقسام كالسوق، وتحتها أصنافُ المختار وحدَه ── (ملاحظةُ المالك ٢٠٢٦-١٠-١٨) */}
              <SectionRail
                items={page.sections.map((sec) => ({
                  id: sec.id,
                  name: sec.name,
                  imageUrl: mediaUrl(sec.image_thumb_url ?? sec.image_url),
                  icon: sec.icon,
                  count: sec.items.length,
                }))}
                activeID={shown?.id ?? ""}
                auto={false}
                onSelect={setActive}
              />
              {shown && <ItemGrid items={shown.items} onOpen={setOpen} storeLink={false} />}
            </>
          )}
        </>
      )}

      <ItemSheet item={open} onClose={() => setOpen(null)} storeLink={false} />
    </div>
  );
}
