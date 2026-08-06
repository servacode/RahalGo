"use client";

/**
 * **صفحةُ التسوّق — شريطُ البحث أوّلاً.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٦: شريطُ البحث **داخل صفحة التسوّق فقط** لا في
 * الشريط العلويّ.)
 *
 * # ولماذا البحثُ قبل الأقسام
 *
 * **من يعرف ما يريد لا يتصفّح.** وسوقٌ فيه واحدٌ وثلاثون صنفاً في تسعة أقسامٍ
 * **يُبلَغ فيه الصنفُ المقصودُ بثلاث ضغطاتٍ أو بكلمةٍ واحدة.**
 *
 * # والبحثُ في الأصناف لا في المتاجر
 *
 * **من يشتهي «شاورما» لا يعرف اسمَ من يصنعها** — وهو لا يحتاج أن يعرف:
 * المتاجرُ محجوبةٌ عن الزبون بالكامل، والمنصةُ سوقٌ يجلب منها.
 *
 * # وثلاثةُ حُرّاسٍ في نداءٍ واحد
 *
 *   حرفان قبل النداء   ← حرفٌ واحدٌ يُعيد نصفَ السوق، **والنتيجةُ بلا معنى**
 *   سكونٌ ٣٠٠ملّي       ← كلُّ حرفٍ نداءٌ للخادم، **والكاتبُ لم يُتمّ كلمتَه**
 *   إلغاءُ ما سبق       ← ردٌّ متأخّرٌ لكلمةٍ قديمةٍ يصل بعد الجديدة **فيمحوها**
 *
 * # و«لم أصل» غيرُ «وصلتُ فلم أجد»
 *
 * **فشلُ النداء لا يُعرض «لا نتائج»** — من كتب كلمةً صحيحةً فقيل له «لا شيء»
 * **ظنّ السوقَ فارغاً وأغلق الصفحة.** والانقطاعُ يُقال لافتةً.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import Link from "next/link";
import {
  Alert,
  BannerSlider,
  EmptyState,
  Input,
  IconSearch,
  IconClose,
  LoadingState,
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

/** لافتةٌ كما يُرسلها الخادم. */
interface Banner {
  id: string;
  title: string;
  image_url: string | null;
  /** وجهةُ الضغط — **ولافتةٌ بلا وجهةٍ تُقرأ ولا تُفتح**، وهي حالٌ مشروعة. */
  target: string | null;
}

export default function ShopPage() {
  const [q, setQ] = useState("");
  const [banners, setBanners] = useState<Banner[]>([]);
  const [hits, setHits] = useState<BrowseItem[] | null>(null);
  const [busy, setBusy] = useState(false);
  /** **لم أصل** — غيرُ «وصلتُ فلم أجد»، ولكلٍّ شاشتُه. */
  const [failed, setFailed] = useState(false);

  /**
   * **وردٌّ متأخّرٌ لكلمةٍ قديمةٍ لا يُكتب فوق الجديدة.**
   *
   * من كتب «شا» ثمّ أتمّها «شاورما» أطلق نداءين. **ولو تأخّر الأوّلُ لَوصل
   * بعد الثاني فمحا نتيجتَه** — فيرى نتائجَ كلمةٍ لم يعد يكتبها.
   */
  const runID = useRef(0);

  const search = useCallback((term: string) => {
    const id = ++runID.current;
    setBusy(true);
    setFailed(false);
    fetch(`${API}/api/v1/public/search/items?q=${encodeURIComponent(term)}`)
      .then((r) => {
        if (!r.ok) throw new Error(String(r.status));
        return r.json();
      })
      .then((j) => {
        if (id !== runID.current) return;
        setHits(j.data?.items ?? []);
      })
      .catch(() => {
        if (id !== runID.current) return;
        // **ولا تُعرض «لا نتائج» على فشل** — الفراغُ كذبٌ هنا.
        setHits(null);
        setFailed(true);
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
      setFailed(false);
      return;
    }
    const t = setTimeout(() => search(term), QUIET_MS);
    return () => clearTimeout(t);
  }, [q, search]);

  /**
   * **واللافتاتُ تُجلب مرّةً عند الفتح.**
   *
   * **وفشلُها لا يُقال**: اللافتةُ زينةٌ تُخفى بلا ضرر — **والأصنافُ هي
   * المحتوى**، وسلايدرٌ غائبٌ لا يُقرأ نقصاً. (بخلاف البحث: فراغُه كذبٌ.)
   */
  useEffect(() => {
    fetch(`${API}/api/v1/public/home`)
      .then((r) => r.json())
      .then((j) => setBanners(j.data?.banners ?? []))
      // @empty-ok — **زينةٌ تُخفى بلا ضرر** (انظر أعلاه).
      .catch(() => setBanners([]));
  }, []);

  const typing = q.trim().length >= MIN_CHARS;

  return (
    <div className="px-3">
      {/* **والحقلُ يملأ العرض** — هو الفعلُ الأوّلُ في الصفحة، **وحجمُ العنصر
          يقول رتبتَه** قبل أن يُقرأ ما فيه. */}
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
          /* **ومسحُ الحقل بضغطةٍ** — من أراد بحثاً جديداً لا يمسح حرفاً حرفاً.
             و`taparea` توسّع اللمسةَ ولا تكبّر الرسم. */
          <button
            type="button"
            onClick={() => setQ("")}
            aria-label={m.common.cancel}
            className="taparea absolute inset-block-0 end-2 my-auto flex h-8 w-8 items-center justify-center rounded-badge text-ink-muted transition-colors hover:bg-page hover:text-ink"
          >
            <IconClose size={16} />
          </button>
        )}
      </div>

      {/* **واللافتاتُ تحت البحث لا فوقه.**

          **البحثُ هو الفعلُ الأوّل** — ومن يعرف ما يريد لا يتصفّح. **ولافتةٌ
          بارتفاع ١٧٥ بكسلاً فوقه تدفعه تحت الطيّة** على شاشة جوّال.

          **وتُخفى أثناء البحث**: من كتب كلمةً ينتظر نتيجتَها، **وزينةٌ بينه
          وبين ما طلبه تُقرأ عائقاً.** */}
      {!typing && banners.length > 0 && (
        <BannerSlider
          className="mt-5"
          Link={Link}
          items={banners.map((b) => ({
            id: b.id,
            title: b.title,
            // **والمسارُ يُحوَّل إلى رابط** — كان يُمرَّر خاماً في موضعٍ آخرَ
            // من قبل، **فالصورةُ لا تُحمَّل ويبقى إطارٌ رماديٌّ بعنوان.**
            imageUrl: mediaUrl(b.image_url) ?? null,
            href: b.target || undefined,
          }))}
        />
      )}

      {/* **وما دون حرفين لا يُعرض شيء** — ولا رسالةَ «اكتب أكثر»: الحقلُ
          نفسُه يقول ما يُنتظر منه، **ورسالةٌ تحته تُقرأ خطأً لا إرشاداً.** */}
      {typing && (
        <div className="mt-5">
          {failed ? (
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
      )}
    </div>
  );
}
