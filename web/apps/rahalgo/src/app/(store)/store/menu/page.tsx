"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/store/menu — «الأصناف»: لوحةُ الأصناف (٢٠٢٦-١١-٠٨)**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `MenuItemsGrid` في `ui/menu/MenuGrid.kt` — نسخةٌ واحدةٌ للمتجر والمندوب في التطبيق،
 *  وهذه مرآتُها على الويب. والتصميمُ المعتمد ٢٠٢٦-١١-٠٨: جدولٌ بمقام Shopify وقائمةٌ مُدرجةٌ بمقام iOS.)
 *
 * **حاسوباً**: رأسٌ (العنوان + العدد + أدواتُ الأصناف ▾ · أنشئ عرضاً · أضف صنفاً) · تبويباتٌ بعدّاداتها
 * (الكلّ · غير المتوفر · بلا صورة) · شريطُ بحثٍ وترشيحٍ بالقسم · **جدولٌ كثيف** (تحديدٌ · صورة واسم · رمز ·
 * قسم · سعر بالدولار والليرة · مفتاحُ توفّر · تعديل ونسخ) بشريطِ تحديدٍ جماعيّ، أو شبكةُ بطاقات.
 * **جوّالاً**: عنوانٌ كبير · بحثٌ وزرُّ تصفية (ورقةٌ بالأقسام) · مفتاحٌ مقسَّمٌ بالتبويبات ·
 * مجموعةٌ مُدرجةٌ من الأصناف **تُسحب نحو اليمين لتعديلها أو إخفائها**، و«تحديد» للعمل الجماعيّ.
 *
 * **والتحديثُ الجماعيّ يستعمل نداءاتِ محرّر الصنف نفسَها** (توفّرٌ · قسمٌ · حذف) صنفاً صنفاً، بأربعةٍ في آنٍ واحد.
 * **ومصدرُ التبويبات والقوائم الخادمُ**: ملخّصُ الأصناف ثمّ صفحاتُ ٦٠ بمفاتيحَ (`filter` · `section` · `q`).
 *
 * **والبطاقةُ/الصفُّ كلُّه يفتح المحرّر، والمفتاحُ وحدَه يبدّل التوفّر** — يتبدّل فوراً ثمّ يُسأل المحرّك،
 * **فإن ردّه عاد كما كان** وقيل السببُ كما قاله. **وغيرُ المتوفّر باهت.**
 *
 * **و`?filter=unavailable` و`?q=` يفتحانها مصفّاةً** — من «متجري» ومن بحث الشريط العلويّ.
 */

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useRef, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtMoney, fmtNum } from "@rahalgo/i18n";
import {
  ActionMenu,
  Button,
  ButtonLink,
  CheckMark,
  Confirm,
  EmptyState,
  FilterMenu,
  IGroup,
  IRow,
  IconAdd,
  IconBanknote,
  IconButton,
  IconCamera,
  IconCatDrinks,
  IconCheck,
  IconClose,
  IconCopy,
  IconDelete,
  IconEdit,
  IconFolder,
  IconGrid,
  IconList,
  IconMore,
  IconNext,
  IconOptions,
  IconPromos,
  IconRotate,
  IconRows,
  IconSearch,
  IconTag,
  IconView,
  IconViewOff,
  ImageViewer,
  LoadingState,
  Modal,
  PageHead,
  ReloadState,
  Segmented,
  Select,
  Sheet,
  SwipeRow,
  SwitchTrack,
  LineTabs,
  Thumb,
  useIsDesktop,
  useLiveData,
  useViewMode,
  type ActionMenuItem,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import { MENU_SEARCH_EVENT } from "@/lib/store/menuSearch";
import { useDrinksAllowed } from "@/lib/store/drinks";
import type { MenuItem, MenuItemsPage, MenuSummary } from "@/lib/store/types";
import { dollars, fill } from "@/lib/customer/texts";
import { useShopToast } from "@/components/customer/toast";

const m = getMessages(defaultLocale);
const M = m.store.menu;
const SH = m.store.shell;

type Tab = "all" | "off" | "photo";

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

/** **سطرٌ وصفيٌّ للصنف**: القسمُ (أو «بلا قسم») · السعر. */
function metaLine(it: MenuItem): string {
  return [it.platform_section_name || M.noSection, priceLine(it)].join(" · ");
}

/**
 * **بطاقةُ صنف (عرضُ الشبكة)** — صورتُه فوق، ثمّ اسمُه وسعرُه، ثمّ مفتاحُ توفّره.
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
      <span className="pill absolute start-2 top-2" data-tone="attention">
        {M.pendingReview}
      </span>
    ) : null;
  return (
    <li className="item-card" data-testid="store-menu-item" data-item-id={item.id} data-available={item.available}>
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
          <span className="tile-initial absolute inset-0 flex items-center justify-center text-5xl" aria-hidden>
            {item.name.trim().charAt(0) || SH.noName}
          </span>
          {badge}
        </Link>
      )}
      {img && viewing && <ImageViewer src={img} alt={item.name} onClose={() => setViewing(false)} />}
      <Link href={`/store/menu/${item.id}`} aria-label={`${M.edit} · ${item.name}`} data-testid="store-menu-edit" className="flex flex-1 flex-col">
        <span className={`flex flex-col gap-0.5 px-3 pt-3 ${dim}`}>
          <span className="line-clamp-2 min-h-10 text-sm font-bold leading-5 text-ink">{item.name}</span>
          <span className="line-clamp-2 min-h-8 text-xs leading-4 text-ink-dim">{desc}</span>
          <span className="mt-1 line-clamp-2 text-base font-bold tabular-nums text-accent-text">{priceLine(item)}</span>
          {item.approved === false && item.review_note && <span className="line-clamp-2 text-xs text-danger">{item.review_note}</span>}
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
        <span className={`min-w-0 flex-1 truncate text-xs font-bold ${item.available ? "text-success" : "text-ink-dim"}`}>
          {item.available ? M.available : M.unavailable}
        </span>
        <SwitchTrack on={item.available} />
      </button>
    </li>
  );
}

/** **صورةٌ مصغّرةٌ في صفّ** — لمستُها تكبّرها، ومن غيرها مربّعٌ متقطّعٌ بأيقونة «بلا صورة». */
function RowThumb({ item, size }: { item: MenuItem; size: number }) {
  const img = mediaUrl(item.image_thumb_url || item.image_url);
  const [viewing, setViewing] = useState(false);
  return (
    <>
      {img ? (
        <button
          type="button"
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            setViewing(true);
          }}
          aria-label={`${item.name} — ${m.shop.item.zoomBtn}`}
          className={item.available ? "" : "opacity-50"}
        >
          <Thumb src={img} size={size} />
        </button>
      ) : (
        <span title={M.noPhoto} aria-label={M.noPhoto} className={item.available ? "" : "opacity-60"}>
          <Thumb size={size} />
        </span>
      )}
      {img && viewing && <ImageViewer src={img} alt={item.name} onClose={() => setViewing(false)} />}
    </>
  );
}

/** **Page size of the paged menu** — 60 cards, a few tens of KB even for a 15,000-item mall. */
const PAGE = 60;

/** **تحديثٌ لعدّة أصناف** — بأربعةٍ في آنٍ واحدٍ، ويُعاد ما نجح وعددُ ما فشل. */
async function inChunks(ids: string[], run: (id: string) => Promise<void>): Promise<{ ok: string[]; failed: number }> {
  const ok: string[] = [];
  let failed = 0;
  for (let i = 0; i < ids.length; i += 4) {
    const chunk = ids.slice(i, i + 4);
    const res = await Promise.allSettled(chunk.map((id) => run(id)));
    res.forEach((r, k) => {
      const id = chunk[k];
      if (r.status === "fulfilled" && id) ok.push(id);
      else failed += 1;
    });
  }
  return { ok, failed };
}

/**
 * **The paged menu** (owner 2026-11-07: «ما يصير يكون بطيء») — mirror of `MenuViewModel` in the
 * merchant app. A mall of 5,311 items made the full menu 3 MB (45 s on a weak link), so the page
 * reads a light summary (counts and sections) and then pages of 60 items; the tabs, the section filter
 * and the search ask the server, and the next page comes when the end of the list shows.
 */
export default function MenuPage() {
  const { store } = useStore();
  const id = store?.id ?? "";
  // **أداةُ المشروبات للمطاعم وحدَها** (قرارُ المالك ٢٠٢٦-١٠-٣٠).
  const drinksAllowed = useDrinksAllowed(id);
  const toast = useShopToast();
  const router = useRouter();
  const desk = useIsDesktop();
  /** **جدولٌ على الحاسوب افتراضاً** (آلافُ الأصناف)؛ والجوّالُ قائمةٌ دائماً؛ والاختيارُ محفوظ. */
  const [view, setView] = useViewMode("store-menu", "table");
  const dupBusy = useRef(false);
  const load = useCallback(() => api<MenuSummary>(`/api/v1/merchant/stores/${id}/menu/summary`), [id]);
  const { data, loading, error, reload, setData } = useLiveData(load, ["menu"], [id]);
  const [q, setQ] = useState("");
  /** **The search sent to the server** — 350 ms after the last keystroke. */
  const [needle, setNeedle] = useState("");
  useEffect(() => {
    const t = setTimeout(() => setNeedle(q.trim()), 350);
    return () => clearTimeout(t);
  }, [q]);
  /** **القسمُ المفتوح**: `""` الكلّ، أو معرّفُ قسم. */
  const [section, setSection] = useState("");
  const [tab, setTab] = useState<Tab>("all");
  const [sheet, setSheet] = useState(false);
  const suspended = store?.status === "suspended";

  // **`?filter=unavailable` و`?q=`** — من «متجري» ومن بحث الشريط العلويّ.
  useEffect(() => {
    try {
      const p = new URLSearchParams(window.location.search);
      if (p.get("filter") === "unavailable") setTab("off");
      const qq = p.get("q");
      if (qq) setQ(qq.slice(0, 60));
    } catch {
      /* لا عنوانَ يُقرأ — تبقى «الكل». */
    }
  }, []);

  // **بحثُ الشريط العلويّ وأنا في الصفحة** — يصل حدثاً (انظر `menuSearch.ts`).
  useEffect(() => {
    const on = (e: Event) => {
      const t = (e as CustomEvent<string>).detail;
      if (typeof t === "string") setQ(t.slice(0, 60));
    };
    window.addEventListener(MENU_SEARCH_EVENT, on);
    return () => window.removeEventListener(MENU_SEARCH_EVENT, on);
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
      if (tab === "off") p.set("filter", "unavailable");
      else if (tab === "photo") p.set("filter", "no_photo");
      if (section) p.set("section", section);
      if (after) p.set("after", after);
      return `/api/v1/merchant/stores/${id}/menu/items?${p.toString()}`;
    },
    [id, needle, tab, section],
  );

  // **الاختيارُ يُمسح مع كلّ ترشيحٍ جديد** — فلا يبقى تحديدٌ لما لم يعد معروضاً.
  const [picked, setPicked] = useState<Set<string>>(new Set());
  const [selecting, setSelecting] = useState(false);
  useEffect(() => {
    setPicked(new Set());
  }, [needle, tab, section]);

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

  // **The next page when the end of the list comes into view.**
  const sentinel = useRef<HTMLDivElement | null>(null);
  useEffect(() => {
    const el = sentinel.current;
    if (!el || !next) return;
    const io = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting)) more();
      },
      { rootMargin: "600px" },
    );
    io.observe(el);
    return () => io.disconnect();
  }, [next, more]);

  const sections = data?.sections ?? [];
  const total = data?.total ?? 0;
  const off = data?.unavailable ?? 0;
  const noPhoto = data?.no_photo ?? 0;
  const filtered = needle !== "" || tab !== "all" || section !== "";
  const shown = filtered ? listTotal || items.length : total;

  /** **والعدّاداتُ تتبع فوراً** — «غير المتوفر» ورقاقةُ القسم. */
  function bump(item: MenuItem, v: boolean) {
    const d = v === item.available ? 0 : v ? -1 : 1;
    setItems((list) => list.map((i) => (i.id === item.id ? { ...i, available: v } : i)));
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
  }

  async function toggle(item: MenuItem) {
    const want = !item.available;
    bump(item, want);
    try {
      await api(`/api/v1/merchant/menu/items/${item.id}/availability`, {
        method: "PATCH",
        body: JSON.stringify({ available: want }),
      });
    } catch (e) {
      bump({ ...item, available: want }, item.available);
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

  // ══════════════════════════════════════════════════════════════════
  // **التحديدُ والتحديثُ الجماعيّ** — النداءاتُ نفسُها التي يستعملها محرّرُ الصنف، صنفاً صنفاً
  // ══════════════════════════════════════════════════════════════════
  const [bulkBusy, setBulkBusy] = useState(false);
  const [askDelete, setAskDelete] = useState(false);
  const [askSection, setAskSection] = useState(false);
  const [sectionPick, setSectionPick] = useState("");
  const chosen = items.filter((i) => picked.has(i.id));

  const flipPick = (idd: string) =>
    setPicked((s) => {
      const n = new Set(s);
      if (n.has(idd)) n.delete(idd);
      else n.add(idd);
      return n;
    });
  const allState: boolean | "mixed" = picked.size === 0 ? false : picked.size === items.length ? true : "mixed";
  const flipAll = () => setPicked(allState === true ? new Set() : new Set(items.map((i) => i.id)));
  const clearPick = () => {
    setPicked(new Set());
    setSelecting(false);
  };

  function report(res: { ok: string[]; failed: number }) {
    if (res.ok.length > 0) toast(fill(SH.bulkOk, { n: fmtNum(res.ok.length) }), "success");
    if (res.failed > 0) toast(fill(SH.bulkFail, { n: fmtNum(res.failed) }), "error");
  }

  async function bulkAvail(available: boolean) {
    const targets = chosen.filter((i) => i.available !== available);
    if (targets.length === 0) return clearPick();
    setBulkBusy(true);
    try {
      const byId = new Map(targets.map((i) => [i.id, i]));
      const res = await inChunks(
        targets.map((i) => i.id),
        async (itemId) => {
          await api(`/api/v1/merchant/menu/items/${itemId}/availability`, { method: "PATCH", body: JSON.stringify({ available }) });
          const it = byId.get(itemId);
          if (it) bump(it, available);
        },
      );
      report(res);
    } finally {
      setBulkBusy(false);
      clearPick();
    }
  }

  async function bulkSection() {
    const sec = sections.find((s) => s.id === sectionPick);
    if (!sec) return;
    setBulkBusy(true);
    try {
      const res = await inChunks(
        chosen.map((i) => i.id),
        async (itemId) => {
          await api(`/api/v1/merchant/menu/items/${itemId}`, { method: "PATCH", body: JSON.stringify({ platform_section_id: sec.id }) });
        },
      );
      const done = new Set(res.ok);
      setItems((list) => list.map((i) => (done.has(i.id) ? { ...i, platform_section_id: sec.id, platform_section_name: sec.name } : i)));
      report(res);
      void reload();
    } finally {
      setBulkBusy(false);
      setAskSection(false);
      clearPick();
    }
  }

  async function bulkDelete() {
    setBulkBusy(true);
    try {
      const res = await inChunks(
        chosen.map((i) => i.id),
        async (itemId) => {
          await api(`/api/v1/merchant/menu/items/${itemId}`, { method: "DELETE" });
        },
      );
      const gone = new Set(res.ok);
      setItems((list) => list.filter((i) => !gone.has(i.id)));
      report(res);
      void reload();
    } finally {
      setBulkBusy(false);
      setAskDelete(false);
      clearPick();
    }
  }

  // ── أدواتُ المتجر في قائمةٍ واحدة ──
  const tools: ActionMenuItem[] = suspended
    ? []
    : [
        { key: "prices", label: M.toolBulk, icon: IconBanknote, onSelect: () => router.push("/store/menu/prices") },
        { key: "options", label: M.toolPresets, icon: IconOptions, onSelect: () => router.push("/store/menu/options") },
        ...(noPhoto > 0
          ? [{ key: "photos", label: fill(M.toolQuickPhoto, { n: fmtNum(noPhoto) }), icon: IconCamera, onSelect: () => router.push("/store/menu/photos") }]
          : []),
        ...(drinksAllowed === true ? [{ key: "drinks", label: M.toolDrinks, icon: IconCatDrinks, onSelect: () => router.push("/store/menu/drinks") }] : []),
        ...(store?.uses_ameen === true
          ? [{ key: "sync", label: m.store.priceSync.open, icon: IconRotate, onSelect: () => router.push("/store/price-sync") }]
          : []),
      ];

  if (loading && !data) return <LoadingState variant="list" rows={5} />;
  if (error && !data) return <ReloadState onRetry={() => void reload()} />;

  const asTable = view === "table";
  const sectionOptions = [{ key: "", label: M.bulk.allSections }, ...sections.map((s) => ({ key: s.id, label: s.name, count: s.count }))];
  const addHref = section ? `/store/menu/new?section=${section}` : "/store/menu/new";

  const bulkButtons = (
    <>
      <Button size="sm" variant="secondary" disabled={bulkBusy} onClick={() => setAskSection(true)}>
        <IconFolder size={15} />
        {SH.bulkSection}
      </Button>
      <Button size="sm" variant="secondary" disabled={bulkBusy} onClick={() => void bulkAvail(true)}>
        <IconCheck size={15} />
        {M.available}
      </Button>
      <Button size="sm" variant="secondary" disabled={bulkBusy} onClick={() => void bulkAvail(false)}>
        <IconViewOff size={15} />
        {M.unavailable}
      </Button>
      <Button size="sm" variant="secondary" disabled={bulkBusy} onClick={() => router.push("/store/menu/prices")}>
        <IconTag size={15} />
        {M.editPrice}
      </Button>
      <Button size="sm" variant="danger-outline" disabled={bulkBusy} onClick={() => setAskDelete(true)}>
        <IconDelete size={15} />
        {m.common.delete}
      </Button>
    </>
  );

  const dialogs = (
    <>
      <Confirm
        open={askDelete}
        title={fill(SH.bulkDeleteQ, { n: fmtNum(picked.size) })}
        body={SH.bulkDeleteBody}
        confirmLabel={m.common.delete}
        busy={bulkBusy}
        onConfirm={() => void bulkDelete()}
        onCancel={() => setAskDelete(false)}
      />
      <Modal open={askSection} onClose={() => setAskSection(false)} title={SH.bulkSection}>
        <div className="flex flex-col gap-3">
          <Select id="bulk-section" label={SH.pickSection} value={sectionPick} onChange={(e) => setSectionPick(e.target.value)}>
            <option value="">{SH.pickSection}</option>
            {sections.map((s) => (
              <option key={s.id} value={s.id}>
                {s.name}
              </option>
            ))}
          </Select>
          <Button disabled={!sectionPick || bulkBusy} onClick={() => void bulkSection()}>
            {m.common.confirm}
          </Button>
        </div>
      </Modal>
    </>
  );

  const empty = (
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
  );
  const none = (
    <p className="py-8 text-center text-sm text-ink-dim" data-testid="menu-none">
      {tab === "off" && !needle ? M.offNone : M.noMatch}
    </p>
  );
  const sentinelNode = next ? (
    <div ref={sentinel} data-testid="menu-more" className="flex flex-col items-center gap-2">
      <LoadingState variant="list" rows={1} />
    </div>
  ) : null;

  // ══════════════════════════ الحاسوب ══════════════════════════
  if (desk) {
    const tabs = [
      { key: "all" as const, label: M.all, count: total },
      { key: "off" as const, label: M.unavailable, count: off },
      { key: "photo" as const, label: M.noPhoto, count: noPhoto, warn: noPhoto > 0 },
    ];
    return (
      <div className="console-page" data-testid="store-menu">
        <PageHead
          title={M.title}
          subtitle={fill(M.count, { n: fmtNum(total) })}
          actions={
            !suspended && (
              <>
                {tools.length > 0 && <ActionMenu variant="secondary" label={SH.toolsMenu} icon={IconOptions} items={tools} width={260} />}
                <ButtonLink Link={Link} href="/store/offers?new=1" variant="secondary" data-testid="menu-offer">
                  <IconPromos size={15} />
                  {M.addOffer}
                </ButtonLink>
                <ButtonLink Link={Link} href={addHref} data-testid="menu-add">
                  <IconAdd size={15} />
                  {M.add}
                </ButtonLink>
              </>
            )
          }
        />

        {/* ── التبويبات: الكلّ · غير المتوفر · بلا صورة (كلُّها من الخادم) ── */}
        <LineTabs items={tabs} value={tab} onChange={(k) => setTab(k)} ariaLabel={M.filters} />

        {/* ── البحث والترشيح ── */}
        <div className="toolbar">
          <div className="relative w-80 max-w-full">
            <span className="pointer-events-none absolute inset-y-0 start-3 flex items-center text-ink-dim" aria-hidden>
              <IconSearch size={16} />
            </span>
            <input
              type="search"
              value={q}
              onChange={(e) => setQ(e.target.value.slice(0, 60))}
              placeholder={SH.search}
              aria-label={M.search}
              data-testid="store-menu-search"
              className="search-field"
            />
            {q && (
              <button type="button" onClick={() => setQ("")} aria-label={M.clearSearch} className="absolute inset-y-0 end-1 flex w-8 items-center justify-center text-ink-dim hover:text-ink">
                <IconClose size={15} />
              </button>
            )}
          </div>
          {sections.length > 0 && (
            <FilterMenu label={M.colSection} icon={IconFolder} options={sectionOptions} value={section} onChange={setSection} testId="menu-section-filter" />
          )}
          {filtered && (
            <Button
              size="sm"
              variant="ghost"
              onClick={() => {
                setQ("");
                setTab("all");
                setSection("");
              }}
            >
              <IconClose size={14} />
              {SH.reset}
            </Button>
          )}
          <div className="ms-auto flex items-center gap-2">
            <Segmented
              ariaLabel={M.viewTable}
              value={view}
              onChange={setView}
              items={[
                { key: "table", label: <IconRows size={15} />, title: M.viewTable },
                { key: "cards", label: <IconGrid size={15} />, title: M.viewGrid },
              ]}
            />
          </div>
        </div>

        {/* ── الأصناف ── */}
        {total === 0 ? (
          empty
        ) : items.length === 0 && listLoading ? (
          <LoadingState variant="list" rows={3} />
        ) : items.length === 0 && listFailed ? (
          <ReloadState onRetry={() => setRev((r) => r + 1)} />
        ) : items.length === 0 ? (
          none
        ) : asTable ? (
          <div className="panel overflow-clip" data-testid="store-menu-table">
            {picked.size > 0 && (
              <div className="bulk-bar" data-testid="menu-bulk">
                <CheckMark state={allState} onChange={flipAll} label={SH.selectAll} />
                <span className="text-primary">{fill(SH.selectedCount, { n: fmtNum(picked.size) })}</span>
                <span className="mx-1 h-5 w-px bg-primary-edge" aria-hidden />
                {bulkButtons}
                <Button size="sm" variant="ghost" className="ms-auto" onClick={clearPick}>
                  <IconClose size={14} />
                  {SH.bulkClear}
                </Button>
              </div>
            )}
            <div className="overflow-x-auto lg:overflow-visible">
              <table className="console-table">
                <thead>
                  <tr>
                    <th className="w-11 !pe-0">
                      <CheckMark state={allState} onChange={flipAll} label={SH.selectAll} />
                    </th>
                    <th>{M.colName}</th>
                    <th>{M.colCode}</th>
                    <th>{M.colSection}</th>
                    <th>{M.colPrice}</th>
                    <th>{M.colAvail}</th>
                    <th className="w-24" />
                  </tr>
                </thead>
                <tbody>
                  {items.map((i) => (
                    <tr key={i.id} data-selected={picked.has(i.id)} data-testid="store-menu-item" data-item-id={i.id} data-available={i.available}>
                      <td className="w-11 !pe-0">
                        <CheckMark state={picked.has(i.id)} onChange={() => flipPick(i.id)} label={i.name} />
                      </td>
                      <td className="w-[40%] min-w-[16rem]">
                        <div className={`flex min-w-0 items-center gap-3 ${i.available ? "" : "opacity-60"}`}>
                          <RowThumb item={i} size={36} />
                          <div className="min-w-0">
                            <Link
                              href={`/store/menu/${i.id}`}
                              aria-label={`${M.edit} · ${i.name}`}
                              data-testid="store-menu-edit"
                              className="line-clamp-2 font-bold hover:underline"
                            >
                              {i.name}
                            </Link>
                            {i.approved === false ? (
                              <span className="block text-xs font-bold text-warning">
                                {M.pendingReview}
                                {i.review_note ? ` — ${i.review_note}` : ""}
                              </span>
                            ) : descOf(i) ? (
                              <span className="line-clamp-1 block text-xs text-ink-dim">{descOf(i)}</span>
                            ) : !i.image_thumb_url && !i.image_url ? (
                              <span className="block text-xs text-warning">{M.noPhoto}</span>
                            ) : null}
                          </div>
                        </div>
                      </td>
                      <td>
                        <span dir="ltr" className="inline-block text-xs text-ink-muted">
                          {i.external_code || i.barcode || "—"}
                        </span>
                      </td>
                      <td>
                        {i.platform_section_name ? (
                          <span className="line-clamp-1">{i.platform_section_name}</span>
                        ) : (
                          <span className="pill" data-tone="attention">
                            {M.noSection}
                          </span>
                        )}
                      </td>
                      <td className="whitespace-nowrap">
                        {i.price_currency === "USD" && i.merchant_price_usd != null ? (
                          <>
                            <b>{dollars(i.merchant_price_usd)}</b>
                            <span className="ms-1.5 text-xs text-ink-dim">· {fmtMoney(i.price)}</span>
                          </>
                        ) : (
                          <b>{fmtMoney(i.price)}</b>
                        )}
                      </td>
                      <td>
                        <button
                          type="button"
                          role="switch"
                          aria-checked={i.available}
                          aria-label={`${i.name} · ${i.available ? M.available : M.unavailable}`}
                          disabled={suspended}
                          onClick={() => void toggle(i)}
                          data-testid="store-menu-toggle"
                          className="flex min-h-9 items-center gap-2.5 whitespace-nowrap text-sm disabled:opacity-50"
                        >
                          <SwitchTrack on={i.available} size="sm" />
                          <span className={i.available ? "" : "text-ink-dim"}>{i.available ? M.available : M.unavailable}</span>
                        </button>
                      </td>
                      <td>
                        <div className="flex justify-end gap-0.5">
                          <IconButton icon={IconEdit} label={`${M.edit} · ${i.name}`} href={`/store/menu/${i.id}`} Link={Link} size={16} />
                          {!suspended && (
                            <IconButton icon={IconCopy} label={`${M.duplicate} · ${i.name}`} onClick={() => void duplicate(i)} testId="store-menu-duplicate" size={16} />
                          )}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <div className="table-foot">
              <span>{fill(M.shownOf, { a: fmtNum(items.length), b: fmtNum(shown) })}</span>
              {next && (
                <Button size="sm" variant="secondary" className="ms-auto" onClick={more} data-testid="menu-more-btn">
                  {SH.rowMore}
                </Button>
              )}
            </div>
          </div>
        ) : (
          <ul className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4 lg:gap-4 xl:grid-cols-5 2xl:grid-cols-6">
            {items.map((it) => (
              <ItemCard key={it.id} item={it} disabled={suspended} onToggle={() => void toggle(it)} />
            ))}
          </ul>
        )}
        {sentinelNode}
        {dialogs}
      </div>
    );
  }

  // ══════════════════════════ الهاتف ══════════════════════════
  const segItems = [
    { key: "all" as const, label: M.all },
    { key: "off" as const, label: M.unavailable },
    { key: "photo" as const, label: M.noPhoto },
  ];
  return (
    <div className="console-page" data-testid="store-menu">
      <PageHead
        title={M.title}
        navStart={
          <button type="button" className="text-base font-medium" onClick={() => (selecting ? clearPick() : setSelecting(true))} data-testid="menu-select">
            {selecting ? SH.selectDone : SH.select}
          </button>
        }
        navEnd={
          !suspended && (
            <>
              {tools.length > 0 && <ActionMenu variant="ghost" size="sm" chevron={false} label={<IconMore size={22} />} ariaLabel={SH.toolsMenu} items={tools} width={240} />}
              <IconButton icon={IconAdd} label={M.add} href={addHref} Link={Link} size={24} testId="menu-add" />
            </>
          )
        }
      />
      <div className="flex gap-2.5">
        <label className="relative min-w-0 flex-1">
          <span className="pointer-events-none absolute inset-y-0 start-2.5 flex items-center text-ink-dim" aria-hidden>
            <IconSearch size={17} />
          </span>
          <input
            type="search"
            value={q}
            onChange={(e) => setQ(e.target.value.slice(0, 60))}
            placeholder={fill(SH.searchN, { n: fmtNum(total) })}
            aria-label={M.search}
            data-testid="store-menu-search"
            className="search-field"
          />
        </label>
        {sections.length > 0 && (
          <button
            type="button"
            onClick={() => setSheet(true)}
            aria-label={SH.filtersTitle}
            data-testid="menu-filter-open"
            className="relative flex size-10 shrink-0 items-center justify-center rounded-control bg-field text-primary"
          >
            <IconOptions size={19} />
            {section !== "" && <span className="absolute -start-1 -top-1 grid size-4 place-items-center rounded-full bg-primary text-2xs font-bold text-on-bright">1</span>}
          </button>
        )}
      </div>
      <div>
        <Segmented ariaLabel={M.filters} value={tab} onChange={setTab} items={segItems} block />
      </div>
      <div className="flex items-center justify-between px-5 text-xs text-ink-dim">
        <span>{fill(M.count, { n: fmtNum(shown) })}</span>
        {section !== "" && (
          <button type="button" className="font-medium text-primary" onClick={() => setSection("")}>
            {sections.find((s) => s.id === section)?.name}
          </button>
        )}
      </div>

      {total === 0 ? (
        empty
      ) : items.length === 0 && listLoading ? (
        <LoadingState variant="list" rows={3} />
      ) : items.length === 0 && listFailed ? (
        <ReloadState onRetry={() => setRev((r) => r + 1)} />
      ) : items.length === 0 ? (
        none
      ) : (
        <div className="igroup md:max-lg:columns-2 md:max-lg:gap-0 md:max-lg:[column-rule:1px_solid_var(--color-line-soft)] md:max-lg:[&>*]:break-inside-avoid" data-testid="store-menu-list">
          {items.map((i) => {
            const row = (
              <div className="irow gap-3 ps-4 pe-3" style={{ minHeight: "4.125rem" }} data-testid="store-menu-item" data-item-id={i.id} data-available={i.available}>
                {selecting && <CheckMark state={picked.has(i.id)} onChange={() => flipPick(i.id)} label={i.name} />}
                <Link href={`/store/menu/${i.id}`} className="flex min-w-0 flex-1 items-center gap-3 self-stretch" data-testid="store-menu-edit" aria-label={`${M.edit} · ${i.name}`}>
                  <RowThumb item={i} size={46} />
                  <span className={`min-w-0 flex-1 py-2 ${i.available ? "" : "opacity-60"}`}>
                    <span className="line-clamp-2 block break-words text-base font-bold leading-snug [text-wrap:balance]">{i.name}</span>
                    <span className="console-meta block [text-wrap:balance]">{metaLine(i)}</span>
                  </span>
                </Link>
                <button
                  type="button"
                  role="switch"
                  aria-checked={i.available}
                  aria-label={`${i.name} · ${i.available ? M.available : M.unavailable}`}
                  disabled={suspended}
                  onClick={() => void toggle(i)}
                  data-testid="store-menu-toggle"
                  className="flex min-h-11 min-w-11 items-center justify-center text-xs disabled:opacity-50"
                >
                  {i.available ? (
                    <span className="size-2.5 rounded-full bg-success-solid" aria-label={M.available} />
                  ) : (
                    <span className="pill">{M.unavailable}</span>
                  )}
                </button>
                <IconNext size={14} className="shrink-0 text-ink-dim opacity-70" />
              </div>
            );
            return suspended || selecting ? (
              <div key={i.id}>{row}</div>
            ) : (
              <SwipeRow
                key={i.id}
                actions={[
                  { key: "edit", label: SH.swipeEdit, icon: IconEdit, tone: "info", onClick: () => router.push(`/store/menu/${i.id}`) },
                  { key: "hide", label: i.available ? SH.swipeHide : SH.swipeShow, icon: i.available ? IconViewOff : IconView, tone: "attention", onClick: () => void toggle(i) },
                ]}
              >
                {row}
              </SwipeRow>
            );
          })}
        </div>
      )}
      {sentinelNode}
      {items.length > 0 && !selecting && <p className="sec-foot text-center">{SH.swipeHint}</p>}

      {selecting && picked.size > 0 && (
        <div
          className="fixed inset-x-3 z-30 flex flex-wrap items-center justify-center gap-2 rounded-2xl border border-line-soft bg-raised p-2.5 shadow-e3"
          style={{ bottom: "calc(4.25rem + env(safe-area-inset-bottom))" }}
          data-testid="menu-bulk"
        >
          <b className="w-full text-center text-sm">{fill(SH.selectedCount, { n: fmtNum(picked.size) })}</b>
          {bulkButtons}
        </div>
      )}

      <Sheet open={sheet} onClose={() => setSheet(false)} title={SH.filtersTitle}>
        <div className="-mx-1 flex flex-col gap-3">
          <IGroup title={M.colSection}>
            {sectionOptions.map((o) => (
              <IRow
                key={o.key}
                title={o.label}
                value={"count" in o && o.count != null ? fmtNum(o.count) : undefined}
                trailing={o.key === section ? <IconCheck size={18} className="text-primary" /> : undefined}
                chevron={false}
                onClick={() => setSection(o.key)}
              />
            ))}
          </IGroup>
          <Button size="lg" onClick={() => setSheet(false)}>
            {fill(SH.showN, { n: fmtNum(shown) })}
          </Button>
        </div>
      </Sheet>
      {dialogs}
    </div>
  );
}
