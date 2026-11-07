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
import { getMessages, defaultLocale, errorText, fmtNum } from "@rahalgo/i18n";
import { Alert, EmptyState, LoadingState, ReloadState, IconClose, IconPrev, IconSearch, IconStore } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useBrowseQuery } from "@/lib/customer/context";
import type { Item, StoreItemsPage, StoreSummary } from "@/lib/customer/types";
import { clock12, fill } from "@/lib/customer/texts";
import { ItemGrid, ItemSheet } from "@/components/customer/items";
import { ShopPic } from "@/components/customer/ShopPic";
import { StoreCartPanel } from "@/components/customer/StoreCartPanel";

const m = getMessages(defaultLocale);
const ST = m.shop.stores;
const SK = m.shop.skin;

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
  const activeName = sections.find((x) => x.id === active)?.name ?? "";

  return (
    <div className="flex flex-col gap-4">
      {error ? (
        <>
          <Link href="/shop" className="shop-chip w-fit">
            <IconPrev size={18} />
            {ST.back}
          </Link>
          <ReloadState label={error} onRetry={load} />
        </>
      ) : !page ? (
        <LoadingState variant="list" rows={4} />
      ) : (
        <>
          {/* ── الرأس: شريطٌ مصبوغٌ وشعارٌ يعلوه ودوامُ اليوم ── */}
          <header className="shop-store-hero">
            {/* **زرٌّ واضحٌ بسهم رجوع** (ملاحظةُ المالك ٢٠٢٦-١٠-١٨) */}
            <Link href="/shop" className="shop-back">
              <IconPrev size={18} />
              {ST.back}
            </Link>
            <div className="shop-store-band">
              <ShopPic src={null} name={page.store.name} iconSize={0} />
            </div>
            <div className="shop-store-head">
              <div className="shop-store-logo">
                <ShopPic
                  src={page.store.logo_thumb_url ?? page.store.logo_url}
                  name={page.store.name}
                  fit="contain"
                  initial
                  className="h-full w-full"
                />
              </div>
              <div className="min-w-0 flex-1 space-y-1.5 pb-1">
                <h1 className="truncate">{page.store.name}</h1>
                {/* **دوامُ اليوم وحدَه بلا حبّة «مفتوح / مسكّر»** (قرارُ المالك ٢٠٢٦-١٠-٢٨، مرآةُ `StoreScreen.kt`) —
                    **والإغلاقُ الطارئُ يُقال بلفظه** فلا يُقرأ الدوامُ وعداً بأنّه مفتوح. */}
                <div className="flex flex-wrap items-center gap-2">
                  {page.store.emergency_closed ? (
                    <span className="shop-cap" data-tone="bad">
                      {ST.emergencyClosed}
                    </span>
                  ) : (
                    <span className="shop-cap" data-tone={page.store.open_now ? "ok" : "brand"}>
                      {todayText(page.store.today)}
                    </span>
                  )}
                  {page.store.prep_minutes > 0 && (
                    <span className="shop-cap" data-tone="brand">
                      {fill(SK.prep, { n: fmtNum(page.store.prep_minutes) })}
                    </span>
                  )}
                </div>
              </div>
            </div>
          </header>

          {!page.store.open_now && <Alert tone="warning">{ST.browseOnly}</Alert>}

          {sections.length === 0 ? (
            <EmptyState icon={IconStore} title={ST.empty} action={<span className="text-xs text-ink-muted">{ST.emptyHint}</span>} />
          ) : (
            <div className="shop-store-cols">
              {/* ── أقسامُ القائمة: عمودٌ لاصقٌ على الحاسوب ── */}
              {!q.trim() ? (
                <nav className="shop-menu-rail" aria-label={SK.menuSections}>
                  <h2>{SK.menuSections}</h2>
                  {sections.map((sec) => (
                    <button
                      key={sec.id}
                      type="button"
                      className="shop-menu-link"
                      aria-pressed={sec.id === active}
                      onClick={() => setActive(sec.id)}
                    >
                      {sec.name}
                      <small>{fmtNum(sec.count)}</small>
                    </button>
                  ))}
                </nav>
              ) : (
                <span className="hidden lg:block" />
              )}

              <div className="flex min-w-0 flex-col gap-4">
                {/* ── search inside the store (2026-11-07) ── */}
                <div className="relative">
                  <span className="pointer-events-none absolute inset-y-0 start-3 z-10 flex items-center text-ink-muted" aria-hidden>
                    <IconSearch size={18} />
                  </span>
                  <input
                    type="search"
                    value={q}
                    onChange={(e) => setQ(e.target.value.slice(0, 60))}
                    placeholder={ST.search}
                    aria-label={ST.search}
                    data-testid="store-page-search"
                    className="h-12 w-full rounded-tile border border-line bg-surface pe-11 ps-10 text-sm text-ink outline-none placeholder:text-ink-muted focus:border-accent"
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
                  <div className="shop-chips" data-phone>
                    {sections.map((sec) => (
                      <button
                        key={sec.id}
                        type="button"
                        className="shop-chip"
                        aria-pressed={sec.id === active}
                        onClick={() => setActive(sec.id)}
                      >
                        {sec.name}
                      </button>
                    ))}
                  </div>
                )}
                {!q.trim() && activeName && (
                  <div className="shop-sec-head">
                    <h2>{activeName}</h2>
                  </div>
                )}
                {items.length === 0 && listLoading ? (
                  <LoadingState variant="list" rows={3} />
                ) : items.length === 0 && listError ? (
                  <ReloadState label={listError} onRetry={() => setRev((r) => r + 1)} />
                ) : items.length === 0 ? (
                  <p className="py-6 text-center text-sm text-ink-muted">{ST.noMatch}</p>
                ) : (
                  <>
                    <ItemGrid items={items} onOpen={setOpen} storeLink={false} layout="row" />
                    {next && (
                      <div ref={sentinel}>
                        <LoadingState variant="list" rows={1} />
                      </div>
                    )}
                  </>
                )}
              </div>

              <StoreCartPanel storeId={id} storeName={page.store.name} />
            </div>
          )}
        </>
      )}

      <ItemSheet item={open} onClose={() => setOpen(null)} storeLink={false} />
    </div>
  );
}
