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

import { Suspense, useCallback, useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth";
import { getMessages, defaultLocale, fmtNum, fmtMoney, fmtDateTime, fmtRelStamp, errorText, fmtStoreNo, fmtRef } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Chips,
  ConfirmedMenu,
  useLiveRefresh,
  StoreHours,
  MenuManager,
  type MenuPaths,
  AdminPage,
  AdminAsync,
  AdminCard,
  AdminLink,
  ContactButtons,
  RowList,
  Row,
  StatusBadge,
  SummaryStrip,
  useLiveData,
  IconStore,
  IconPrev,
  IconUser,
  IconWarning,
  IconOrder,
  IconDate,
  IconCamera,
  IconEdit,
  ButtonLink,
} from "@rahalgo/ui";
import { useUrlTab } from "@/components/admin/useUrlTab";
import DrinksAdmin from "@/components/admin/market/DrinksAdmin";
import ImageUpload, { MediaThumb, useImagePicker } from "@/components/admin/ImageUpload";
import { api, mediaUrl } from "@/lib/api";
import ViolationsModal from "@/components/admin/ViolationsModal";
import { MerchantModal, type Merchant as EditableMerchant } from "@/components/admin/MerchantModal";
import { MerchantSettlement } from "@/components/admin/MerchantSettlement";
import PriceSyncAdmin from "@/components/admin/PriceSyncAdmin";
import { DeleteMerchantButton, OrderPhonesAdmin, StoreStaffAdmin } from "@/components/admin/StoreStaffAdmin";
import { commissionText } from "@/components/admin/ProfileRoleTabs";
import { NotAllowed } from "@/components/admin/accounts/ProfileParts";
import { useCanCall } from "@/lib/policy";
import { MerchantAnnounce } from "@/components/admin/MerchantAnnounce";

const m = getMessages(defaultLocale);
const P = m.admin.merchantProfile;
const PS = m.admin.kit.people.store;

interface Merchant {
  id: string;
  /** **رقمُه القصير** («#M-1045») — يفرّق بين متجرين باسمٍ واحد. */
  number: number;
  name: string;
  phone: string;
  address_text: string;
  category_name: string;
  status: string;
  owner_user_id: string | null;
  owner_phone: string | null;
  sales_rep_phone: string | null;
  violations: number;
  /** **نسبتُه الخاصّة** — وفراغُها: يتبع العامّة (قرارُ المالك ٢٠٢٦-١٠-٠٤). */
  commission_percent: number | null;
  general_commission_percent: number;
  /** **بمنزلتين** (١٫٥) — والصحيحتان أعلاه مقرَّبتان للعرض القديم (٢٠٢٦-١٠-١٨). */
  commission_percent_exact?: number | null;
  general_commission_percent_exact?: number;
  emergency_closed: boolean;
  /** أيستردّ بضاعةَ طلبٍ تعذّر تسليمُه — وعليه يظهر زرُّ الردّ في الطلبات. */
  accepts_returns: boolean;
  /** طريقةُ تسويةِ مستحقّاته — نقدٌ أو محفظة (الأدمنُ ذو الصلاحيّة الماليّة يغيّرها). */
  settlement_method: "cash" | "wallet";
  /** **الشعار** — يُرى في رأس الملفّ، وفراغُه أيقونةُ متجر. */
  logo_url?: string | null;
  logo_thumb_url?: string | null;
  /** **«متجر مميّز»** في شريط السوق وترتيبُه (٢٠٢٦-١٠-١٨). */
  featured?: boolean;
  featured_order?: number;
  created_at: string;
}

/** مساراتُ القائمة من باب الإدارة — الحارسُ في الخادم يختلف والشكلُ واحد. */
const PATHS: MenuPaths = {
  menu: (id) => `/api/v1/admin/merchants/${id}/menu`,
  items: (id) => `/api/v1/admin/merchants/${id}/menu/items`,
  item: (id) => `/api/v1/admin/menu/items/${id}`,
  platformSections: () => `/api/v1/admin/sections`,
  merchant: (id) => `/api/v1/admin/merchants/${id}`,
};

/** نبضُ المتجر الآن — `GET /merchants/{id}/pulse` (قراءةٌ فقط). */
interface Pulse {
  open_now: boolean;
  next_open: string | null;
  default_prep_minutes: number;
  orders_today: number;
  delivered_today: number;
  orders_open: number;
  sent_30d: number;
  rejected_30d: number;
  items_total: number;
  items_unavailable: number;
}

type Tab = "overview" | "menu" | "hours" | "orders";
const TAB_KEYS: readonly Tab[] = ["overview", "menu", "hours", "orders"];

/** **حالةُ المتجر بكلمةٍ واحدة** — الإيقافُ حظرٌ، والتعطيلُ إجازة، وغيرُهما يقرّره الدوام. */
function storeState(mr: Merchant, pulse: Pulse | null): string {
  if (mr.status === "suspended") return "banned";
  if (mr.status === "inactive") return "paused";
  if (mr.emergency_closed) return "closed";
  return pulse && !pulse.open_now ? "closed" : "open";
}

export default function MerchantProfilePage() {
  return (
    <Suspense fallback={null}>
      <MerchantProfile />
    </Suspense>
  );
}

function MerchantProfile() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const [mr, setMr] = useState<Merchant | null>(null);
  const [tab, setTab] = useUrlTab<Tab>(TAB_KEYS, "overview");
  const [violationsOpen, setViolationsOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [error, setError] = useState("");
  const [loadFailed, setLoadFailed] = useState(false);
  const [savingReturns, setSavingReturns] = useState(false);
  // **ومن لا يملك قراءةَ المتجر لا يُنادي بابَه** (فحصُ المتصفّح ٢٠٢٦-١٠-٠٥):
  // المالية تصل الصفحةَ من ملفّ صاحب المتجر، فكانت تُردّ ٤٠٣ وتُعرض
  // رسالةَ الخطأ حمراءَ بدل «غير مسموح».
  const canCall = useCanCall();
  const { can } = useAuth();
  const canMarket = can("market.manage");
  const canRead = canCall("GET", "/merchants/{id}");
  const canPulse = canCall("GET", "/merchants/{id}/pulse");
  const canEditMenu = canCall("POST", "/merchants/{id}/menu/items");
  const canEdit = canCall("PATCH", "/merchants/{id}");
  const canEditHours = canCall("PUT", "/merchants/{id}/hours");
  const canViolations = canCall("GET", "/merchants/{id}/violations");
  const canCash = can("finance.read");

  // **الضغطُ على اللوغو يفتح ملفّات الجهاز فوراً** (قرارُ المالك ٢٠٢٦-١٠-٢٦).
  const logoPicker = useImagePicker({
    kind: "merchant_logo",
    onUploaded: async (mediaID) => {
      await api(`/api/v1/admin/merchants/${id}`, { method: "PATCH", body: JSON.stringify({ logo_media_id: mediaID }) });
      await load();
    },
    onError: (msg) => setError(msg),
  });

  const load = useCallback(async () => {
    if (!canRead) return;
    try {
      setMr(await api<Merchant>(`/api/v1/admin/merchants/${id}`));
      setLoadFailed(false);
    } catch (err) {
      setError(errorText(err));
      setLoadFailed(true);
    }
  }, [id, canRead]);

  /** النبضُ والمستحقُّ النقديّ — يتجدّدان مع كلّ حدثِ متجرٍ أو طلبٍ أو قائمة. */
  const live = useLiveData(
    async () => {
      const [pulse, cash] = await Promise.all([
        canPulse ? api<Pulse>(`/api/v1/admin/merchants/${id}/pulse`) : Promise.resolve(null),
        canCash
          ? api<{ outstanding_total?: number }>(`/api/v1/admin/merchants/${id}/cash-settlements`).catch(() => null)
          : Promise.resolve(null),
      ]);
      return { pulse, cashDue: cash ? (cash.outstanding_total ?? 0) : null };
    },
    ["order", "merchant", "catalog", "menu"],
    [id, canPulse, canCash],
    800,
  );
  const pulse = live.data?.pulse ?? null;

  useLiveRefresh(["merchant", "catalog"], load);

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
      } catch (err) {
        setError(errorText(err));
      } finally {
        setSavingReturns(false);
      }
    },
    [id, load],
  );

  useEffect(() => {
    void load();
  }, [load]);

  if (!canRead) return <NotAllowed />;
  if (!mr) {
    return (
      <AdminPage title={PS.storeLabel}>
        <AdminAsync loading={!loadFailed} error={loadFailed} onRetry={() => void load()} hasData={false}>
          {null}
        </AdminAsync>
      </AdminPage>
    );
  }

  const state = storeState(mr, pulse);
  const off = pulse?.items_unavailable ?? 0;
  const historyHref = `/dashboard/history?merchant=${encodeURIComponent(mr.id)}`;
  const ordersHref = `/dashboard/orders?merchant=${encodeURIComponent(mr.id)}`;

  return (
    <AdminPage
      title={mr.name}
      icon={IconStore}
      badge={
        <>
          <StatusBadge domain="store" status={state} />
          {mr.emergency_closed && <Badge variant="danger">{m.admin.hours.emergencyActive}</Badge>}
          {mr.featured && (
            <Badge variant="accent">{`${m.admin.merchants.featuredBadge} · ${fmtNum(mr.featured_order ?? 0)}`}</Badge>
          )}
        </>
      }
      meta={[fmtStoreNo(mr.number), mr.category_name, mr.address_text].filter(Boolean).join(" · ")}
      actions={
        <>
          <Button variant="ghost" size="sm" onClick={() => router.back()}>
            <span className="flex items-center gap-1.5">
              <IconPrev size={15} />
              {m.admin.kit.back}
            </span>
          </Button>
          {can("audit.read") && (
            <ButtonLink variant="secondary" size="sm" href={`/dashboard/audit?entity=merchant&entity_id=${mr.id}`}>
              {m.admin.kit.finance.viewHistory}
            </ButtonLink>
          )}
          {/* **وتعديلُ المتجر من صفحته** (بلاغُ المالك ٢٠٢٦-١٠-٠٧: «ما في خيار تعديل»). */}
          {canEdit && (
            <Button variant="secondary" size="sm" onClick={() => setEditOpen(true)}>
              <span className="flex items-center gap-1.5">
                <IconEdit size={15} />
                {m.admin.merchants.edit}
              </span>
            </Button>
          )}
          {canViolations && (
            <ConfirmedMenu
              items={[
                {
                  key: "violations",
                  label: m.admin.merchants.violationsLog.viewLog,
                  icon: IconWarning,
                  onSelect: () => setViolationsOpen(true),
                },
              ]}
            />
          )}
        </>
      }
      tabs={[
        { key: "overview", label: P.tabs.overview },
        { key: "menu", label: P.tabs.menu, count: off > 0 ? off : null, attention: true },
        { key: "hours", label: P.tabs.hours },
        { key: "orders", label: P.tabs.orders },
      ]}
      tab={tab}
      onTab={setTab}
      notices={error ? <Alert tone="warning">{error}</Alert> : undefined}
      summary={
        <SummaryStrip
          max={6}
          items={[
            {
              key: "open",
              label: state === "open" ? PS.openNow : PS.closedNow,
              value: <StatusBadge domain="store" status={state} />,
              tone: state === "open" ? "good" : state === "banned" ? "danger" : "muted",
              meta:
                state === "closed" && pulse?.next_open
                  ? PS.opensAt.replace("{when}", fmtRelStamp(pulse.next_open))
                  : undefined,
              onOpen: () => setTab("hours"),
            },
            {
              key: "today",
              label: PS.ordersToday,
              value: fmtNum(pulse?.orders_today ?? 0),
              meta: PS.delivered.replace("{n}", fmtNum(pulse?.delivered_today ?? 0)),
              href: historyHref,
            },
            {
              key: "live",
              label: PS.ordersLive,
              value: fmtNum(pulse?.orders_open ?? 0),
              tone: (pulse?.orders_open ?? 0) > 0 ? "attention" : "neutral",
              href: ordersHref,
            },
            {
              key: "off",
              label: PS.itemsOff,
              value: fmtNum(off),
              tone: off > 0 ? "attention" : "neutral",
              meta: PS.itemsOffOf.replace("{n}", fmtNum(pulse?.items_total ?? 0)),
              onOpen: () => setTab("menu"),
            },
            {
              key: "violations",
              label: P.violations,
              value: fmtNum(mr.violations),
              tone: mr.violations > 0 ? "attention" : "neutral",
              onOpen: canViolations ? () => setViolationsOpen(true) : undefined,
            },
            live.data?.cashDue != null
              ? {
                  key: "cash",
                  label: PS.cashDue,
                  value: fmtMoney(live.data.cashDue),
                  tone: live.data.cashDue > 0 ? "attention" : "neutral",
                  onOpen: () => setTab("overview"),
                }
              : {
                  key: "rejected",
                  label: PS.rejected,
                  value: PS.rejectedOf
                    .replace("{r}", fmtNum(pulse?.rejected_30d ?? 0))
                    .replace("{s}", fmtNum(pulse?.sent_30d ?? 0)),
                  tone: (pulse?.rejected_30d ?? 0) > 0 ? "attention" : "neutral",
                },
          ]}
        />
      }
    >
      {tab === "overview" && (
        <>
          <section className="space-y-2">
            <h2 className="admin-h2">{PS.contacts}</h2>
            <RowList>
              <Row
                leading={
                  <button
                    type="button"
                    title={m.admin.market.stores.changeLogo}
                    aria-label={m.admin.market.stores.changeLogo}
                    disabled={!canEdit || logoPicker.busy}
                    onClick={logoPicker.open}
                    className="group relative block h-11 w-11 overflow-hidden rounded-card bg-field ring-1 ring-line enabled:hover:ring-primary"
                  >
                    {mr.logo_thumb_url || mr.logo_url ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img src={mediaUrl(mr.logo_thumb_url ?? mr.logo_url) ?? ""} alt={mr.name} className="h-full w-full object-cover" />
                    ) : (
                      <span className="flex h-full items-center justify-center">
                        <IconStore size={22} className="text-primary-dark" />
                      </span>
                    )}
                    {canEdit && (
                      <span className="absolute bottom-0.5 start-0.5 inline-flex h-4 w-4 items-center justify-center rounded-full bg-surface text-ink elev-1">
                        <IconCamera size={10} />
                      </span>
                    )}
                  </button>
                }
                title={PS.storeLabel}
                meta={<span dir="ltr">{mr.phone || "—"}</span>}
                trailing={<ContactButtons phone={mr.phone} name={mr.name} />}
              />
              {/* **وصاحبُه حسابٌ آخر له ملفُّه** — والرابطُ ظاهرٌ ولا يُدمجان. */}
              {mr.owner_user_id && (
                <Row
                  leading={<IconUser size={18} />}
                  title={
                    <AdminLink href={`/dashboard/users/${mr.owner_user_id}`} className="hover:underline">
                      {PS.ownerLabel}
                    </AdminLink>
                  }
                  meta={<span dir="ltr">{mr.owner_phone ?? "—"}</span>}
                  trailing={<ContactButtons phone={mr.owner_phone} name={PS.ownerLabel} />}
                />
              )}
              {mr.sales_rep_phone && (
                <Row
                  leading={<IconUser size={18} />}
                  title={PS.repLabel}
                  meta={<span dir="ltr">{mr.sales_rep_phone}</span>}
                  trailing={<ContactButtons phone={mr.sales_rep_phone} name={PS.repLabel} />}
                />
              )}
            </RowList>
          </section>

          <AdminCard>
            <dl className="grid grid-cols-1 gap-x-6 gap-y-3 sm:grid-cols-2">
              <Info label={P.address} value={mr.address_text} />
              <Info
                label={m.admin.merchants.commission}
                value={commissionText(
                  mr.commission_percent_exact !== undefined ? mr.commission_percent_exact : mr.commission_percent,
                  mr.general_commission_percent_exact ?? mr.general_commission_percent,
                )}
              />
              <Info
                label={PS.prep}
                value={pulse ? PS.prepValue.replace("{n}", fmtNum(pulse.default_prep_minutes)) : "—"}
              />
              <Info
                label={PS.rejected}
                value={
                  pulse
                    ? PS.rejectedOf.replace("{r}", fmtNum(pulse.rejected_30d)).replace("{s}", fmtNum(pulse.sent_30d))
                    : "—"
                }
              />
              {/* **بندٌ في الاتّفاق لا رأيٌ يُبديه ساعتَها.**

                  عليه يظهر زرُّ «رُدّت إلى المتجر» حين يتعذّر تسليمُ طلبٍ من
                  عنده — **ومن يملك تغييرَه وحدَه يغلقه ساعةَ تُردّ إليه بضاعة.**
                  فموضعُه هنا لا في شاشته. (قرارُ المالك ٢٠٢٦-٠٨-٠٤.) */}
              <div className="flex items-center justify-between gap-3 sm:col-span-2">
                <dt className="shrink-0 text-sm text-ink-muted">{P.acceptsReturns}</dt>
                <dd>
                  <Chips
                    value={mr.accepts_returns ? "yes" : "no"}
                    onChange={(v) => {
                      if (savingReturns || !canEdit) return;
                      void setAcceptsReturns(v === "yes");
                    }}
                    items={[
                      { id: "yes", label: P.returnsYes },
                      { id: "no", label: P.returnsNo },
                    ]}
                  />
                </dd>
              </div>
            </dl>
          </AdminCard>

          <MerchantSettlement
            merchantId={mr.id}
            method={mr.settlement_method}
            onChanged={() => {
              void load();
              live.reload();
            }}
          />
          {/* **«انضمّ متجر جديد» — و«لا ترسل»** (قرارُ المالك ٢٠٢٦-١٠-٢١). */}
          <MerchantAnnounce merchantId={mr.id} canManage={canCall("PATCH", "/merchants/{id}/announce")} />
          {/* **مزامنةُ الأمين — فنيّ** (٢٠٢٦-١٠-١٩). */}
          {/* **موظّفو المتجر ورقمُ الطلبات بحسب الدوام** (قرارُ المالك ٢٠٢٦-١٠-٢٠). */}
          <StoreStaffAdmin merchantId={mr.id} canManage={canCall("POST", "/merchants/{id}/staff")} />
          <OrderPhonesAdmin merchantId={mr.id} canManage={canCall("PUT", "/merchants/{id}/order-phones")} />
          {canEditMenu && <PriceSyncAdmin merchantId={mr.id} />}
          {canCall("DELETE", "/merchants/{id}") && (
            <DeleteMerchantButton
              merchantId={mr.id}
              name={mr.name}
              onDeleted={() =>
                /* **بعد الحذف إلى قائمة المتاجر الموجودة** — تبويبُ «المتاجر» في السوق لمن يملكه،
                   وإلّا الحساباتُ مفلترةً بأصحاب المتاجر. */
                router.push(canMarket ? "/dashboard/sections?tab=stores" : "/dashboard/users?role=merchant")
              }
            />
          )}
        </>
      )}
      {tab === "menu" && <DrinksAdmin merchantId={mr.id} canEdit={canEditMenu} />}
      {tab === "menu" && (
        <MenuManager
          api={api}
          paths={PATHS}
          merchantID={mr.id}
          mediaUrl={mediaUrl}
          showSalePrice
          readOnly={!canEditMenu}
          // **وصورةُ الصنف تُبدَّل من هنا** (قرارُ المالك ٢٠٢٦-١٠-٢٥: «الأدمن يقدر يغيّر الصورة»).
          imageUpload={(initialUrl, onChange) => (
            <ImageUpload kind="menu_item" label={m.shared.menuEditor.itemImage} initialUrl={initialUrl} onChange={onChange} />
          )}
          thumb={(url, alt) => <MediaThumb url={url} alt={alt} fallback={alt} size={48} />}
        />
      )}
      {tab === "hours" && (
        <AdminCard>
          <StoreHours
            api={api}
            path={`/api/v1/admin/merchants/${mr.id}/hours`}
            readOnly={!canEditHours}
            emergency={
              canEdit
                ? {
                    value: mr.emergency_closed,
                    save: async (v) => {
                      await api(`/api/v1/admin/merchants/${mr.id}`, {
                        method: "PATCH",
                        body: JSON.stringify({ emergency_closed: v }),
                      });
                      await load();
                      live.reload();
                    },
                  }
                : undefined
            }
          />
        </AdminCard>
      )}
      {tab === "orders" && <StoreOrders merchantId={mr.id} historyHref={historyHref} />}

      {editOpen && (
        <MerchantModal
          merchant={mr as unknown as EditableMerchant}
          onClose={() => setEditOpen(false)}
          onSaved={() => {
            setEditOpen(false);
            void load();
          }}
        />
      )}
      {violationsOpen && (
        <ViolationsModal
          merchant={mr}
          onClose={() => setViolationsOpen(false)}
          onChanged={load}
        />
      )}
    </AdminPage>
  );
}

interface StoreOrder {
  id: string;
  number: number;
  status: string;
  total: number;
  created_at: string;
}

/** **آخرُ طلباته في تبويبه** — وكلُّها في السجلّ بجدوله وأزراره كاملة (نسخُه هنا يعطّل زرّاً في موضعٍ ويصلحه في آخر). */
function StoreOrders({ merchantId, historyHref }: { merchantId: string; historyHref: string }) {
  const q = useLiveData(
    async () => {
      const r = await api<{ orders?: StoreOrder[] }>(
        `/api/v1/admin/orders?merchant_id=${encodeURIComponent(merchantId)}&per_page=10`,
      );
      return r.orders ?? [];
    },
    ["order"],
    [merchantId],
    800,
  );
  const rows = q.data ?? [];
  return (
    <AdminAsync
      loading={q.loading}
      error={q.error}
      onRetry={q.reload}
      hasData={q.data != null}
      empty={q.data != null && rows.length === 0}
      emptyTitle={PS.noOrders}
      emptyIcon={IconOrder}
    >
      <section className="space-y-2">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h2 className="admin-h2">{PS.latest}</h2>
          <AdminLink href={historyHref} className="text-sm font-bold text-primary-dark hover:underline">
            {PS.allOrders}
          </AdminLink>
        </div>
        <RowList>
          {rows.map((o) => (
            <Row
              key={o.id}
              leading={<IconDate size={16} />}
              title={
                <AdminLink href={`/dashboard/history?order=${o.id}`} className="hover:underline">
                  <span dir="ltr">#{fmtRef(o.number)}</span>
                </AdminLink>
              }
              meta={fmtDateTime(o.created_at)}
              trailing={
                <span className="flex items-center gap-3">
                  <span dir="ltr" className="tabular-nums">
                    {fmtMoney(o.total)}
                  </span>
                  <StatusBadge domain="order" status={o.status} />
                </span>
              }
            />
          ))}
        </RowList>
      </section>
    </AdminAsync>
  );
}

function Info({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-baseline justify-between gap-3">
      <dt className="shrink-0 text-sm text-ink-muted">{label}</dt>
      <dd className="min-w-0 truncate font-medium">{value || "—"}</dd>
    </div>
  );
}
