"use client";

/**
 * **ملفُّ المتجر — كيانٌ لا شخص.**
 *
 * # لماذا لزم
 *
 * بُني الملفُّ الشخصيُّ للأشخاص (`users/[id]`) فجمع كلَّ ما يخصّهم. **والمتجرُ
 * لا ملفَّ له**: قائمتُه وساعاتُه ومخالفاتُه وعمولتُه ومبيعاتُه **كلُّها في
 * نوافذَ منبثقةٍ داخل جدول** — تُفتح واحدةً وتُغلق لتُفتح أخرى، **ولا تُرى
 * صورتُه مجتمعةً.**
 *
 * **وهو أعقدُ من إنسان**: للإنسان هاتفٌ ورصيد، وللمتجر ثلاثون صنفاً وسبعةُ
 * أيّامِ دوامٍ وعدّادُ مخالفاتٍ يقود إلى حظر.
 *
 * # والمتجرُ ليس صاحبَه
 *
 * `owner_user_id` حسابٌ آخر له ملفُّه، **وقد يتغيّر ويبقى المتجرُ**. فالرابطُ
 * بينهما ظاهرٌ ولا يُدمجان: **مخالفاتُ المتجر لا تُحسب على من اشتراه أمس.**
 */

import { useCallback, useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, fmtDate } from "@rahalgo/i18n";
import {
  Tabs,
  type TabDef,
  Badge,
  Button,
  StoreHours,
  MenuManager,
  type MenuPaths,
  LoadingState,
  StatGrid,
  StatCard,
  IconStore,
  IconPrev,
  IconUser,
  IconWarning,
  IconOrder,
  IconBalance,
  IconDate,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import ViolationsModal from "@/components/ViolationsModal";

const m = getMessages(defaultLocale);
const P = m.admin.merchantProfile;

const MERCHANT_STATUS: Record<string, string> = {
  active: m.admin.merchants.active,
  inactive: m.admin.merchants.inactive,
  suspended: m.admin.merchants.suspended,
};

interface Merchant {
  id: string;
  name: string;
  phone: string;
  address_text: string;
  category_name: string;
  status: string;
  owner_user_id: string | null;
  owner_phone: string | null;
  sales_rep_phone: string | null;
  violations: number;
  commission_percent: number;
  emergency_closed: boolean;
  /** أيستردّ بضاعةَ طلبٍ تعذّر تسليمُه — وعليه يظهر زرُّ الردّ في الطلبات. */
  accepts_returns: boolean;
  created_at: string;
}

/** مساراتُ القائمة من باب الإدارة — الحارسُ في الخادم يختلف والشكلُ واحد. */
const PATHS: MenuPaths = {
  menu: (id) => `/api/v1/admin/merchants/${id}/menu`,
  sections: (id) => `/api/v1/admin/merchants/${id}/menu/sections`,
  section: (id) => `/api/v1/admin/menu/sections/${id}`,
  items: (id) => `/api/v1/admin/merchants/${id}/menu/items`,
  item: (id) => `/api/v1/admin/menu/items/${id}`,
  platformSections: () => `/api/v1/admin/sections`,
};

type Tab = "overview" | "menu" | "hours" | "orders";

export default function MerchantProfilePage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const [mr, setMr] = useState<Merchant | null>(null);
  const [tab, setTab] = useState<Tab>("overview");
  const [violationsOpen, setViolationsOpen] = useState(false);
  const [error, setError] = useState("");
  const [savingReturns, setSavingReturns] = useState(false);

  const load = useCallback(async () => {
    try {
      setMr(await api<Merchant>(`/api/v1/admin/merchants/${id}`));
      setError("");
    } catch {
      setError(m.errors.internal);
    }
  }, [id]);

  /** بندُ الاسترداد — يُحفظ فوراً ثمّ يُعاد التحميلُ ليُقرأ من القاعدة. */
  const setAcceptsReturns = useCallback(
    async (v: boolean) => {
      setSavingReturns(true);
      try {
        await api(`/api/v1/admin/merchants/${id}`, {
          method: "PATCH",
          body: JSON.stringify({ accepts_returns: v }),
        });
        await load();
      } catch {
        setError(m.errors.internal);
      } finally {
        setSavingReturns(false);
      }
    },
    [id, load],
  );

  useEffect(() => {
    void load();
  }, [load]);

  if (error) return <p className="py-10 text-center text-danger">{error}</p>;
  if (!mr) return <LoadingState />;

  const TABS: TabDef<Tab>[] = [
    { key: "overview", label: P.tabs.overview, icon: IconStore },
    { key: "menu", label: P.tabs.menu, icon: IconOrder },
    { key: "hours", label: P.tabs.hours, icon: IconDate },
    { key: "orders", label: P.tabs.orders, icon: IconOrder },
  ];

  return (
    <div>
      <button
        onClick={() => router.push("/dashboard/users")}
        className="mb-3 flex items-center gap-1.5 text-sm text-ink-muted hover:text-ink"
      >
        <IconPrev size={15} />
        {P.back}
      </button>

      <div className="mb-4 flex flex-wrap items-start gap-3">
        <span className="flex h-14 w-14 shrink-0 items-center justify-center rounded-card bg-primary-light">
          <IconStore size={26} className="text-primary-dark" />
        </span>
        <div className="min-w-0 flex-1">
          <h1 className="text-2xl font-bold">{mr.name}</h1>
          <p className="text-sm text-ink-muted">
            {mr.category_name} · <span dir="ltr">{mr.phone}</span>
          </p>
          <p className="text-xs text-ink-muted">{mr.address_text}</p>
        </div>
        <div className="flex shrink-0 flex-wrap items-center gap-2">
          <Badge variant={mr.status === "active" ? "success" : "danger"}>
            {MERCHANT_STATUS[mr.status] ?? mr.status}
          </Badge>
          {/* **والإغلاقُ الطارئ يُعلَن** — يتجاوز الجدولَ ويُظهره مغلقاً فوراً. */}
          {mr.emergency_closed && <Badge variant="danger">{m.admin.hours.emergencyActive}</Badge>}
        </div>
      </div>

      <StatGrid>
        <StatCard
          label={m.admin.merchants.commission}
          value={`${fmtNum(mr.commission_percent)}%`}
          icon={IconBalance}
        />
        <StatCard label={P.violations} value={fmtNum(mr.violations)} icon={IconWarning} />
        <StatCard label={P.joined} value={fmtDate(mr.created_at)} icon={IconDate} />
      </StatGrid>

      <div className="mt-4 flex flex-wrap gap-2">
        <Button variant="secondary" onClick={() => setViolationsOpen(true)}>
          <span className="flex items-center gap-1.5">
            <IconWarning size={15} />
            {m.admin.merchants.violationsLog.viewLog}
          </span>
        </Button>
        {/* **وصاحبُه حسابٌ آخر له ملفُّه** — والرابطُ ظاهرٌ ولا يُدمجان. */}
        {mr.owner_user_id && (
          <Button
            variant="secondary"
            onClick={() => router.push(`/dashboard/users/${mr.owner_user_id}`)}
          >
            <span className="flex items-center gap-1.5">
              <IconUser size={15} />
              {P.owner}
              {mr.owner_phone && <span dir="ltr">· {mr.owner_phone}</span>}
            </span>
          </Button>
        )}
      </div>

      <Tabs className="mb-4 mt-5" items={TABS} value={tab} onChange={setTab} />

      {tab === "overview" && (
        <dl className="grid gap-3 surface p-4 sm:grid-cols-2">
          <Row label={m.terms.phone} value={mr.phone} ltr />
          <Row label={P.address} value={mr.address_text} />
          <Row label={m.terms.category} value={mr.category_name} />
          <Row label={P.rep} value={mr.sales_rep_phone ?? "—"} ltr />
          {/* **بندٌ في الاتّفاق لا رأيٌ يُبديه ساعتَها.**

              عليه يظهر زرُّ «رُدّت إلى المتجر» حين يتعذّر تسليمُ طلبٍ من
              عنده — **ومن يملك تغييرَه وحدَه يغلقه ساعةَ تُردّ إليه بضاعة.**
              فموضعُه هنا لا في شاشته. (قرارُ المالك ٢٠٢٦-٠٨-٠٤.) */}
          <div className="flex items-center justify-between gap-3 sm:col-span-2">
            <dt className="shrink-0 text-sm text-ink-muted">{P.acceptsReturns}</dt>
            <dd className="flex gap-1">
              {[true, false].map((v) => (
                <button
                  key={String(v)}
                  type="button"
                  disabled={savingReturns}
                  onClick={() => void setAcceptsReturns(v)}
                  className={`rounded-control border px-3 py-1 text-sm transition-colors ${
                    mr.accepts_returns === v
                      ? "border-accent bg-accent-tint font-medium"
                      : "border-line text-ink-muted hover:border-accent-edge"
                  }`}
                >
                  {v ? P.returnsYes : P.returnsNo}
                </button>
              ))}
            </dd>
          </div>
        </dl>
      )}
      {tab === "menu" && <MenuManager api={api} paths={PATHS} merchantID={mr.id} />}
      {tab === "hours" && (
        <div className="surface p-4">
          <StoreHours
            api={api}
            path={`/api/v1/admin/merchants/${mr.id}/hours`}
            emergency={{
              value: mr.emergency_closed,
              save: async (v) => {
                await api(`/api/v1/admin/merchants/${mr.id}`, {
                  method: "PATCH",
                  body: JSON.stringify({ emergency_closed: v }),
                });
                await load();
              },
            }}
          />
        </div>
      )}
      {tab === "orders" && (
        /* **وطلباتُه في شاشتها** — الجدولُ هناك يحمل كلَّ أزراره، **ونسخُه
           هنا يجعل زرّاً يُصلَح في موضعٍ ويبقى معطوباً في الآخر.** */
        <div className="surface p-6 text-center">
          <p className="mb-3 text-sm text-ink-muted">{P.ordersHint}</p>
          <Button onClick={() => router.push(`/dashboard/history?q=${encodeURIComponent(mr.name)}`)}>
            {P.openOrders}
          </Button>
        </div>
      )}

      {violationsOpen && (
        <ViolationsModal
          merchant={mr}
          onClose={() => setViolationsOpen(false)}
          onChanged={load}
        />
      )}
    </div>
  );
}

function Row({ label, value, ltr }: { label: string; value: string; ltr?: boolean }) {
  return (
    <div className="flex items-baseline justify-between gap-3">
      <dt className="shrink-0 text-sm text-ink-muted">{label}</dt>
      <dd className="min-w-0 truncate font-medium" dir={ltr ? "ltr" : undefined}>
        {value || "—"}
      </dd>
    </div>
  );
}
