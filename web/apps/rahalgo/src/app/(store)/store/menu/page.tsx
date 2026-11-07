"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/store/menu — «الأصناف»: لوحةُ الأصناف (٢٠٢٦-١١-٠٧)**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `MenuItemsGrid` في `ui/menu/MenuGrid.kt` — نسخةٌ واحدةٌ للمتجر والمندوب في التطبيق،
 *  وهذه مرآتُها على الويب. وإعادةُ التصميم بطلب المالك: «لازم يكون أقوى واحترافي أكثر».)
 *
 * **من فوق لتحت**: رأسٌ (العنوان + عدد الأصناف + الأدوات: أضف · أنشئ عرضاً · تغيير الأسعار
 * · الخيارات · صوّر الأصناف · المشروبات · مزامنة الأمين) · **شريطُ أدواتٍ لاصق** (البحث
 * + مبدّل «شبكة | جدول» على الحاسوب + رقائقُ «الكل» و«غير المتوفر» والأقسام) · ثمّ الأصناف.
 *
 * **جوّالاً**: بطاقتان في كلّ سطر، وأدواتٌ في مسارٍ أفقيّ بأهدافِ لمسٍ ٤٤، ومفتاحُ التوفّر
 * سطرٌ كاملٌ يُضغط. **حاسوباً**: شبكةٌ حتّى ستّة أعمدة **أو جدولٌ كثيف** (صورة · اسم · رمز ·
 * قسم · سعر بالدولار والليرة · توفّر · نسخ وتعديل) لمتاجرَ بآلاف الأصناف، والاختيارُ محفوظٌ
 * لكلّ جهاز (`useViewMode`).
 *
 * **والبطاقةُ/الصفُّ كلُّه يفتح المحرّر، والمفتاحُ وحدَه يبدّل التوفّر** — يتبدّل فوراً ثمّ
 * يُسأل المحرّك، **فإن ردّه عاد كما كان** وقيل السببُ كما قاله. **وغيرُ المتوفّر باهت.**
 *
 * **و`?filter=unavailable` يفتحها مصفّاةً على غير المتوفرة** — زرُّ «غير المتوفر» في
 * «متجري» يصل إليها.
 */

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useRef, useState, type ComponentType } from "react";
import { getMessages, defaultLocale, errorText, fmtMoney, fmtNum } from "@rahalgo/i18n";
import {
  ButtonLink,
  Chip,
  DataView,
  EmptyState,
  IconAdd,
  IconBanknote,
  IconCamera,
  IconCatDrinks,
  IconClose,
  IconCopy,
  IconEdit,
  IconList,
  IconOptions,
  IconPromos,
  IconRotate,
  IconSearch,
  ImageViewer,
  LoadingState,
  PageHead,
  ReloadState,
  ViewToggle,
  useLiveData,
  useViewMode,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import { useDrinksAllowed } from "@/lib/store/drinks";
import type { MenuItem, MenuItemsPage, MenuSummary } from "@/lib/store/types";
import { dollars, fill } from "@/lib/customer/texts";
import { useShopToast } from "@/components/customer/toast";

const m = getMessages(defaultLocale);
const M = m.store.menu;

type Icon = ComponentType<{ size?: number; className?: string }>;

function priceLine(it: MenuItem): string {
  return it.price_currency === "USD" && it.merchant_price_usd != null
    ? `${dollars(it.merchant_price_usd)} · ${fmtMoney(it.price)}`
    : fmtMoney(it.price);
}

/** **أسماءُ مجموعات خياراته** — لا وصفَ للصنف (قرارُ المالك ٢٠٢٦-١٠-٣٠)؛ مرآةُ `MenuScreen` في التطبيق. */
function descOf(it: MenuItem): string {
  return (it.modifiers ?? [])
    .map((g) => g.name.trim())
    .filter(Boolean)
    .join(" · ");
}

/**
 * **زرُّ أداةٍ** — أيقونةٌ واسمُها في سطرٍ واحد، ارتفاعُه ٤٤ (هدفُ لمس). والأوّلُ (`primary`)
 * بلون التمييز، والبقيةُ مؤطَّرة. من `ButtonLink` المركزيّ مع رابط التطبيق (بلا إعادة تحميل).
 */
function Tool({ icon: Ico, label, href, testId, primary = false }: { icon: Icon; label: string; href: string; testId: string; primary?: boolean }) {
  if (!primary) {
    return (
      <Link href={href} data-testid={testId} className="chip min-h-11 shrink-0">
        <Ico size={18} className="text-cta-start" />
        {label}
      </Link>
    );
  }
  return (
    <ButtonLink Link={Link} href={href} variant="primary" data-testid={testId} className="min-h-11 shrink-0 whitespace-nowrap">
      <Ico size={18} />
      {label}
    </ButtonLink>
  );
}

/** **مفتاحُ التوفّر** — مسارٌ ودائرة؛ والزرُّ الحاضنُ له هو هدفُ اللمس. */
function SwitchTrack({ on }: { on: boolean }) {
  return (
    <span
      aria-hidden
      className={`relative inline-flex h-6 w-11 shrink-0 items-center rounded-badge border transition-colors ${
        on ? "border-success-edge bg-success-fill" : "border-line bg-field"
      }`}
    >
      <span
        className={`absolute h-4.5 w-4.5 rounded-badge transition-[inset-inline-start] duration-200 ${
          on ? "start-[1.5rem] bg-success" : "start-0.5 bg-ink-muted"
        }`}
      />
    </span>
  );
}

/**
 * **بطاقةُ صنف** — صورتُه فوق، ثمّ اسمُه ووصفُه وسعرُه، ثمّ مفتاحُ توفّره.
 * **والبطاقةُ كلُّها تفتح المحرّر، والمفتاحُ وحدَه يبدّل التوفّر.**
 */
function ItemCard({ item, disabled, onToggle }: { item: MenuItem; disabled: boolean; onToggle: () => void }) {
  // **صورةٌ كاملةٌ طوليّةٌ ٤:٥** (قرار المالك ٢٠٢٦-١١-٠٦): المصغّرةُ ٤٠٠ بكسل فتتشوّش في بطاقةٍ أطول.
  const img = mediaUrl(item.image_url || item.image_thumb_url);
  const [viewing, setViewing] = useState(false);
  const dim = item.available ? "" : "opacity-50";
  const desc = descOf(item);
  const badge =
    item.approved === false ? (
      <span className="absolute start-2 top-2 rounded-full bg-accent px-2 py-0.5 text-2xs font-bold text-on-bright">
        {M.pendingReview}
      </span>
    ) : null;
  return (
    <li
      className="item-card"
      data-testid="store-menu-item"
      data-item-id={item.id}
      data-available={item.available}
    >
      {img ? (
        // **لمسةُ الصورة تكبّرها** — والباقي من البطاقة يفتح المحرّر.
        <button
          type="button"
          onClick={() => setViewing(true)}
          aria-label={`${item.name} — ${m.shop.item.zoomBtn}`}
          data-testid="store-menu-zoom"
          className={`relative block aspect-[4/5] w-full bg-field ${dim}`}
        >
          <img src={img} alt="" loading="lazy" className="absolute inset-0 h-full w-full object-cover" />
          {badge}
        </button>
      ) : (
        <Link href={`/store/menu/${item.id}`} aria-label={`${M.edit} · ${item.name}`} className={`relative block aspect-[4/5] w-full bg-field ${dim}`}>
          <span className="absolute inset-0 flex items-center justify-center tile-initial text-5xl" aria-hidden>
            {item.name.trim().charAt(0) || "?"}
          </span>
          {badge}
        </Link>
      )}
      {img && viewing && <ImageViewer src={img} alt={item.name} onClose={() => setViewing(false)} />}
      <Link href={`/store/menu/${item.id}`} aria-label={`${M.edit} · ${item.name}`} data-testid="store-menu-edit" className="flex flex-1 flex-col">
        <span className={`flex flex-col gap-0.5 px-3 pt-3 ${dim}`}>
          <span className="line-clamp-2 min-h-[2.5rem] text-sm font-bold leading-5 text-ink">{item.name}</span>
          <span className="line-clamp-2 min-h-[2rem] text-xs leading-4 text-ink-muted">{desc}</span>
          <span className="mt-1 line-clamp-2 text-base font-extrabold tabular-nums text-accent-text">{priceLine(item)}</span>
          {item.approved === false && item.review_note && (
            <span className="line-clamp-2 text-xs text-danger">{item.review_note}</span>
          )}
        </span>
      </Link>
      {/* **سطرُ التوفّر كلُّه زرٌّ** (≥ ٤٤) — كان المفتاحُ وحدَه ٢٤ بكسلاً. */}
      <button
        type="button"
        role="switch"
        aria-checked={item.available}
        aria-label={`${item.name} · ${item.available ? M.available : M.unavailable}`}
        disabled={disabled}
        onClick={onToggle}
        data-testid="store-menu-toggle"
        className="mx-3 mt-2 flex min-h-11 items-center gap-2 border-t border-line-soft text-start disabled:opacity-50"
      >
        <span className={`min-w-0 flex-1 truncate text-xs font-bold ${item.available ? "text-success" : "text-ink-muted"}`}>
          {item.available ? M.available : M.unavailable}
        </span>
        <SwitchTrack on={item.available} />
      </button>
    </li>
  );
}

/** **صورةٌ مصغّرةٌ في صفّ الجدول** — لمستُها تكبّرها، ومن غيرها حرفٌ أوّل. */
function RowThumb({ item }: { item: MenuItem }) {
  const img = mediaUrl(item.image_thumb_url || item.image_url);
  const [viewing, setViewing] = useState(false);
  return (
    <>
      {img ? (
        <button
          type="button"
          onClick={() => setViewing(true)}
          aria-label={`${item.name} — ${m.shop.item.zoomBtn}`}
          className={`block size-12 overflow-hidden rounded-tile bg-field ${item.available ? "" : "opacity-50"}`}
        >
          <img src={img} alt="" loading="lazy" className="size-full object-cover" />
        </button>
      ) : (
        <span
          className="flex size-12 items-center justify-center rounded-tile tile-initial text-lg"
          title={M.noPhoto}
          aria-label={M.noPhoto}
        >
          {item.name.trim().charAt(0) || "?"}
        </span>
      )}
      {img && viewing && <ImageViewer src={img} alt={item.name} onClose={() => setViewing(false)} />}
    </>
  );
}

/** **Page size of the paged menu** — 60 cards, a few tens of KB even for a 15,000-item mall. */
const PAGE = 60;

/**
 * **The paged menu** (owner 2026-11-07: «ما يصير يكون بطيء») — mirror of `MenuViewModel` in the
 * merchant app. A mall of 5,311 items made the full menu 3 MB (45 s on a weak link), so the page
 * reads a light summary (chips and counts) and then pages of 60 items; the chips, «غير المتوفر»
 * and the search ask the server, and the next page comes when the end of the grid shows.
 */
export default function MenuPage() {
  const { store } = useStore();
  const id = store?.id ?? "";
  // **أداةُ المشروبات للمطاعم وحدَها** (قرارُ المالك ٢٠٢٦-١٠-٣٠).
  const drinksAllowed = useDrinksAllowed(id);
  const toast = useShopToast();
  const router = useRouter();
  /** **جدولٌ على الحاسوب افتراضاً** (آلافُ الأصناف)، والجوّالُ بطاقاتٌ دائماً؛ والاختيارُ محفوظ. */
  const [view, setView] = useViewMode("store-menu", "table");
  const dupBusy = useRef(false);
  /** **ارتفاعُ شريط الأدوات اللاصق** — تقف تحتَه ترويسةُ الجدول اللاصقة. */
  const [barEl, setBarEl] = useState<HTMLDivElement | null>(null);
  const [barH, setBarH] = useState(0);
  useEffect(() => {
    if (!barEl) return;
    const sync = () => setBarH(barEl.offsetHeight + 8);
    sync();
    const ro = new ResizeObserver(sync);
    ro.observe(barEl);
    return () => ro.disconnect();
  }, [barEl]);
  const load = useCallback(() => api<MenuSummary>(`/api/v1/merchant/stores/${id}/menu/summary`), [id]);
  const { data, loading, error, reload, setData } = useLiveData(load, ["menu"], [id]);
  const [q, setQ] = useState("");
  /** **The search sent to the server** — 350 ms after the last keystroke. */
  const [needle, setNeedle] = useState("");
  useEffect(() => {
    const t = setTimeout(() => setNeedle(q.trim()), 350);
    return () => clearTimeout(t);
  }, [q]);
  /** **المفتوح**: `null` الكلّ، أو معرّفُ قسم. */
  const [open, setOpen] = useState<string | null>(null);
  const [onlyOff, setOnlyOff] = useState(false);
  const suspended = store?.status === "suspended";

  // **`?filter=unavailable`** — من زرّ «غير المتوفر» في «متجري».
  useEffect(() => {
    try {
      if (new URLSearchParams(window.location.search).get("filter") === "unavailable") setOnlyOff(true);
    } catch {
      /* لا عنوانَ يُقرأ — تبقى «الكل». */
    }
  }, []);

  // ── the items of the current filter, page by page ──
  const [items, setItems] = useState<MenuItem[]>([]);
  const [next, setNext] = useState("");
  const [listLoading, setListLoading] = useState(true);
  const [listFailed, setListFailed] = useState(false);
  const [rev, setRev] = useState(0);
  /** **عددُ ما يطابق الترشيحَ** — من الصفحة الأولى وحدَها (`-1` بعدها). */
  const [listTotal, setListTotal] = useState(0);
  // **A live «menu» event reloads the summary — and the list with it** (not on the first load).
  const seenSummary = useRef(false);
  useEffect(() => {
    if (!data) return;
    if (!seenSummary.current) {
      seenSummary.current = true;
      return;
    }
    setRev((r) => r + 1);
  }, [data]);

  const pageUrl = useCallback(
    (after: string) => {
      const p = new URLSearchParams({ limit: String(PAGE) });
      if (needle) p.set("q", needle);
      else if (onlyOff) p.set("filter", "unavailable");
      else if (open) p.set("section", open);
      if (after) p.set("after", after);
      return `/api/v1/merchant/stores/${id}/menu/items?${p.toString()}`;
    },
    [id, needle, onlyOff, open],
  );

  useEffect(() => {
    if (!id) return;
    let alive = true;
    setListLoading(true);
    setListFailed(false);
    setItems([]);
    setNext("");
    api<MenuItemsPage>(pageUrl(""))
      .then((pg) => {
        if (!alive) return;
        setItems(pg.items ?? []);
        setNext(pg.next_after ?? "");
        setListTotal(pg.total >= 0 ? pg.total : 0);
      })
      .catch(() => alive && setListFailed(true))
      .finally(() => alive && setListLoading(false));
    return () => {
      alive = false;
    };
  }, [id, pageUrl, rev]);

  const moreBusy = useRef(false);
  const more = useCallback(() => {
    if (!next || moreBusy.current || listLoading) return;
    moreBusy.current = true;
    const after = next;
    const url = pageUrl(after);
    api<MenuItemsPage>(url)
      .then((pg) => {
        setItems((cur) => {
          const have = new Set(cur.map((i) => i.id));
          return [...cur, ...(pg.items ?? []).filter((i) => !have.has(i.id))];
        });
        setNext((cur) => (cur === after ? (pg.next_after ?? "") : cur));
      })
      .catch((e) => toast(errorText(e), "error"))
      .finally(() => {
        moreBusy.current = false;
      });
  }, [next, listLoading, pageUrl, toast]);

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

  const sections = data?.sections ?? [];
  const total = data?.total ?? 0;
  const off = data?.unavailable ?? 0;
  const noPhoto = data?.no_photo ?? 0;

  // **ومن داخل قسمٍ يُضاف إليه** — فلا يُعيد اختيارَ ما هو فيه.
  const openSection = open != null && !onlyOff ? open : "";

  const showAll = () => {
    setOnlyOff(false);
    setOpen(null);
  };

  async function toggle(item: MenuItem) {
    const next = !item.available;
    const flip = (v: boolean) => {
      setItems((list) => list.map((i) => (i.id === item.id ? { ...i, available: v } : i)));
      // **والعدّادات تتبع فوراً** — «غير المتوفر» ورقاقةُ القسم.
      const d = v === item.available ? 0 : v ? -1 : 1;
      setData((s) =>
        s
          ? {
              ...s,
              unavailable: Math.max(0, s.unavailable + d),
              sections: s.sections.map((x) =>
                x.id === item.platform_section_id ? { ...x, unavailable: Math.max(0, x.unavailable + d) } : x,
              ),
            }
          : s,
      );
    };
    flip(next);
    try {
      await api(`/api/v1/merchant/menu/items/${item.id}/availability`, {
        method: "PATCH",
        body: JSON.stringify({ available: next }),
      });
    } catch (e) {
      flip(item.available);
      toast(errorText(e), "error");
    }
  }

  async function duplicate(item: MenuItem) {
    if (dupBusy.current) return;
    dupBusy.current = true;
    try {
      const r = await api<{ id: string }>(`/api/v1/merchant/menu/items/${item.id}/duplicate`, { method: "POST" });
      toast(M.duplicated, "success");
      router.push(`/store/menu/${r.id}`);
    } catch (e) {
      toast(errorText(e), "error");
    } finally {
      dupBusy.current = false;
    }
  }

  if (loading && !data) return <LoadingState variant="list" rows={5} />;
  if (error && !data) return <ReloadState onRetry={() => void reload()} />;

  const asTable = view === "table";

  return (
    <div className="console-page" data-testid="store-menu">
      <PageHead
        title={M.title}
        subtitle={fill(M.count, { n: fmtNum(total) })}
        actions={
          !suspended && (
            <div className="flex flex-wrap items-center gap-2">
              <Tool
                icon={IconAdd}
                label={M.add}
                href={openSection ? `/store/menu/new?section=${openSection}` : "/store/menu/new"}
                testId="menu-add"
                primary
              />
              <Tool icon={IconPromos} label={M.addOffer} href="/store/offers?new=1" testId="menu-offer" />
            </div>
          )
        }
      />

      {/* ── أدواتُ المتجر: سطرٌ واحدٌ يُمرَّر على الجوّال ── */}
      {!suspended && (
        <div className="rail-x -mx-3 px-3 sm:mx-0 sm:flex-wrap sm:px-0" aria-label={M.tools}>
          <Tool icon={IconBanknote} label={M.toolBulk} href="/store/menu/prices" testId="menu-tool-prices" />
          <Tool icon={IconOptions} label={M.toolPresets} href="/store/menu/options" testId="menu-tool-options" />
          {noPhoto > 0 && (
            <Tool
              icon={IconCamera}
              label={fill(M.toolQuickPhoto, { n: fmtNum(noPhoto) })}
              href="/store/menu/photos"
              testId="menu-tool-photos"
            />
          )}
          {drinksAllowed === true && <Tool icon={IconCatDrinks} label={M.toolDrinks} href="/store/menu/drinks" testId="menu-tool-drinks" />}
          {/* **مزامنةُ الأمين أداةٌ ظاهرة** (٢٠٢٦-١١-٠٧) — كانت سطراً تحت آلاف الأصناف فلا يُرى. */}
          {store?.uses_ameen === true && (
            <Tool icon={IconRotate} label={m.store.priceSync.open} href="/store/price-sync" testId="menu-tool-price-sync" />
          )}
        </div>
      )}

      {/* ── شريطُ البحث والترشيح: لاصقٌ على الحاسوب ── */}
      <div ref={setBarEl} className="toolbar max-lg:static max-lg:border-0 max-lg:bg-transparent max-lg:p-0 max-lg:shadow-none max-lg:backdrop-blur-none">
        <div className="flex items-center gap-3">
          <div className="relative min-w-0 flex-1">
            <span className="pointer-events-none absolute inset-y-0 start-3 flex items-center text-ink-muted" aria-hidden>
              <IconSearch size={18} />
            </span>
            <input
              type="search"
              value={q}
              onChange={(e) => setQ(e.target.value.slice(0, 60))}
              placeholder={M.search}
              aria-label={M.search}
              data-testid="store-menu-search"
              className="search-field"
            />
            {q && (
              <button
                type="button"
                onClick={() => setQ("")}
                aria-label={M.clearSearch}
                className="absolute inset-y-0 end-1 flex w-11 items-center justify-center text-ink-muted hover:text-ink"
              >
                <IconClose size={18} />
              </button>
            )}
          </div>
          <ViewToggle view={view} onChange={setView} tableLabel={M.viewTable} cardsLabel={M.viewGrid} />
        </div>

        {/* ── الرقائق: «الكل» · «غير المتوفر» · الأقسام ── */}
        {sections.length > 0 && !q.trim() && (
          <div className="rail-x -mx-3 px-3 pb-0.5 lg:mx-0 lg:px-0" aria-label={M.filters}>
            <Chip selected={open == null && !onlyOff} count={total} onClick={showAll} testId="menu-chip">
              {M.all}
            </Chip>
            <Chip
              selected={onlyOff}
              count={off}
              testId="menu-stat-off"
              onClick={() => {
                if (onlyOff) return showAll();
                setOnlyOff(true);
                setOpen(null);
                setQ("");
              }}
            >
              {M.unavailable}
            </Chip>
            {sections.map((g) => (
              <Chip
                key={g.id}
                selected={open === g.id && !onlyOff}
                count={g.count}
                testId="menu-chip"
                onClick={() => {
                  setOnlyOff(false);
                  setOpen(g.id);
                }}
              >
                {g.name}
              </Chip>
            ))}
          </div>
        )}
      </div>

      {/* ── الأصناف ── */}
      {total === 0 ? (
        <EmptyState
          icon={IconList}
          title={M.empty}
          action={
            !suspended ? (
              <ButtonLink Link={Link} href="/store/menu/new">
                <IconAdd size={18} />
                {M.add}
              </ButtonLink>
            ) : undefined
          }
        />
      ) : items.length === 0 && listLoading ? (
        <LoadingState variant="list" rows={3} />
      ) : items.length === 0 && listFailed ? (
        <ReloadState onRetry={() => setRev((r) => r + 1)} />
      ) : items.length === 0 ? (
        <p className="py-6 text-center text-sm text-ink-muted" data-testid="menu-none">
          {onlyOff && !needle ? M.offNone : M.noMatch}
        </p>
      ) : (
        <>
          {asTable ? (
            <div style={{ ["--table-head-offset" as string]: `${barH}px` }} data-testid="store-menu-table">
              <DataView<MenuItem>
                dense
                view="table"
                items={items}
                getKey={(i) => i.id}
                empty={M.noMatch}
                onRowClick={(i) => router.push(`/store/menu/${i.id}`)}
                columns={[
                  {
                    id: "image",
                    header: M.colImage,
                    widthClass: "w-16",
                    cell: (i) => (
                      <span onClick={(e) => e.stopPropagation()} className="inline-flex">
                        <RowThumb item={i} />
                      </span>
                    ),
                  },
                  {
                    id: "name",
                    header: M.colName,
                    cell: (i) => (
                      <div className={`flex min-w-0 flex-col ${i.available ? "" : "opacity-60"}`} data-testid="store-menu-item" data-item-id={i.id} data-available={i.available}>
                        <Link
                          href={`/store/menu/${i.id}`}
                          onClick={(e) => e.stopPropagation()}
                          aria-label={`${M.edit} · ${i.name}`}
                          data-testid="store-menu-edit"
                          className="line-clamp-2 font-bold text-ink hover:underline"
                        >
                          {i.name}
                        </Link>
                        {descOf(i) && <span className="line-clamp-1 text-xs text-ink-muted">{descOf(i)}</span>}
                        {i.approved === false && (
                          <span className="text-xs font-bold text-warning">
                            {M.pendingReview}
                            {i.review_note ? ` — ${i.review_note}` : ""}
                          </span>
                        )}
                      </div>
                    ),
                  },
                  {
                    id: "code",
                    header: M.colCode,
                    widthClass: "w-36",
                    hide: (i) => !i.external_code && !i.barcode,
                    cell: (i) => (
                      <span dir="ltr" className="text-xs text-ink-muted">
                        {i.external_code || i.barcode || "—"}
                      </span>
                    ),
                  },
                  {
                    id: "section",
                    header: M.colSection,
                    widthClass: "w-40",
                    cell: (i) =>
                      i.platform_section_name ? (
                        <span className="line-clamp-1 text-ink-muted">{i.platform_section_name}</span>
                      ) : (
                        <span className="text-ink-dim">{M.noSection}</span>
                      ),
                  },
                  {
                    id: "price",
                    header: M.colPrice,
                    align: "end",
                    widthClass: "w-44",
                    cell: (i) =>
                      i.price_currency === "USD" && i.merchant_price_usd != null ? (
                        <span className="flex flex-col items-end">
                          <span className="font-bold text-accent-text">{dollars(i.merchant_price_usd)}</span>
                          <span className="text-xs text-ink-muted">{fmtMoney(i.price)}</span>
                        </span>
                      ) : (
                        <span className="font-bold text-accent-text">{fmtMoney(i.price)}</span>
                      ),
                  },
                  {
                    id: "available",
                    header: M.colAvail,
                    widthClass: "w-40",
                    cell: (i) => (
                      <button
                        type="button"
                        role="switch"
                        aria-checked={i.available}
                        aria-label={`${i.name} · ${i.available ? M.available : M.unavailable}`}
                        disabled={suspended}
                        onClick={(e) => {
                          e.stopPropagation();
                          void toggle(i);
                        }}
                        data-testid="store-menu-toggle"
                        className="flex min-h-9 items-center gap-2 disabled:opacity-50"
                      >
                        <SwitchTrack on={i.available} />
                        <span className={`text-xs font-bold ${i.available ? "text-success" : "text-ink-muted"}`}>
                          {i.available ? M.available : M.unavailable}
                        </span>
                      </button>
                    ),
                  },
                ]}
                actions={(i) => (
                  <>
                    <Link
                      href={`/store/menu/${i.id}`}
                      title={M.edit}
                      aria-label={`${M.edit} · ${i.name}`}
                      className="flex size-9 items-center justify-center rounded-control text-ink-muted hover:bg-row-hover hover:text-ink"
                    >
                      <IconEdit size={18} />
                    </Link>
                    {!suspended && (
                      <button
                        type="button"
                        title={M.duplicate}
                        aria-label={`${M.duplicate} · ${i.name}`}
                        onClick={() => void duplicate(i)}
                        data-testid="store-menu-duplicate"
                        className="flex size-9 items-center justify-center rounded-control text-ink-muted hover:bg-row-hover hover:text-ink"
                      >
                        <IconCopy size={18} />
                      </button>
                    )}
                  </>
                )}
              />
            </div>
          ) : (
            <ul className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 lg:gap-4 xl:grid-cols-5 2xl:grid-cols-6">
              {items.map((it) => (
                <ItemCard key={it.id} item={it} disabled={suspended} onToggle={() => void toggle(it)} />
              ))}
            </ul>
          )}
          {/* **عدّادُ ما عُرض، وزرُّ «المزيد» احتياطٌ للصفحة التالية** — والصفحةُ تأتي وحدَها حين تُرى النهاية. */}
          {next ? (
            <div ref={sentinel} data-testid="menu-more" className="flex flex-col items-center gap-2">
              {listTotal > 0 && (
                <p className="console-meta">{fill(M.shownOf, { a: fmtNum(items.length), b: fmtNum(listTotal) })}</p>
              )}
              <LoadingState variant="list" rows={1} />
            </div>
          ) : null}
        </>
      )}
    </div>
  );
}
