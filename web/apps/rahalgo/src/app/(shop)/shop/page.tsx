"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **السوق — بحثٌ ثمّ لافتاتٌ ثمّ شريطُ الأقسام وأصنافُ المختار تحته**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والترتيبُ (قرارُ المالك ٢٠٢٦-١٠-١٨)**: **سلايدرٌ إن كان ← شريطُ «متاجر مميّزة» ← بحثٌ
 * ← أقسام** — والسلايدرُ مطفأٌ أو فارغٌ يجعل الشريطَ فوق البحث مباشرةً. **وترتيبُ تطبيق
 * أندرويد هو هو** (`ShopTop.kt`). **والأصنافُ تحت الشريط مباشرةً**
 * (طلبُ المالك ٢٠٢٦-١٠-١٧: «بدي التسوق نفس طريقة الجوال، مافي داعي افوت على الاقسام») —
 * أوّلُ قسمٍ مختارٌ، والنقرُ يبدّله في مكانه بلا صفحةٍ ثانية.
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
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import {
  Alert,
  BannerSlider,
  EmptyState,
  Input,
  LoadingState,
  ReloadState,
  IconSearch,
  IconStore,
  SectionRail,
  type RailItem,
  type SlideItem,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useBrowseQuery } from "@/lib/customer/context";
import type { HomePage, Item } from "@/lib/customer/types";
import { AddressChip, AddressSheet } from "@/components/customer/AddressSheet";
import { ItemGrid, ItemSheet } from "@/components/customer/items";
import { FeaturedStrip } from "@/components/customer/stores";
import { InviteHomeCard } from "@/components/customer/InviteNudge";

const m = getMessages(defaultLocale);
const H = m.shop.home;
const SC = m.shop.section;

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

  // **القسمُ المختار وأصنافُه** — أوّلُ قسمٍ افتراضاً.
  const [active, setActive] = useState("");
  const [items, setItems] = useState<Item[] | null>(null);
  const [itemsErr, setItemsErr] = useState("");
  const sectionID = active || home?.sections[0]?.id || "";
  const loadItems = useCallback(() => {
    if (!sectionID) return;
    setItemsErr("");
    setItems(null);
    api<{ items: Item[] }>(`/api/v1/public/sections/${sectionID}/items${scopeQ}`)
      .then((r) => setItems(r.items ?? []))
      .catch((err) => setItemsErr(errorText(err)));
  }, [sectionID, scopeQ]);
  useEffect(loadItems, [loadItems]);

  const slides: SlideItem[] = (home?.banners ?? []).map((b) => ({
    id: b.id,
    title: b.title,
    imageUrl: mediaUrl(b.image_url),
    sizes: b.sizes,
  }));

  const searching = search.q.trim().length > 0;
  const rail: RailItem[] = (home?.sections ?? []).map((sec) => ({
    id: sec.id,
    name: sec.name,
    imageUrl: mediaUrl(sec.image_thumb_url ?? sec.image_url),
    icon: sec.icon,
    count: sec.count,
  }));

  return (
    <div className="flex flex-col gap-4">
      {user ? (
        <AddressChip onOpen={() => setPickAddr(true)} />
      ) : (
        <p className="text-xs text-ink-muted">{H.browsing}</p>
      )}

      {/* **سلايدرٌ ثمّ «متاجر مميّزة» ثمّ البحث** — ولا يُرسمان وهو يبحث: نتائجُه أولى. */}
      {!searching && home && slides.length > 0 && (
        <BannerSlider items={slides} Link={Link} everyMs={home.banner_auto ? home.banner_every_ms || 5000 : 0} />
      )}
      {!searching && home && <FeaturedStrip stores={home.featured_stores ?? []} />}

      <Input
        id="shop-search"
        type="search"
        icon={<IconSearch />}
        placeholder={H.search}
        value={search.q}
        onChange={(e) => search.setQ(e.target.value)}
        aria-label={H.search}
      />

      {/* **«ادعُ صديقك» تحت البحث** — تُخفى بـ✕ للأبد (قرارُ المالك ٢٠٢٦-١٠-٣٠). */}
      {user && !searching && <InviteHomeCard />}

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
          {home.sections.length === 0 ? (
            <EmptyState icon={IconStore} title={H.empty} action={<span className="text-xs text-ink-muted">{H.emptyHint}</span>} />
          ) : (
            <section className="flex flex-col gap-3">
              <SectionRail items={rail} activeID={sectionID} auto={false} onSelect={setActive} />
              {itemsErr ? (
                <ReloadState label={itemsErr} onRetry={loadItems} />
              ) : items === null ? (
                <LoadingState variant="list" rows={4} />
              ) : items.length === 0 ? (
                <EmptyState icon={IconStore} title={SC.empty} action={<span className="text-xs text-ink-muted">{SC.emptyHint}</span>} />
              ) : (
                <ItemGrid items={items} onOpen={setOpen} />
              )}
            </section>
          )}
        </>
      )}

      <ItemSheet item={open} onClose={() => setOpen(null)} />
      <AddressSheet open={pickAddr} onClose={() => setPickAddr(false)} />
    </div>
  );
}
