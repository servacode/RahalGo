"use client";

/**
 * **صفحةُ التسوّق.**
 *
 * # الترتيبُ ولماذا هو هكذا
 *
 *   ١ · اللافتات      ← ما تُنزله المنصةُ ويُرى بلا بحث
 *   ٢ · شريطُ البحث   ← لمن يعرف ما يريد
 *   ٣ · شريطُ الأقسام ← صورٌ دائريّةٌ تمشي وحدَها
 *   ٤ · أصنافُ المختار
 *
 * **واللافتةُ صعدت فوق البحث بقرار المالك** (٢٠٢٦-٠٨-٠٦: «نبدّل بين مكان
 * السلايدر والبحث»). وكانت تحته بحجّة أنّ البحثَ هو الفعلُ الأوّل — **وهي
 * حجّةُ من يعرف ما يريد.** والزائرُ الذي لا يعرف بعدُ **يحتاج أن يُعرض عليه**،
 * واللافتةُ هي ما تعرض.
 *
 * # ولماذا الأقسامُ دوائرُ تمشي
 *
 * (قرارُ المالك: «شريطٌ أفقيٌّ لصورٍ دائريّةٍ فيه الأصناف… ومن ضغط الصورةَ ظهر
 * ما بها من عناصر».)
 *
 * **والضغطُ يُظهر الأصنافَ في مكانها لا في صفحةٍ ثانية**: من يقارن بين قسمين
 * يضغط هذا ثمّ ذاك **بلا أن يخرج من الصفحة ويعود** — وكلُّ خروجٍ يُفقده موضعَه.
 *
 * # وأوّلُ قسمٍ عامرٍ يُفتح وحدَه
 *
 * **شريطٌ من دوائرَ فوق فراغٍ يُقرأ عطباً** — والزائرُ لا يعرف أنّ عليه أن
 * يضغط. **فيُفتح أوّلُ قسمٍ فيه أصناف** فيرى السوقَ عاملاً من أوّل نظرة.
 *
 * # و«لم أصل» غيرُ «وصلتُ فلم أجد»
 *
 * **فشلُ النداء لا يُعرض فراغاً** — من فتح قسماً عامراً فرآه خالياً **ظنّ
 * السوقَ فارغاً وأغلق الصفحة.** واللافتاتُ وحدَها تُستثنى: **زينةٌ تُخفى بلا
 * ضرر.**
 */

import { useCallback, useEffect, useRef, useState } from "react";
import Link from "next/link";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Alert,
  BannerSlider,
  EmptyState,
  Input,
  IconSearch,
  IconClose,
  IconStore,
  LoadingState,
  SectionRail,
} from "@rahalgo/ui";
import ItemGrid from "@/components/ItemGrid";
import { type BrowseItem } from "@/components/ItemCard";
import { mediaUrl } from "@/lib/api";

const m = getMessages(defaultLocale);
const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

/** أقلُّ ما يُبحث به — **وحرفٌ واحدٌ يُعيد نصفَ السوق.** */
const MIN_CHARS = 2;
/** سكونُ الكتابة قبل النداء — **حدُّ ما يُحسّ تأخيراً.** */
const QUIET_MS = 300;

interface Banner {
  id: string;
  title: string;
  image_url: string | null;
  /** وجهةُ الضغط — **ولافتةٌ بلا وجهةٍ تُقرأ ولا تُفتح**، وهي حالٌ مشروعة. */
  target: string | null;
}
interface Section {
  id: string;
  name: string;
  icon: string;
  image_url: string | null;
  image_thumb_url: string | null;
  count: number;
}

export default function ShopPage() {
  const [banners, setBanners] = useState<Banner[]>([]);
  /**
   * **وإعداداتُ الجولة من اللوحة لا من الشيفرة.**
   *
   * **السرعةُ ذوقٌ يُجرَّب** — ومن أراد أن يعرف أتناسبه أربعُ ثوانٍ أم سبع
   * يجب أن يبدّلها ويرى، **لا أن ينتظر نشرةً جديدة.**
   *
   * **والافتراضُ هنا يطابق الكتالوج** (`shop.rail_auto` · `shop.rail_seconds`)
   * — **فما يُرى قبل وصول الردّ هو ما سيُرى بعده.**
   */
  const [rail, setRail] = useState({ auto: true, everyMs: 5000 });
  const [sections, setSections] = useState<Section[] | null>(null);
  const [homeFailed, setHomeFailed] = useState(false);

  const [pick, setPick] = useState<string>("");
  const [items, setItems] = useState<BrowseItem[] | null>(null);
  const [itemsFailed, setItemsFailed] = useState(false);

  const [q, setQ] = useState("");
  const [hits, setHits] = useState<BrowseItem[] | null>(null);
  const [busy, setBusy] = useState(false);
  const [searchFailed, setSearchFailed] = useState(false);

  /* ── اللافتاتُ والأقسامُ في نداءٍ واحد ─────────────────────────────────
     **`/public/home` تحملهما معاً** — ونداءان لِما يأتي في ردٍّ واحدٍ رحلةٌ
     زائدةٌ في أوّل ما يُفتح. */
  useEffect(() => {
    fetch(`${API}/api/v1/public/home`)
      .then((r) => {
        if (!r.ok) throw new Error(String(r.status));
        return r.json();
      })
      .then((j) => {
        setBanners(j.data?.banners ?? []);
        setRail({
          auto: j.data?.rail_auto ?? true,
          everyMs: j.data?.rail_every_ms || 5000,
        });
        const list: Section[] = j.data?.sections ?? [];
        setSections(list);
        // **وأوّلُ قسمٍ عامرٍ يُفتح** — ولا يُفتح فارغٌ فيُرى السوقُ ميّتاً.
        setPick((p) => p || list.find((s) => s.count > 0)?.id || list[0]?.id || "");
      })
      .catch(() => {
        setSections(null);
        setHomeFailed(true);
      });
  }, []);

  /* ── أصنافُ القسم المختار ───────────────────────────────────────────── */
  useEffect(() => {
    if (!pick) return;
    let alive = true;
    setItems(null);
    setItemsFailed(false);
    fetch(`${API}/api/v1/public/sections/${pick}/items`)
      .then((r) => {
        if (!r.ok) throw new Error(String(r.status));
        return r.json();
      })
      .then((j) => {
        if (alive) setItems(j.data?.items ?? []);
      })
      .catch(() => {
        if (!alive) return;
        setItems(null);
        setItemsFailed(true);
      });
    return () => {
      alive = false;
    };
  }, [pick]);

  /* ── البحث ──────────────────────────────────────────────────────────
     **وردٌّ متأخّرٌ لكلمةٍ قديمةٍ لا يُكتب فوق الجديدة**: من كتب «شا» ثمّ
     أتمّها «شاورما» أطلق نداءين، **ولو تأخّر الأوّلُ لَمحا نتيجةَ الثاني.** */
  const runID = useRef(0);
  const search = useCallback((term: string) => {
    const id = ++runID.current;
    setBusy(true);
    setSearchFailed(false);
    fetch(`${API}/api/v1/public/search/items?q=${encodeURIComponent(term)}`)
      .then((r) => {
        if (!r.ok) throw new Error(String(r.status));
        return r.json();
      })
      .then((j) => {
        if (id === runID.current) setHits(j.data?.items ?? []);
      })
      .catch(() => {
        if (id !== runID.current) return;
        setHits(null);
        setSearchFailed(true);
      })
      .finally(() => {
        if (id === runID.current) setBusy(false);
      });
  }, []);

  useEffect(() => {
    const term = q.trim();
    if (term.length < MIN_CHARS) {
      runID.current++;
      setHits(null);
      setBusy(false);
      setSearchFailed(false);
      return;
    }
    const t = setTimeout(() => search(term), QUIET_MS);
    return () => clearTimeout(t);
  }, [q, search]);

  const typing = q.trim().length >= MIN_CHARS;
  const active = sections?.find((s) => s.id === pick);

  return (
    /* **ولا حشوةَ هنا** — صارت مركزيّةً في `<main>` بغلاف الموقع. */
    <div>
      {/* ١ · **اللافتات** — وتُخفى أثناء البحث: من كتب كلمةً ينتظر نتيجتَها،
             **وزينةٌ بينه وبين ما طلبه تُقرأ عائقاً.** */}
      {!typing && banners.length > 0 && (
        <BannerSlider
          className="mb-5"
          Link={Link}
          items={banners.map((b) => ({
            id: b.id,
            title: b.title,
            // **والمسارُ يُحوَّل إلى رابط** — الخامُ لا يُحمَّل ويبقى إطارٌ بعنوان.
            imageUrl: mediaUrl(b.image_url) ?? null,
            href: b.target || undefined,
          }))}
        />
      )}

      {/* ٢ · **البحث** */}
      <div className="relative">
        <Input
          id="shop-search"
          icon={<IconSearch size={18} />}
          placeholder={m.site.search.itemsPlaceholder}
          value={q}
          onChange={(e) => setQ(e.target.value)}
          autoComplete="off"
        />
        {q && (
          /* **ومسحُ الحقل بضغطةٍ** — من أراد بحثاً جديداً لا يمسح حرفاً حرفاً. */
          <button
            type="button"
            onClick={() => setQ("")}
            aria-label={m.common.cancel}
            className="taparea absolute inset-block-0 end-2 my-auto flex h-8 w-8 items-center justify-center rounded-badge text-ink-muted transition-colors hover:bg-row-hover hover:text-ink"
          >
            <IconClose size={16} />
          </button>
        )}
      </div>

      {/* ── نتائجُ البحث تحجب التصفّح ما دام يُكتب ──────────────────── */}
      {typing ? (
        <div className="mt-5">
          {searchFailed ? (
            <Alert tone="warning" title={m.errors.offline}>
              {m.errors.offlineHint}
            </Alert>
          ) : hits === null || busy ? (
            <LoadingState />
          ) : hits.length === 0 ? (
            <EmptyState icon={IconSearch} title={m.site.search.empty} />
          ) : (
            <ItemGrid items={hits} next="/shop" />
          )}
        </div>
      ) : homeFailed ? (
        <div className="mt-5">
          <Alert tone="warning" title={m.errors.offline}>
            {m.errors.offlineHint}
          </Alert>
        </div>
      ) : sections === null ? (
        <div className="mt-5">
          <LoadingState />
        </div>
      ) : (
        <>
          {/* ٣ · **شريطُ الأقسام** */}
          <SectionRail
            className="mt-5"
            activeID={pick}
            onSelect={setPick}
            auto={rail.auto}
            everyMs={rail.everyMs}
            items={sections.map((s) => ({
              id: s.id,
              name: s.name,
              icon: s.icon,
              imageUrl: mediaUrl(s.image_url ?? s.image_thumb_url) ?? null,
              count: s.count,
            }))}
          />

          {/* ٤ · **أصنافُ المختار** — واسمُه فوقها فلا يضيع ما يُنظر إليه. */}
          {active && (
            <div className="mt-5">
              <h2 className="heading-card mb-3">{active.name}</h2>
              {itemsFailed ? (
                <Alert tone="warning" title={m.errors.offline}>
                  {m.errors.offlineHint}
                </Alert>
              ) : items === null ? (
                <LoadingState />
              ) : items.length === 0 ? (
                <EmptyState icon={IconStore} title={m.site.sections.empty} />
              ) : (
                <ItemGrid items={items} next="/shop" />
              )}
            </div>
          )}
        </>
      )}
    </div>
  );
}
