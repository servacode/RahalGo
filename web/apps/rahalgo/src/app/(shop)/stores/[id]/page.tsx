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
import { useCallback, useEffect, useRef, useState } from "react";
import { useParams } from "next/navigation";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Alert, EmptyState, LoadingState, ReloadState, IconClose, IconPrev, IconSearch, IconStore, SectionRail } from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useBrowseQuery } from "@/lib/customer/context";
import type { Item, StoreItemsPage, StoreSummary } from "@/lib/customer/types";
import { clock12, fill } from "@/lib/customer/texts";
import { ItemGrid, ItemSheet } from "@/components/customer/items";
import { StoreLogo } from "@/components/customer/stores";

const m = getMessages(defaultLocale);
const ST = m.shop.stores;

/**
 * **دوامُ اليوم بلفظ الزبون** (ملاحظةُ المالك ٢٠٢٦-١٠-١٨) — «مفتوح ٢٤ ساعة» أو «اليوم من … لـ …»
 * أو «مغلق اليوم»، **لا «مفتوح هلّق».**
 */
function todayText(today: StoreSummary["store"]["today"]): string {
  if (!today || today.open_24h) return ST.open24;
  if (today.closed || !today.open) return ST.closedToday;
  return fill(ST.todayHours, { from: clock12(today.open), to: clock12(today.close) });
}

/** **Items per page** — mirror of `StoreList` in the customer app (owner 2026-11-07). */
const PAGE = 40;

export default function StorePageView() {
  const params = useParams<{ id: string }>();
  const id = params?.id ?? "";
  const scope = useBrowseQuery("?");
  const scopeAmp = useBrowseQuery("&");
  const [page, setPage] = useState<StoreSummary | null>(null);
  const [error, setError] = useState("");
  const [open, setOpen] = useState<Item | null>(null);
  const [active, setActive] = useState("");

  // **The head first: header and section counts** (owner 2026-11-07: «ما يصير يكون بطيء») —
  // then the picked section page by page, not a whole mall in one answer.
  const load = useCallback(() => {
    if (!id) return;
    setError("");
    setPage(null);
    api<StoreSummary>(`/api/v1/public/stores/${id}/summary${scope}`)
      .then((p) => {
        setPage(p);
        setActive(p.sections.find((x) => x.count > 0)?.id ?? "");
      })
      .catch((err) => setError(errorText(err)));
  }, [id, scope]);

  useEffect(load, [load]);

  // ── search inside the store, on the server, 350 ms after the last keystroke ──
  const [q, setQ] = useState("");
  const [needle, setNeedle] = useState("");
  useEffect(() => {
    const t = setTimeout(() => setNeedle(q.trim()), 350);
    return () => clearTimeout(t);
  }, [q]);

  // ── the items of the picked section (or the search), page by page ──
  const [items, setItems] = useState<Item[]>([]);
  const [next, setNext] = useState("");
  const [listLoading, setListLoading] = useState(false);
  const [listError, setListError] = useState("");
  const [rev, setRev] = useState(0);
  const pageUrl = useCallback(
    (after: string) => {
      const p = new URLSearchParams({ limit: String(PAGE) });
      if (needle) p.set("q", needle);
      else if (active) p.set("section", active);
      if (after) p.set("after", after);
      return `/api/v1/public/stores/${id}/items?${p.toString()}${scopeAmp}`;
    },
    [id, needle, active, scopeAmp],
  );
  useEffect(() => {
    if (!page || (!active && !needle)) return;
    let alive = true;
    setListLoading(true);
    setListError("");
    setItems([]);
    setNext("");
    api<StoreItemsPage>(pageUrl(""))
      .then((pg) => {
        if (!alive) return;
        setItems(pg.items ?? []);
        setNext(pg.next_after ?? "");
      })
      .catch((err) => alive && setListError(errorText(err)))
      .finally(() => alive && setListLoading(false));
    return () => {
      alive = false;
    };
  }, [page, active, needle, pageUrl, rev]);

  const moreBusy = useRef(false);
  const more = useCallback(() => {
    if (!next || moreBusy.current || listLoading) return;
    moreBusy.current = true;
    const after = next;
    api<StoreItemsPage>(pageUrl(after))
      .then((pg) => {
        setItems((cur) => {
          const have = new Set(cur.map((i) => i.id));
          return [...cur, ...(pg.items ?? []).filter((i) => !have.has(i.id))];
        });
        setNext((cur) => (cur === after ? (pg.next_after ?? "") : cur));
      })
      .catch((err) => setListError(errorText(err)))
      .finally(() => {
        moreBusy.current = false;
      });
  }, [next, listLoading, pageUrl]);

  // **The next page when the end of the grid comes into view.**
  const sentinel = useRef<HTMLDivElement | null>(null);
  useEffect(() => {
    const el = sentinel.current;
    if (!el || !next) return;
    const io = new IntersectionObserver((entries) => {
      if (entries.some((e) => e.isIntersecting)) more();
    }, { rootMargin: "600px" });
    io.observe(el);
    return () => io.disconnect();
  }, [next, more]);

  const sections = (page?.sections ?? []).filter((x) => x.count > 0);

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
              {/* **دوامُ اليوم وحدَه بلا حبّة «مفتوح / مسكّر»** (قرارُ المالك ٢٠٢٦-١٠-٢٨، مرآةُ `StoreScreen.kt`) —
                  **والإغلاقُ الطارئُ يُقال بلفظه** فلا يُقرأ الدوامُ وعداً بأنّه مفتوح. */}
              {page.store.emergency_closed ? (
                <p className="text-xs font-bold text-danger">{ST.emergencyClosed}</p>
              ) : (
                <p className="text-xs text-ink-muted">{todayText(page.store.today)}</p>
              )}
            </div>
          </header>

          {!page.store.open_now && <Alert tone="warning">{ST.browseOnly}</Alert>}

          {sections.length === 0 ? (
            <EmptyState icon={IconStore} title={ST.empty} action={<span className="text-xs text-ink-muted">{ST.emptyHint}</span>} />
          ) : (
            <>
              {/* ── دوائرُ الأقسام كالسوق، وتحتها أصنافُ المختار وحدَه ── (ملاحظةُ المالك ٢٠٢٦-١٠-١٨) */}
              {/* ── search inside the store (2026-11-07) ── */}
              <div className="relative">
                <span className="pointer-events-none absolute inset-y-0 start-3 flex items-center text-ink-muted" aria-hidden>
                  <IconSearch size={18} />
                </span>
                <input
                  type="search"
                  value={q}
                  onChange={(e) => setQ(e.target.value.slice(0, 60))}
                  placeholder={ST.search}
                  aria-label={ST.search}
                  data-testid="store-page-search"
                  className="h-12 w-full rounded-control border border-transparent bg-field pe-11 ps-10 text-sm text-ink outline-none placeholder:text-ink-muted focus:border-accent"
                />
                {q && (
                  <button
                    type="button"
                    onClick={() => setQ("")}
                    aria-label={ST.clearSearch}
                    className="absolute inset-y-0 end-1 flex w-10 items-center justify-center text-ink-muted hover:text-ink"
                  >
                    <IconClose size={18} />
                  </button>
                )}
              </div>
              {!q.trim() && (
                <SectionRail
                  items={sections.map((sec) => ({
                    id: sec.id,
                    name: sec.name,
                    imageUrl: mediaUrl(sec.image_thumb_url ?? sec.image_url),
                    icon: sec.icon,
                    count: sec.count,
                  }))}
                  activeID={active}
                  auto={false}
                  onSelect={setActive}
                />
              )}
              {items.length === 0 && listLoading ? (
                <LoadingState variant="list" rows={3} />
              ) : items.length === 0 && listError ? (
                <ReloadState label={listError} onRetry={() => setRev((r) => r + 1)} />
              ) : items.length === 0 ? (
                <p className="py-6 text-center text-sm text-ink-muted">{ST.noMatch}</p>
              ) : (
                <>
                  <ItemGrid items={items} onOpen={setOpen} storeLink={false} />
                  {next && (
                    <div ref={sentinel}>
                      <LoadingState variant="list" rows={1} />
                    </div>
                  )}
                </>
              )}
            </>
          )}
        </>
      )}

      <ItemSheet item={open} onClose={() => setOpen(null)} storeLink={false} />
    </div>
  );
}
