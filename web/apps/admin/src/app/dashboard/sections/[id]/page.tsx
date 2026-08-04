"use client";

/**
 * **صفحةُ القسم — أصنافُه كما هي لا كما يراها الزبون.**
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
 */

import { useCallback, useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import {
  Badge,
  Button,
  Input,
  Select,
  PageContainer,
  PageHeader,
  EmptyState,
  LoadingState,
  StatGrid,
  StatCard,
  IconStore,
  IconPrev,
  IconSearch,
  IconOrder,
  IconWarning,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";

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
  merchant_price: number;
  available: boolean;
  approved: boolean;
  merchant_name: string;
  merchant_status: string;
  thumb_url: string | null;
}

/**
 * **حالُ الصنف — وسببُ ظهوره أو غيابه.**
 *
 * **والترتيبُ مقصود**: يُقرأ أوّلُ سببٍ يمنع الظهور. صنفٌ غيرُ مُقَرٍّ ومتجرُه
 * مُطفأٌ **لا يُقال عنه «متجرُه مُطفأ»** — المراجعةُ أوّلُ بابٍ يجب أن يُفتح.
 */
function itemState(it: SectionItem): { label: string; variant: "success" | "warning" | "danger" } {
  if (!it.approved) return { label: S.itemPending, variant: "warning" };
  if (it.merchant_status !== "active") return { label: S.itemStoreOff, variant: "danger" };
  if (!it.available) return { label: S.itemOut, variant: "warning" };
  return { label: S.itemLive, variant: "success" };
}

export default function SectionPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const [sec, setSec] = useState<Section | null>(null);
  const [rows, setRows] = useState<SectionItem[] | null>(null);
  const [q, setQ] = useState("");
  const [state, setState] = useState("");
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      // **القسمُ من قائمته** — والقائمةُ قصيرةٌ دائماً (عشرةُ أقسامٍ أو نحوها)،
      // **ونقطةٌ ثانيةٌ لصفٍّ واحدٍ سطحٌ يُصان بلا حاجة.**
      const list = await api<{ sections: Section[] }>("/api/v1/admin/sections");
      setSec((list.sections ?? []).find((x) => x.id === id) ?? null);
      const res = await api<{ items: SectionItem[] }>(`/api/v1/admin/sections/${id}/items`);
      setRows(res.items ?? []);
      setError("");
    } catch {
      setError(m.errors.internal);
    }
  }, [id]);

  useEffect(() => {
    void load();
  }, [load]);

  if (error) return <p className="py-10 text-center text-danger">{error}</p>;
  if (!rows || !sec) return <LoadingState />;

  const term = q.trim();
  const shown = rows.filter(
    (it) =>
      (term === "" || it.name.includes(term) || it.merchant_name.includes(term)) &&
      (state === "" || itemState(it).label === state),
  );
  const live = rows.filter((it) => itemState(it).variant === "success").length;

  return (
    <PageContainer width="full">
      <button
        onClick={() => router.push("/dashboard/sections")}
        className="mb-3 flex items-center gap-1.5 text-sm text-ink-muted hover:text-ink"
      >
        <IconPrev size={15} />
        {S.backToMarket}
      </button>

      <div className="mb-4 flex flex-wrap items-start gap-3">
        {/* **صورةُ القسم في ترويسته** — هي ما يعرفه بها من يفتحها. */}
        <span className="h-20 w-28 shrink-0 overflow-hidden rounded-card bg-page">
          {sec.image_url || sec.image_thumb_url ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={mediaUrl(sec.image_url ?? sec.image_thumb_url) ?? ""}
              alt={sec.name}
              className="h-full w-full object-cover"
            />
          ) : (
            <span className="flex h-full items-center justify-center">
              <IconStore size={22} className="text-ink-muted" />
            </span>
          )}
        </span>
        <div className="min-w-0 flex-1">
          <PageHeader icon={IconStore} title={sec.name} />
        </div>
        <Badge variant={sec.active ? "success" : "neutral"}>
          {sec.active ? S.available : S.unavailable}
        </Badge>
      </div>

      <StatGrid>
        <StatCard label={S.statAll} value={fmtNum(rows.length)} icon={IconOrder} />
        {/* **والمعروضُ فعلاً لا المسجَّل** — قسمٌ فيه اثنا عشر ويُعرض منه ثلاثةٌ
            حالةٌ تُعالَج، **ورقمٌ واحدٌ يخفيها.** */}
        <StatCard label={S.statLive} value={fmtNum(live)} icon={IconStore} />
        <StatCard label={S.statHidden} value={fmtNum(rows.length - live)} icon={IconWarning} />
      </StatGrid>

      <div className="mb-4 mt-4 flex flex-wrap items-end gap-3">
        <div className="w-64">
          <Input
            id="sec-q"
            icon={<IconSearch />}
            placeholder={S.searchItems}
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
        </div>
        <div className="w-48">
          <Select value={state} onChange={(e) => setState(e.target.value)}>
            <option value="">{S.allStates}</option>
            {[S.itemLive, S.itemPending, S.itemOut, S.itemStoreOff].map((x) => (
              <option key={x} value={x}>
                {x}
              </option>
            ))}
          </Select>
        </div>
      </div>

      {shown.length === 0 ? (
        <EmptyState icon={IconStore} title={S.noItems} />
      ) : (
        <ul className="divide-y divide-line rounded-card border border-line bg-surface">
          {shown.map((it) => {
            const st = itemState(it);
            return (
              <li key={it.id} className="flex items-center gap-3 p-3">
                <span className="h-11 w-11 shrink-0 overflow-hidden rounded-control bg-page">
                  {it.thumb_url && (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={mediaUrl(it.thumb_url) ?? ""}
                      alt={it.name}
                      className="h-full w-full object-cover"
                    />
                  )}
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate font-medium">{it.name}</span>
                  <span className="block text-xs text-ink-muted">{it.merchant_name}</span>
                </span>
                <span dir="ltr" className="shrink-0 tabular-nums">
                  {fmtNum(it.merchant_price)}
                </span>
                <Badge variant={st.variant}>{st.label}</Badge>
              </li>
            );
          })}
        </ul>
      )}
    </PageContainer>
  );
}
