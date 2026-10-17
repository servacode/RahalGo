"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **السوق — بحثٌ ثمّ لافتاتٌ ثمّ أقسام**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (ترتيبُ تطبيق أندرويد حرفاً: **بحثٌ ← لافتاتٌ ← أقسام.**)
 *
 * **وكلُّ نداءٍ يحمل موضعَ التوصيل** — المحرّكُ يرشّح بالمدينة والمسافة
 * (قرارُ المالك ٢٠٢٦-٠٨-٢٠: «مو معقول شخصٌ بالشام يطلب من الرقّة»).
 * **والزائرُ بلا عنوانٍ يرى السوقَ كلَّه** كما يراه التطبيقُ القديم.
 *
 * **والبحثُ ينتظر أن يسكت** ثلاثَ مئةِ ملّي — كما في أندرويد — **وحرفان
 * حدُّه الأدنى**: حرفٌ واحدٌ يجلب السوقَ كلَّه.
 */

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtNum } from "@rahalgo/i18n";
import {
  Alert,
  BannerSlider,
  CategoryIcon,
  EmptyState,
  Input,
  LoadingState,
  ReloadState,
  IconSearch,
  IconStore,
  type SlideItem,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useBrowseQuery } from "@/lib/customer/context";
import { fill } from "@/lib/customer/texts";
import type { HomePage, Item } from "@/lib/customer/types";
import { AddressChip, AddressSheet } from "@/components/customer/AddressSheet";
import { ItemGrid, ItemSheet } from "@/components/customer/items";

const m = getMessages(defaultLocale);
const H = m.shop.home;

function useSearch(scope: string) {
  const [q, setQ] = useState("");
  const [items, setItems] = useState<Item[] | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    const term = q.trim();
    setError("");
    if (term.length < 2) {
      setItems(null);
      return;
    }
    let alive = true;
    setBusy(true);
    const t = setTimeout(() => {
      api<{ items: Item[] }>(`/api/v1/public/search/items?q=${encodeURIComponent(term)}${scope}`)
        .then((r) => alive && setItems(r.items ?? []))
        .catch((err) => alive && setError(errorText(err)))
        .finally(() => alive && setBusy(false));
    }, 300);
    return () => {
      alive = false;
      clearTimeout(t);
    };
  }, [q, scope]);
  return { q, setQ, items, error, busy };
}

export default function ShopPage() {
  const { user } = useAuth();
  const scopeQ = useBrowseQuery("?");
  const scopeA = useBrowseQuery("&");
  const [home, setHome] = useState<HomePage | null>(null);
  const [error, setError] = useState("");
  const [open, setOpen] = useState<Item | null>(null);
  const [pickAddr, setPickAddr] = useState(false);
  const search = useSearch(scopeA);

  const load = useCallback(() => {
    setError("");
    api<HomePage>(`/api/v1/public/home${scopeQ}`)
      .then(setHome)
      .catch((err) => setError(errorText(err)));
  }, [scopeQ]);

  useEffect(load, [load]);

  const slides: SlideItem[] = (home?.banners ?? []).map((b) => ({
    id: b.id,
    title: b.title,
    imageUrl: mediaUrl(b.image_url),
    sizes: b.sizes,
  }));

  const searching = search.q.trim().length > 0;

  return (
    <div className="flex flex-col gap-4">
      {user ? (
        <AddressChip onOpen={() => setPickAddr(true)} />
      ) : (
        <p className="text-xs text-ink-muted">{H.browsing}</p>
      )}

      <Input
        id="shop-search"
        type="search"
        icon={<IconSearch />}
        placeholder={H.search}
        value={search.q}
        onChange={(e) => search.setQ(e.target.value)}
        aria-label={H.search}
      />

      {searching ? (
        <section className="flex flex-col gap-3">
          <h2 className="heading-section">{H.results}</h2>
          {search.q.trim().length < 2 ? (
            <p className="text-sm text-ink-muted">{H.searchShort}</p>
          ) : search.error ? (
            <Alert>{search.error}</Alert>
          ) : search.items === null || search.busy ? (
            <LoadingState variant="list" rows={3} />
          ) : search.items.length === 0 ? (
            <EmptyState icon={IconSearch} title={H.noResults} action={<span className="text-xs text-ink-muted">{H.noResultsHint}</span>} />
          ) : (
            <ItemGrid items={search.items} onOpen={setOpen} />
          )}
        </section>
      ) : error ? (
        <ReloadState label={error} onRetry={load} />
      ) : !home ? (
        <LoadingState variant="list" rows={4} />
      ) : (
        <>
          {slides.length > 0 && (
            <BannerSlider items={slides} Link={Link} everyMs={home.banner_auto ? home.banner_every_ms || 5000 : 0} />
          )}
          <section className="flex flex-col gap-3">
            <h2 className="heading-section">{H.sections}</h2>
            {home.sections.length === 0 ? (
              <EmptyState icon={IconStore} title={H.empty} action={<span className="text-xs text-ink-muted">{H.emptyHint}</span>} />
            ) : (
              <div className="grid grid-cols-3 gap-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 xl:grid-cols-8">
                {home.sections.map((s) => {
                  const img = mediaUrl(s.image_thumb_url ?? s.image_url);
                  return (
                    <Link
                      key={s.id}
                      href={`/s/${s.id}`}
                      className="surface flex min-w-0 flex-col items-center gap-2 overflow-hidden p-2 text-center transition-transform active:scale-[0.97]"
                    >
                      <span className="flex aspect-square w-full items-center justify-center overflow-hidden rounded-control bg-field">
                        {img ? (
                          <img src={img} alt="" loading="lazy" className="h-full w-full object-cover" />
                        ) : (
                          <CategoryIcon name={s.icon} size={28} />
                        )}
                      </span>
                      <span className="line-clamp-2 text-xs font-bold leading-snug">{s.name}</span>
                      <span className="text-2xs text-ink-muted">{fill(H.count, { n: fmtNum(s.count) })}</span>
                    </Link>
                  );
                })}
              </div>
            )}
          </section>
        </>
      )}

      <ItemSheet item={open} onClose={() => setOpen(null)} />
      <AddressSheet open={pickAddr} onClose={() => setPickAddr(false)} />
    </div>
  );
}
