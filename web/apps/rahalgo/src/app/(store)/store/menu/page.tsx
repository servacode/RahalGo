"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/store/menu — «الأصناف»: التصميمُ المعتمد ٢٠٢٦-١٠-٢٩**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `MenuItemsGrid` في `ui/menu/MenuGrid.kt` — نسخةٌ واحدةٌ للمتجر
 *  والمندوب في التطبيق، وهذه مرآتُها على الويب.)
 *
 * من فوق لتحت: **البحث · صفُّ الأدوات (أضف · عرض · الأسعار · الخيارات ·
 * المشروبات) · الرقائق («الكل» · «غير المتوفر» · الأقسام) · ثمّ الأصناف**
 * بطاقتين في كلّ سطرٍ على الجوّال وأكثرَ على الحاسوب. (مراجعةُ التصميم
 * ٢٠٢٦-١٠-٢٩: كانت خمسةَ صفوفٍ قبل أوّل صنف.)
 *
 * **والبطاقةُ كلُّها تفتح المحرّر، والمفتاحُ وحدَه يبدّل التوفّر** — يتبدّل
 * فوراً ثمّ يُسأل المحرّك، **فإن ردّه عاد كما كان** وقيل السببُ كما قاله.
 * **وغيرُ المتوفّر باهت.**
 *
 * **و`?filter=unavailable` يفتحها مصفّاةً على غير المتوفرة** — زرُّ «غير
 * المتوفر» في «متجري» يصل إليها.
 */

import Link from "next/link";
import { useCallback, useEffect, useMemo, useState, type ComponentType } from "react";
import { getMessages, defaultLocale, errorText, fmtMoney, fmtNum } from "@rahalgo/i18n";
import {
  EmptyState,
  IconAdd,
  IconBanknote,
  IconCatDrinks,
  IconClose,
  IconList,
  IconOptions,
  IconPromos,
  IconSearch,
  LoadingState,
  ReloadState,
  useLiveData,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import type { MenuItem, MenuSection, SectionRef } from "@/lib/store/types";
import { dollars } from "@/lib/customer/texts";
import { useShopToast } from "@/components/customer/toast";

const m = getMessages(defaultLocale);
const M = m.store.menu;

type Icon = ComponentType<{ size?: number; className?: string }>;

function priceLine(it: MenuItem): string {
  return it.price_currency === "USD" && it.merchant_price_usd != null
    ? `${dollars(it.merchant_price_usd)} · ${fmtMoney(it.price)}`
    : fmtMoney(it.price);
}

/** **الوصفُ — أو أسماءُ مجموعات خياراته إن خلا** (مرآةُ `ItemsList` في التطبيق). */
function descOf(it: MenuItem): string {
  const d = it.description.trim();
  if (d) return d;
  return (it.modifiers ?? [])
    .map((g) => g.name.trim())
    .filter(Boolean)
    .join(" · ");
}

/**
 * **زرُّ أداةٍ صغير** — أيقونةٌ واسمُها في سطرٍ واحد، **وكلُّ الأدوات في صفٍّ واحد**
 * يُمرَّر أفقيّاً (مراجعةُ التصميم ٢٠٢٦-١٠-٢٩: خمسةُ صفوفٍ قبل أوّل صنفٍ صارت اثنين).
 */
function ToolPill({ icon: Ico, label, href, testId, primary = false }: { icon: Icon; label: string; href: string; testId: string; primary?: boolean }) {
  return (
    <Link
      href={href}
      data-testid={testId}
      className={`flex h-10 shrink-0 items-center gap-1.5 rounded-full border px-3 text-sm font-bold transition-colors ${
        primary ? "border-accent bg-accent text-on-bright hover:opacity-90" : "border-line bg-surface text-ink hover:bg-row-hover"
      }`}
    >
      <Ico size={18} className={primary ? "" : "text-primary"} />
      <span className="whitespace-nowrap">{label}</span>
    </Link>
  );
}

/** **رقاقةُ قسم** — اسمُه وعددُه؛ المختارةُ ممتلئةٌ بلون المنصّة، والباقيةُ مؤطَّرة. */
function SectionChip({
  text,
  count,
  selected,
  onClick,
  testId = "menu-chip",
}: {
  text: string;
  count: number;
  selected: boolean;
  onClick: () => void;
  testId?: string;
}) {
  return (
    <button
      type="button"
      aria-pressed={selected}
      onClick={onClick}
      data-testid={testId}
      className={`flex shrink-0 items-center gap-2 rounded-full border py-1.5 pe-1.5 ps-4 text-sm transition-colors ${
        selected ? "border-accent bg-accent font-bold text-on-bright" : "border-line bg-surface text-ink hover:bg-row-hover"
      }`}
    >
      <span className="whitespace-nowrap">{text}</span>
      <span
        className={`rounded-full px-2 py-0.5 text-2xs font-bold tabular-nums ${
          selected ? "bg-surface text-accent-text" : "bg-field text-ink-muted"
        }`}
      >
        {fmtNum(count)}
      </span>
    </button>
  );
}

/**
 * **بطاقةُ صنف** — صورتُه فوق، ثمّ اسمُه ووصفُه وسعرُه، ثمّ مفتاحُ توفّره.
 * **والبطاقةُ كلُّها تفتح المحرّر، والمفتاحُ وحدَه يبدّل التوفّر.**
 */
function ItemCard({ item, disabled, onToggle }: { item: MenuItem; disabled: boolean; onToggle: () => void }) {
  const img = mediaUrl(item.image_thumb_url ?? item.image_url);
  const dim = item.available ? "" : "opacity-50";
  const desc = descOf(item);
  return (
    <li
      className="surface flex flex-col overflow-hidden rounded-card border border-line"
      data-testid="store-menu-item"
      data-item-id={item.id}
      data-available={item.available}
    >
      <Link href={`/store/menu/${item.id}`} aria-label={`${M.edit} · ${item.name}`} data-testid="store-menu-edit" className="flex flex-1 flex-col">
        <span className={`relative block aspect-[1.15] w-full bg-field ${dim}`}>
          {img ? (
            <img src={img} alt="" loading="lazy" className="absolute inset-0 h-full w-full object-cover" />
          ) : (
            <span className="absolute inset-0 flex items-center justify-center text-3xl font-medium text-ink-muted" aria-hidden>
              {item.name.trim().charAt(0) || "?"}
            </span>
          )}
          {item.approved === false && (
            <span className="absolute start-2 top-2 rounded-full bg-accent px-2 py-0.5 text-2xs font-bold text-on-bright">
              {M.pendingReview}
            </span>
          )}
        </span>
        <span className={`flex flex-col gap-0.5 px-3 pt-2 ${dim}`}>
          <span className="line-clamp-2 min-h-[2.5rem] text-sm font-bold leading-5 text-ink">{item.name}</span>
          <span className="line-clamp-2 min-h-[2rem] text-xs leading-4 text-ink-muted">{desc}</span>
          <span className="mt-1 line-clamp-2 text-sm font-bold text-accent-text">{priceLine(item)}</span>
          {item.approved === false && item.review_note && (
            <span className="line-clamp-2 text-xs text-danger">{item.review_note}</span>
          )}
        </span>
      </Link>
      <div className="mx-3 mt-2 flex items-center gap-2 border-t border-line-soft py-1.5">
        <span className={`min-w-0 flex-1 truncate text-xs font-bold ${item.available ? "text-success" : "text-ink-muted"}`}>
          {item.available ? M.available : M.unavailable}
        </span>
        <button
          type="button"
          role="switch"
          aria-checked={item.available}
          aria-label={`${item.name} · ${item.available ? M.available : M.unavailable}`}
          disabled={disabled}
          onClick={onToggle}
          data-testid="store-menu-toggle"
          className={`relative inline-flex h-6 w-11 shrink-0 items-center rounded-badge border transition-colors disabled:opacity-50 ${
            item.available ? "border-success-edge bg-success-fill" : "border-line bg-field"
          }`}
        >
          <span
            className={`absolute h-4.5 w-4.5 rounded-badge transition-[inset-inline-start] duration-200 ${
              item.available ? "start-[1.5rem] bg-success" : "start-0.5 bg-ink-muted"
            }`}
          />
        </button>
      </div>
    </li>
  );
}

export default function MenuPage() {
  const { store } = useStore();
  const id = store?.id ?? "";
  const toast = useShopToast();
  const load = useCallback(() => api<MenuSection[]>(`/api/v1/merchant/stores/${id}/menu`), [id]);
  const { data, loading, error, reload, setData } = useLiveData(load, ["menu"], [id]);
  const [q, setQ] = useState("");
  /** **المفتوح**: `null` الكلّ، أو اسمُ قسم. */
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

  // **أقسامُ المتجر بترتيبها كما يراها الزبون** — وتعذّرُها لا يُخفي شيئاً.
  const [order, setOrder] = useState<string[]>([]);
  useEffect(() => {
    if (!id) return;
    api<{ sections: SectionRef[] }>(`/api/v1/merchant/stores/${id}/sections`)
      .then((r) => setOrder((r.sections ?? []).map((x) => x.name)))
      .catch(() => undefined);
  }, [id]);

  const all = useMemo(() => (data ?? []).flatMap((s) => s.items), [data]);
  const groups = useMemo(() => {
    const by = new Map<string, MenuItem[]>();
    for (const it of all) {
      const k = it.platform_section_name.trim() || M.unsorted;
      by.set(k, [...(by.get(k) ?? []), it]);
    }
    return [...by.entries()]
      .map(([name, items]) => ({ name, items: items.sort((a, b) => a.name.localeCompare(b.name, "ar")) }))
      .sort((a, b) => {
        const ia = order.indexOf(a.name);
        const ib = order.indexOf(b.name);
        const ra = ia < 0 ? Number.MAX_SAFE_INTEGER : ia;
        const rb = ib < 0 ? Number.MAX_SAFE_INTEGER : ib;
        return ra - rb || b.items.length - a.items.length;
      });
  }, [all, order]);

  const off = all.filter((i) => !i.available).length;
  const needle = q.trim();
  const shown = needle
    ? all.filter((i) => i.name.toLowerCase().includes(needle.toLowerCase()))
    : onlyOff
      ? all.filter((i) => !i.available)
      : open != null
        ? (groups.find((g) => g.name === open)?.items ?? [])
        : groups.flatMap((g) => g.items);

  // **ومن داخل قسمٍ يُضاف إليه** — فلا يُعيد اختيارَ ما هو فيه.
  const openSection = open != null && !onlyOff ? (groups.find((g) => g.name === open)?.items[0]?.platform_section_id ?? "") : "";

  const showAll = () => {
    setOnlyOff(false);
    setOpen(null);
  };

  async function toggle(item: MenuItem) {
    const next = !item.available;
    const flip = (v: boolean) =>
      setData((d) =>
        d ? d.map((s) => ({ ...s, items: s.items.map((i) => (i.id === item.id ? { ...i, available: v } : i)) })) : d,
      );
    flip(next);
    try {
      await api(`/api/v1/merchant/menu/items/${item.id}/availability`, {
        method: "PATCH",
        body: JSON.stringify({ available: next }),
      });
    } catch (e) {
      flip(!next);
      toast(errorText(e), "error");
    }
  }

  if (loading && !data) return <LoadingState variant="list" rows={5} />;
  if (error && !data) return <ReloadState onRetry={() => void reload()} />;

  return (
    <div className="flex flex-col gap-3" data-testid="store-menu">
      <h2 className="sr-only">{M.title}</h2>

      {/* ── ١ · البحث ── */}
      <div className="relative">
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
          className="h-12 w-full rounded-control border border-transparent bg-field pe-11 ps-10 text-sm text-ink outline-none placeholder:text-ink-muted focus:border-accent"
        />
        {q && (
          <button
            type="button"
            onClick={() => setQ("")}
            aria-label={M.clearSearch}
            className="absolute inset-y-0 end-1 flex w-10 items-center justify-center text-ink-muted hover:text-ink"
          >
            <IconClose size={18} />
          </button>
        )}
      </div>

      {/* ── ٢ · الأدواتُ في صفٍّ واحد ── */}
      {!suspended && (
        <div className="-mx-3 flex gap-2 overflow-x-auto px-3 pb-1 sm:-mx-4 sm:px-4">
          <ToolPill
            icon={IconAdd}
            label={M.add}
            href={openSection ? `/store/menu/new?section=${openSection}` : "/store/menu/new"}
            testId="menu-add"
            primary
          />
          <ToolPill icon={IconPromos} label={M.addOffer} href="/store/offers" testId="menu-offer" />
          <ToolPill icon={IconBanknote} label={M.toolBulk} href="/store/menu/prices" testId="menu-tool-prices" />
          <ToolPill icon={IconOptions} label={M.toolPresets} href="/store/menu/options" testId="menu-tool-options" />
          <ToolPill icon={IconCatDrinks} label={M.toolDrinks} href="/store/menu/drinks" testId="menu-tool-drinks" />
        </div>
      )}

      {/* ── ٣ · الرقائق: «الكل» · «غير المتوفر» · الأقسام ── */}
      {groups.length > 0 && !needle && (
        <div className="-mx-3 flex gap-2 overflow-x-auto px-3 pb-1 sm:-mx-4 sm:px-4">
          <SectionChip text={M.all} count={all.length} selected={open == null && !onlyOff} onClick={showAll} />
          <SectionChip
            text={M.unavailable}
            count={off}
            selected={onlyOff}
            testId="menu-stat-off"
            onClick={() => {
              if (onlyOff) return showAll();
              setOnlyOff(true);
              setOpen(null);
              setQ("");
            }}
          />
          {groups.map((g) => (
            <SectionChip
              key={g.name}
              text={g.name}
              count={g.items.length}
              selected={open === g.name && !onlyOff}
              onClick={() => {
                setOnlyOff(false);
                setOpen(g.name);
              }}
            />
          ))}
        </div>
      )}

      {/* ── ٤ · الأصناف ── */}
      {groups.length === 0 ? (
        <EmptyState icon={IconList} title={M.empty} />
      ) : shown.length === 0 ? (
        <p className="py-6 text-center text-sm text-ink-muted" data-testid="menu-none">
          {onlyOff && !needle ? M.offNone : M.noMatch}
        </p>
      ) : (
        <ul className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
          {shown.map((it) => (
            <ItemCard key={it.id} item={it} disabled={suspended} onToggle={() => void toggle(it)} />
          ))}
        </ul>
      )}

      {/* **مزامنةُ الأمين** — للمولات (٢٠٢٦-١٠-١٩)، ويبقى بابُها هنا. */}
      {!suspended && (
        <Link href="/store/price-sync" className="mx-auto py-2 text-xs text-ink-muted underline-offset-2 hover:underline">
          {m.store.priceSync.open}
        </Link>
      )}
    </div>
  );
}
