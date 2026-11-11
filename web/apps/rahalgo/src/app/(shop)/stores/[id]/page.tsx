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
import { Alert, Badge, EmptyState, Input, LoadingState, ReloadState, IconClose, IconPrev, IconSearch, IconStore } from "@rahalgo/ui";
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
  const bar = useRef<HTMLElement | null>(null);

  // **القسمُ المختارُ يبقى ظاهراً في الشريط** — أفقيّاً فقط، فلا تقفز الصفحةُ.
  useEffect(() => {
    const chip = bar.current?.querySelector<HTMLElement>('[aria-pressed="true"]');
    const row = chip?.parentElement;
    if (!chip || !row) return;
    const c = chip.getBoundingClientRect();
    const r = row.getBoundingClientRect();
    row.scrollBy({ left: c.left + c.width / 2 - (r.left + r.width / 2), behavior: "smooth" });
  }, [active, page]);

  // **عند تبديل القسم والشريطُ لاصقٌ** تعود الصفحةُ إلى رأس الأصناف الجديدة — لا تبقى في ذيل قسمٍ طويل.
  const pick = (secId: string) => {
    setActive(secId);
    const el = bar.current;
    if (el && el.getBoundingClientRect().top <= 140) {
      window.scrollTo({ top: window.scrollY + el.getBoundingClientRect().top - (window.innerWidth >= 1024 ? 69 : 61), behavior: "auto" });
    }
  };
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
          {/* **الرأسُ بطاقةٌ بحدٍّ**: سهمُ رجوعٍ (٤٨) وشعارٌ (٥٦ مستدير) واسمٌ بسطرين ودوامُ اليوم — مرآةُ `StoreScreen.kt`. */}
          <header className="flex items-center gap-rg-md rounded-rg-lg border border-line bg-surface p-rg-md">
            <Link
              href="/shop"
              aria-label={ST.back}
              className="flex h-12 w-12 shrink-0 items-center justify-center rounded-full hover:bg-row-hover"
            >
              <IconPrev size={22} />
            </Link>
            <div className="h-14 w-14 shrink-0 overflow-hidden rounded-rg-md border border-line-soft bg-field">
              <ShopPic
                src={page.store.logo_thumb_url ?? page.store.logo_url}
                name={page.store.name}
                fit="contain"
                initial
                className="h-full w-full"
              />
            </div>
            <div className="min-w-0 flex-1">
              <h1 className="line-clamp-2 text-section-title">{page.store.name}</h1>
              {/* **دوامُ اليوم وحدَه بلا حبّة «مفتوح / مسكّر»** (قرارُ المالك ٢٠٢٦-١٠-٢٨، مرآةُ `StoreScreen.kt`) —
                  **والإغلاقُ الطارئُ يُقال بلفظه** فلا يُقرأ الدوامُ وعداً بأنّه مفتوح. */}
              <div className="flex flex-wrap items-center gap-x-rg-md gap-y-rg-xxs text-caption">
                {page.store.emergency_closed ? (
                  <span className="font-bold text-danger">{ST.emergencyClosed}</span>
                ) : (
                  <span className="text-ink-muted">{todayText(page.store.today)}</span>
                )}
                {/* **شارةُ «مفتوح هلق / مسكّر هلق»** (قرارُ المالك ٢٠٢٦-١١-١١، مرآةُ `StoreScreen.kt`) — من
                    `open_now` الذي يردّه المحرّك. **ورسمُ التوصيل لعنوانه لا يُعرض**: لا تقديرَ بلا سلّة
                    (`/public/quote` يطلب أصنافاً) فلا يُختلَق رقم. */}
                <Badge variant={page.store.open_now && !page.store.emergency_closed ? "success" : "neutral"}>
                  {page.store.open_now && !page.store.emergency_closed ? ST.openNow : ST.closedNow}
                </Badge>
                {page.store.prep_minutes > 0 && (
                  <span className="text-ink-muted">{fill(SK.prep, { n: fmtNum(page.store.prep_minutes) })}</span>
                )}
              </div>
            </div>
          </header>

          {!page.store.open_now && <Alert tone="warning">{ST.browseOnly}</Alert>}

          {sections.length === 0 ? (
            <EmptyState icon={IconStore} title={ST.empty} line={ST.emptyHint} />
          ) : (
            <div className="shop-store-cols">
              <div className="flex min-w-0 flex-col gap-4">
                {/* ── search inside the store (2026-11-07) ── */}
                {/* **حقلُ البحث هو حقلُ الرئيسيّة نفسُه** (تدقيق D16، ٢٠٢٦-١١-١١): مكوّنُ `Input` المشترك بعدسته، لا حقلٌ ثانٍ بحدٍّ آخر. */}
                <div className="relative">
                  <Input
                    id="store-page-search"
                    type="search"
                    icon={<IconSearch />}
                    value={q}
                    onChange={(e) => setQ(e.target.value.slice(0, 60))}
                    placeholder={ST.search}
                    aria-label={ST.search}
                    data-testid="store-page-search"
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
                  /* **شريطُ الأقسام تحت البحث على كلّ المقاسات** (قرارُ المالك ٢٠٢٦-١١-٠٩، كالجوّال والتطبيق):
                     لاصقٌ تحت الشريط العلويّ، والقسمُ المختارُ يبقى في المرمى، وتحته أصنافُه وحدَها. */
                  <nav ref={bar} className="shop-store-sections" aria-label={SK.menuSections}>
                    <div className="shop-chips">
                      {sections.map((sec) => (
                        <button
                          key={sec.id}
                          type="button"
                          className="shop-chip"
                          aria-pressed={sec.id === active}
                          onClick={() => pick(sec.id)}
                        >
                          {sec.name}
                        </button>
                      ))}
                    </div>
                  </nav>
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
                  <EmptyState icon={IconSearch} title={ST.noMatch} />
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
