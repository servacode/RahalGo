"use client";

/**
 * **صفحةُ القسم — بضاعةُ المتاجر مجموعةً في بابٍ واحد.**
 *
 * # لماذا صفحةٌ لا نافذة
 *
 * قرارُ المالك (٢٠٢٦-٠٨-٠٤): «عرضُ القسم يجب أن يفتح صفحةً منفصلة وليس نافذةً
 * منبثقة».
 *
 * **والنافذةُ تُخفي ما خلفها**: من يقارن قسمين يفتح واحدةً ويُغلقها ليفتح
 * الأخرى. **ولا تُشارَك برابط**: من أراد أن يقول «انظر هذا القسم» لا يملك
 * عنواناً يرسله. **ولا يعود إليها زرُّ الرجوع** — يُغلق الصفحةَ كلَّها.
 *
 * **وقائمةٌ قد تبلغ خمسمئةِ صنفٍ ليست محتوى نافذة.**
 *
 * # ولماذا «كما هي»
 *
 * نقطةُ التصفّح العامّة تُرشِّح: متجرٌ فعّالٌ وقسمٌ فعّالٌ وصنفٌ مُقَرّ. **وهي
 * الصواب للزبون وخطأٌ للإدارة**: من يفتح قسماً ليقرّر إطفاءَه يريد ما فيه
 * كلَّه — **بما لا يظهر ولماذا لا يظهر.**
 *
 * # والصنفُ يأتي من متجر — من البابين
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «كلُّ شيءٍ يُضاف بالمتاجر يجب أن يأتي إلى هنا
 * تلقائياً، أو من هنا نضيف عنصراً جديداً ونحدّد هو تابعٌ لأيّ متجر».)
 *
 * **ولا صنفَ تملكه المنصة**: كلُّ صفٍّ هنا له `merchant_id`، **وإضافةٌ من هذه
 * الصفحة تسأل عن المتجر أوّلاً** — لا لتزيد خطوة، **بل لأنّ صنفاً بلا مصدرٍ
 * لا يُطبخ ولا يُحاسَب عليه أحد.**
 */

import { useCallback, useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, errorText } from "@rahalgo/i18n";
import {
  Input,
  Select,
  PageContainer,
  Pagination,
  EmptyState,
  LoadingState,
  IconStore,
  IconPrev,
  IconSearch,
  IconOrder,
  IconWarning,
  IconCamera,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import StateBadge from "@/components/admin/market/StateBadge";
import { NotAllowed } from "@/components/admin/accounts/ProfileParts";
import { useCanCall } from "@/lib/policy";

const m = getMessages(defaultLocale);
const S = m.admin.sections;

interface Section {
  id: string;
  name: string;
  active: boolean;
  items: number;
  image_url: string | null;
  image_thumb_url: string | null;
}

interface SectionItem {
  id: string;
  name: string;
  /** **سعرُ الشراء** — ما وضعه المتجر، وأصلُ الحسبتين. */
  merchant_price: number;
  available: boolean;
  merchant_name: string;
  merchant_status: string;
  thumb_url: string | null;
  image_url: string | null;
  /**
   * **الحسبتان — واحدةٌ تنزل وأخرى تصعد.** (والقسمُ صار عرضاً بلا أسعار — قرارُ المالك ٢٠٢٦-١٠-٢٥؛
   * الحقولُ باقيةٌ في العقد لا تُعرض هنا.)
   *
   *	عمولةُ المنصة  ←  تُقتطع من المتجر   ←  `merchant_net` ما يقبضه
   *	هامشُ المنصة   ←  يُضاف على الزبون   ←  `sale_price`  ما يدفعه
   *
   * **ولا تُجمعان في رقمٍ واحد**: من رأى «ربحُ المنصة ٣٬٦٠٠» لا يعرف أيُّهما
   * يُعدَّل حين يشتكي المتجرُ أو يشتكي الزبون. **وربحُ المنصة مجموعُهما.**
   */
  commission_percent: number;
  /** **بمنزلتين** (١٫٥) — و`commission_percent` مقرَّبةٌ (٢٠٢٦-١٠-١٨). */
  commission_percent_exact?: number;
  commission: number;
  merchant_net: number;
  /** هامشُ الصنف النافذ بالليرة — **بعد الوراثة**. */
  margin_value: number;
  margin: number;
  sale_price: number;
  margin_override: number | null;
}

/**
 * **حالُ الصنف — وسببُ ظهوره أو غيابه.**
 *
 * **والترتيبُ مقصود**: يُقرأ أوّلُ سببٍ يمنع الظهور. **ولا «بانتظار المراجعة»**
 * — رُفعت المراجعة (قرارُ المالك ٢٠٢٦-١٠-٠٤).
 */
function itemState(it: SectionItem): { label: string; variant: "success" | "warning" | "danger" } {
  if (it.merchant_status !== "active") return { label: S.itemStoreOff, variant: "danger" };
  if (!it.available) return { label: S.itemOut, variant: "warning" };
  return { label: S.itemLive, variant: "success" };
}

export default function SectionPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const [sec, setSec] = useState<Section | null>(null);
  const [rows, setRows] = useState<SectionItem[] | null>(null);
  /** **صفحةُ أصناف القسم** — (قرارُ المالك ٢٠٢٦-٠٨-١٠).

      **والقسمُ يجمع أصنافَ كلّ المتاجر** — ينمو بعدد المتاجر لا بعدد
      الأقسام. **وخمسُمئةٍ صامتةٌ تعني أنّ صنفاً لا يُوافَق عليه لأنّ أحداً
      لم يره.** */
  const [page, setPage] = useState(1);
  const [count, setCount] = useState(0);
  const [perPage, setPerPage] = useState(50);
  const [q, setQ] = useState("");
  const [state, setState] = useState("");
  /** **وعددا القسم كلِّه من المحرّك** — لا من طول الصفحة. */
  const [all, setAll] = useState(0);
  const [live, setLive] = useState(0);
  const [error, setError] = useState("");
  // **ومن لا يملك السوقَ لا يُنادي أبوابَه** (فحصُ المتصفّح ٢٠٢٦-١٠-٠٥):
  // المالية تفتح رابطَ قسمٍ فتُردّ ٤٠٣ وتُعرض رسالةُ الخطأ حمراء.
  const canCall = useCanCall();
  const canRead = canCall("GET", "/sections");

  const load = useCallback(async () => {
    if (!canRead) return;
    try {
      // **القسمُ من قائمته** — والقائمةُ قصيرةٌ دائماً (عشرةُ أقسامٍ أو نحوها)،
      // **ونقطةٌ ثانيةٌ لصفٍّ واحدٍ سطحٌ يُصان بلا حاجة.**
      const list = await api<{ sections: Section[] }>("/api/v1/admin/sections");
      setSec((list.sections ?? []).find((x) => x.id === id) ?? null);
      // ══════════════════════════════════════════════════════════════
      // **والبحثُ والترشيحُ يُرسلان إلى المحرّك**
      // ══════════════════════════════════════════════════════════════
      //
      // (كشفه فحصُ المالك ٢٠٢٦-٠٨-١٦.)
      //
      // **كانا يعملان على الصفحة المعروضة وحدَها** — فمن بحث عن صنفٍ في
      // الصفحة الثالثة **قرأ «لا أصناف»**، فيُضيفه مرّتين.
      const qs = new URLSearchParams({ page: String(page) });
      if (q.trim()) qs.set("q", q.trim());
      if (state) qs.set("state", state);
      const res = await api<{
        items: SectionItem[];
        count: number;
        per_page: number;
        all: number;
        live: number;
      }>(`/api/v1/admin/sections/${id}/items?${qs}`);
      setRows(res.items ?? []);
      setCount(res.count ?? 0);
      setPerPage(res.per_page || 50);
      setAll(res.all ?? 0);
      setLive(res.live ?? 0);
      setError("");
    } catch (err) {
      setError(errorText(err));
    }
  }, [id, page, q, state, canRead]);

  useEffect(() => {
    void load();
  }, [load]);

  if (!canRead) return <NotAllowed />;
  if (error) return <p className="py-10 text-center text-danger">{error}</p>;
  if (!rows || !sec) return <LoadingState />;

  // **ولا ترشيحَ هنا** — المحرّكُ رشّح وعدّ، **وترشيحٌ فوق ترشيحٍ يُنقص
  // ما رُشِّح أصلاً.**
  const shown = rows;

  return (
    <PageContainer width="full">
      <button
        onClick={() => router.push("/dashboard/sections")}
        className="mb-3 flex items-center gap-1.5 text-sm text-ink-muted hover:text-ink"
      >
        <IconPrev size={15} />
        {S.backToMarket}
      </button>

      {/* **رأسُ القسم بطاقةٌ واحدة** (قرارُ المالك ٢٠٢٦-١٠-٢٦: «بطريقة احترافية»): الصورة، والاسم،
          والحالة، والأعداد — بدل شريطٍ فارغٍ وشارةٍ طائرةٍ وثلاث مربّعاتٍ كبيرة. */}
      <div className="surface mb-4 flex flex-wrap items-center gap-4 p-4">
        <span className="h-24 w-24 shrink-0 overflow-hidden rounded-card bg-field ring-1 ring-line sm:h-28 sm:w-28">
          {sec.image_url || sec.image_thumb_url ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={mediaUrl(sec.image_url ?? sec.image_thumb_url) ?? ""}
              alt={sec.name}
              className="h-full w-full object-cover"
            />
          ) : (
            <span className="flex h-full items-center justify-center">
              <IconStore size={26} className="text-ink-muted" />
            </span>
          )}
        </span>
        <div className="min-w-0 flex-1 space-y-2">
          <div className="flex flex-wrap items-center gap-2">
            <h1 className="truncate text-2xl font-bold">{sec.name}</h1>
            <StateBadge
              label={sec.active ? S.available : S.unavailable}
              variant={sec.active ? "success" : "danger"}
            />
          </div>
          <div className="flex flex-wrap gap-2 text-sm tabular-nums">
            <Count icon={IconOrder} label={S.statAll} value={all} />
            <Count icon={IconStore} label={S.statLive} value={live} tone="text-success" />
            <Count icon={IconWarning} label={S.statHidden} value={all - live} tone={all - live > 0 ? "text-warning" : ""} />
          </div>
        </div>
      </div>

      <div className="mb-4 flex flex-wrap items-end gap-3">
        <div className="w-64">
          <Input
            id="sec-q"
            icon={<IconSearch />}
            placeholder={S.searchItems}
            value={q}
            onChange={(e) => {
              setQ(e.target.value);
              setPage(1);
            }}
          />
        </div>
        <div className="w-48">
          {/* **والقيمةُ رمزٌ لا نصٌّ معروض** — **وترشيحٌ بنصّ الشارة
              العربيّ ينكسر بتبديل كلمةٍ في المعجم**، ولا يُكتشف حتّى
              يُجرَّب. */}
          <Select
            value={state}
            onChange={(e) => {
              setState(e.target.value);
              setPage(1);
            }}
          >
            <option value="">{S.allStates}</option>
            <option value="live">{S.itemLive}</option>
            <option value="out">{S.itemOut}</option>
            <option value="store_off">{S.itemStoreOff}</option>
          </Select>
        </div>
      </div>

      {/* **بطاقةٌ لكلّ صنفٍ بصورته — كبطاقة القسم.**
          (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «لازم يكون كرت يحوي على صورة لكلّ طعام
          واسمِ المتجر الخاصّ بالصنف والسعر ومعروضٌ أو لا».)

          **والشكلُ واحدٌ في الشاشتين**: من يمسح السوقَ بالعين ثمّ يفتح قسماً
          **لا يُعيد تعلُّمَ أين يقع كلُّ شيء.** */}
      {shown.length === 0 ? (
        <EmptyState icon={IconStore} title={S.noItems} />
      ) : (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5">
          {shown.map((it) => {
            const st = itemState(it);
            return (
              <div
                key={it.id}
                className={`flex flex-col overflow-hidden surface transition-shadow hover:elev-2 ${
                  st.variant === "success" ? "" : "opacity-70"
                }`}
              >
                <div className="relative flex aspect-square items-center justify-center bg-field">
                  {it.image_url || it.thumb_url ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={mediaUrl(it.image_url ?? it.thumb_url) ?? ""}
                      alt={it.name}
                      className="h-full w-full object-cover"
                    />
                  ) : (
                    /* **ونقصُ الصورة يُقال لا يُملأ برمز** — رمزٌ رماديٌّ يجعل
                       الصنفَ يبدو تامّاً وهو ناقص، فيُنسى أنّ صورتَه لم تُرفع. */
                    <span className="flex flex-col items-center gap-1 text-xs text-ink-muted">
                      <IconCamera size={18} />
                      {S.noImage}
                    </span>
                  )}
                  {/* **الحالُ فوق الصورة** — تُقرأ قبل الاسم، وهي أوّلُ ما يُسأل عنه. */}
                  <span className="absolute end-1.5 top-1.5">
                    <StateBadge label={st.label} variant={st.variant} />
                  </span>
                </div>

                <div className="flex flex-1 flex-col gap-2 p-3">
                  <div>
                    <p className="truncate font-bold">{it.name}</p>
                    {/* **واسمُ المتجر يبقى هنا وحدَه** — شاشتُنا لا شاشةُ الزبون.
                        السوقُ يُخفي المصدر عن الزبون، **والإدارةُ لا تُدير ما
                        لا ترى مصدرَه.** */}
                    <p className="truncate text-xs text-ink-muted">{it.merchant_name}</p>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* **والترقيمُ من المكوّن المشترك** — ولا يظهر لصفحةٍ واحدة. */}
      {count > perPage && (
        <div className="mt-4 flex justify-center">
          <Pagination page={page} total={count} perPage={perPage} onChange={setPage} />
        </div>
      )}
    </PageContainer>
  );
}

/** **عدٌّ صغيرٌ في رأس القسم** — أيقونةٌ ورقمٌ ولفظ. */
function Count({
  icon: Icon,
  label,
  value,
  tone = "",
}: {
  icon: typeof IconStore;
  label: string;
  value: number;
  tone?: string;
}) {
  return (
    <span className="inline-flex items-center gap-1.5 rounded-full bg-field px-3 py-1">
      <Icon size={15} className={tone || "text-ink-muted"} />
      <b className={tone}>{fmtNum(value)}</b>
      <span className="text-ink-muted">{label}</span>
    </span>
  );
}
