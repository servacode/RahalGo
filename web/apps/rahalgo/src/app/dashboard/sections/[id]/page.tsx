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

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Button,
  AdminPage,
  AdminAsync,
  AdminEmpty,
  AdminEntityGrid,
  AdminLink,
  FilterBar,
  useLiveData,
  IconPrev,
  IconOrder,
  IconGrid,
  IconCamera,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import StateBadge from "@/components/admin/market/StateBadge";
import { NotAllowed } from "@/components/admin/accounts/ProfileParts";
import { useCanCall } from "@/lib/policy";

const m = getMessages(defaultLocale);
const S = m.admin.sections;
const U = m.admin.market.ui;
const PE = m.admin.kit.people.section;

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
  merchant_id?: string;
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

type StateKey = "all" | "live" | "out" | "store_off";

export default function SectionPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  /** **صفحةُ أصناف القسم** — (قرارُ المالك ٢٠٢٦-٠٨-١٠).

      **والقسمُ يجمع أصنافَ كلّ المتاجر** — ينمو بعدد المتاجر لا بعدد
      الأقسام. **وخمسُمئةٍ صامتةٌ تعني أنّ صنفاً لا يُوافَق عليه لأنّ أحداً
      لم يره.** */
  const [page, setPage] = useState(1);
  const [q, setQ] = useState("");
  const [state, setState] = useState<StateKey>("all");
  const [busyId, setBusyId] = useState("");
  const [actionError, setActionError] = useState("");
  // **ومن لا يملك السوقَ لا يُنادي أبوابَه** (فحصُ المتصفّح ٢٠٢٦-١٠-٠٥):
  // المالية تفتح رابطَ قسمٍ فتُردّ ٤٠٣ وتُعرض رسالةُ الخطأ حمراء.
  const canCall = useCanCall();
  const canRead = canCall("GET", "/sections");
  const canToggle = canCall("PATCH", "/market/items/{itemID}");

  // **القسمُ من قائمته** — والقائمةُ قصيرةٌ دائماً، **ونقطةٌ ثانيةٌ لصفٍّ واحدٍ سطحٌ يُصان بلا حاجة.**
  // **وتُجلب وحدَها** — لا مع كلّ حرفٍ في البحث.
  const secs = useLiveData<{ sections: Section[] }>(
    () => (canRead ? api("/api/v1/admin/sections") : Promise.resolve({ sections: [] })),
    ["catalog"],
    [canRead],
  );
  // ══════════════════════════════════════════════════════════════
  // **والبحثُ والترشيحُ يُرسلان إلى المحرّك**
  // ══════════════════════════════════════════════════════════════
  //
  // (كشفه فحصُ المالك ٢٠٢٦-٠٨-١٦.) **كانا يعملان على الصفحة المعروضة وحدَها** — فمن بحث عن صنفٍ في
  // الصفحة الثالثة **قرأ «لا أصناف»**، فيُضيفه مرّتين.
  const qs = new URLSearchParams({ page: String(page) });
  if (q.trim()) qs.set("q", q.trim());
  if (state !== "all") qs.set("state", state);
  const items = useLiveData<ItemsRes>(
    () => (canRead ? api(`/api/v1/admin/sections/${id}/items?${qs}`) : Promise.reject(new Error("forbidden"))),
    ["menu", "catalog"],
    [id, qs.toString(), canRead],
  );

  if (!canRead) return <NotAllowed />;

  const sec = (secs.data?.sections ?? []).find((x) => x.id === id) ?? null;
  const res = items.data;

  // **قسمٌ انحذف أو رابطٌ قديم يُقال** — كان الدولابُ يلفّ للأبد.
  if (secs.data && !sec) {
    return (
      <AdminPage title={S.backToMarket}>
        <AdminEmpty
          icon={IconGrid}
          title={U.sectionGoneTitle}
          action={<Button onClick={() => router.push("/dashboard/sections")}>{S.backToMarket}</Button>}
        />
      </AdminPage>
    );
  }

  // **ولا ترشيحَ هنا** — المحرّكُ رشّح وعدّ، **وترشيحٌ فوق ترشيحٍ يُنقص ما رُشِّح أصلاً.**
  const shown = res?.items ?? [];
  const perPage = res?.per_page || 50;
  const all = res?.all ?? 0;
  const live = res?.live ?? 0;
  // **و«غير متوفر» و«المتجر موقوف» عدّان** — كان الثالثُ «الكلّ ناقص المعروض» فيُخلط بهما.
  const out = res?.out ?? 0;
  const storeOff = res?.store_off ?? 0;
  const filtered = q.trim() !== "" || state !== "all";
  const pick = (v: StateKey) => {
    setState(v);
    setPage(1);
  };

  /** **تبديلُ التوفّر من البطاقة** — `PATCH /market/items/{id}` نفسُه الذي يستعمله تبويبُ الأصناف. */
  const toggle = async (it: SectionItem) => {
    setBusyId(it.id);
    setActionError("");
    try {
      await api(`/api/v1/admin/market/items/${it.id}`, {
        method: "PATCH",
        body: JSON.stringify({ available: !it.available }),
      });
      items.reload();
    } catch (err) {
      setActionError(errorText(err));
    } finally {
      setBusyId("");
    }
  };

  return (
    <AdminPage
      title={sec?.name ?? S.backToMarket}
      icon={IconGrid}
      badge={
        sec ? (
          <StateBadge label={sec.active ? S.available : S.unavailable} variant={sec.active ? "success" : "warning"} />
        ) : undefined
      }
      actions={
        <Button variant="ghost" size="sm" onClick={() => router.push("/dashboard/sections")}>
          <span className="flex items-center gap-1.5">
            <IconPrev size={15} />
            {S.backToMarket}
          </span>
        </Button>
      }
      tabs={[
        { key: "all", label: S.statAll, count: all },
        { key: "live", label: S.itemLive, count: live },
        { key: "out", label: S.itemOut, count: out, attention: true },
        { key: "store_off", label: S.itemStoreOff, count: storeOff, attention: true },
      ]}
      tab={state}
      onTab={pick}
      notices={actionError ? <Alert tone="warning">{actionError}</Alert> : undefined}
      toolbar={
        <FilterBar
          search={{
            id: "sec-q",
            value: q,
            onChange: (v) => {
              setQ(v);
              setPage(1);
            },
            placeholder: S.searchItems,
          }}
          filtered={filtered}
          onClear={() => {
            setQ("");
            pick("all");
          }}
        />
      }
      pagination={res ? { page, total: res.count, perPage, onChange: setPage } : undefined}
    >
      <AdminAsync
        loading={items.loading || secs.loading}
        error={items.error || secs.error}
        onRetry={() => {
          secs.reload();
          items.reload();
        }}
        hasData={!!res}
        empty={!!res && shown.length === 0}
        emptyTitle={filtered ? U.sectionNoMatch : S.noItems}
        emptyIcon={IconOrder}
        emptyAction={
          filtered ? (
            <Button
              variant="secondary"
              onClick={() => {
                setQ("");
                pick("all");
              }}
            >
              {U.clearFilters}
            </Button>
          ) : (
            <Button onClick={() => router.push("/dashboard/sections?tab=items")}>{U.goAddItem}</Button>
          )
        }
      >
        {/* **بطاقةٌ لكلّ صنفٍ بصورته — كبطاقة القسم.**
            (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «لازم يكون كرت يحوي على صورة لكلّ طعام
            واسمِ المتجر الخاصّ بالصنف والسعر ومعروضٌ أو لا».) */}
        <AdminEntityGrid>
          {shown.map((it) => {
            const st = itemState(it);
            return (
              <article
                key={it.id}
                className="admin-panel flex flex-col overflow-hidden"
                data-tone={st.variant === "success" ? "good" : st.variant === "warning" ? "attention" : "danger"}
              >
                <div className="relative flex aspect-[4/3] items-center justify-center bg-field">
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
                    <span className="flex flex-col items-center gap-1 text-sm text-ink-muted">
                      <IconCamera size={18} />
                      {S.noImage}
                    </span>
                  )}
                  {/* **الحالُ فوق الصورة** — تُقرأ قبل الاسم، وهي أوّلُ ما يُسأل عنه. */}
                  <span className="absolute end-2 top-2">
                    <StateBadge label={st.label} variant={st.variant} />
                  </span>
                </div>

                <div className="flex flex-1 flex-col gap-3 p-3">
                  <div className="min-w-0">
                    <p className="truncate font-bold">{it.name}</p>
                    {/* **واسمُ المتجر رابطٌ لملفّه** — كان نصّاً ميتاً. */}
                    <p className="truncate text-sm text-ink-muted">
                      {it.merchant_id ? (
                        <AdminLink href={`/dashboard/merchants/${it.merchant_id}`} className="hover:underline">
                          {it.merchant_name}
                        </AdminLink>
                      ) : (
                        it.merchant_name
                      )}
                    </p>
                  </div>
                  {canToggle && it.merchant_status === "active" && (
                    <div className="mt-auto">
                      <Button
                        size="sm"
                        variant="secondary"
                        disabled={busyId === it.id}
                        onClick={() => void toggle(it)}
                      >
                        {it.available ? PE.markOut : PE.markAvailable}
                      </Button>
                    </div>
                  )}
                </div>
              </article>
            );
          })}
        </AdminEntityGrid>
      </AdminAsync>
    </AdminPage>
  );
}

interface ItemsRes {
  items: SectionItem[];
  count: number;
  per_page: number;
  all: number;
  live: number;
  out?: number;
  store_off?: number;
}
