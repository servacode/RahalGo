"use client";

import { Suspense, useCallback, useEffect, useMemo, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDate, fmtMoney, errorText } from "@rahalgo/i18n";
import {
  Chips,
  Alert,
  PageContainer,
  Button,
  Input,
  Badge,
  Modal,
  Pagination,
  LoadingState,
  IconPromos,
  IconAdd,
  IconWallet,
  IconDate,
  IconStore,
  IconUsers,
  IconMoto,
  IconSearch,
  IconHourglass,
  Checkbox,
  Confirm,
  FormActions,
} from "@rahalgo/ui";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import DiscountsTab from "@/components/admin/DiscountsTab";
import PromoReferralsTab from "@/components/admin/PromoReferralsTab";
import FreeDeliveryTab from "@/components/admin/FreeDeliveryTab";
import { PageHero, type HeroTab } from "@/components/admin/PageHero";
import { Tile } from "@/components/admin/DashParts";

const m = getMessages(defaultLocale);
const O = m.admin.promosOwner;

interface Promo {
  id: string;
  code: string;
  kind: "percent" | "fixed" | "free_delivery";
  value: number;
  max_discount: number | null;
  min_order: number;
  first_order_only: boolean;
  once_per_user: boolean;
  max_uses: number | null;
  used_count: number;
  expires_at: string | null;
  active: boolean;
  status: PromoStatus;
  approval_state: "ok" | "pending" | "rejected";
  created_by_name: string;
  cost: number;
}

type PromoStatus = "active" | "expired" | "exhausted" | "paused" | "pending_approval" | "rejected";

interface Summary {
  active_codes: number;
  active_discounts: number;
  pending_approvals: number;
  platform_cost: number;
  codes: number;
  free_delivery: number;
  platform_items: number;
  referrals: number;
  stores_cost: number;
  auto_approve?: boolean;
}

const KIND_LABEL: Record<Promo["kind"], string> = {
  percent: m.admin.promos.kindPercent,
  fixed: m.admin.promos.kindFixed,
  free_delivery: m.admin.promos.kindFreeDelivery,
};

/** **الحالةُ الحقيقيّة كما يقولها الخادم** — لا علامةُ التفعيل وحدَها. */
const STATUS: Record<PromoStatus, { label: string; tone: "success" | "neutral" | "warning" | "danger" | "info" }> = {
  active: { label: O.stActive, tone: "success" },
  expired: { label: O.stExpired, tone: "neutral" },
  exhausted: { label: O.stExhausted, tone: "neutral" },
  paused: { label: O.stPaused, tone: "warning" },
  pending_approval: { label: O.stPending, tone: "info" },
  rejected: { label: O.stRejected, tone: "danger" },
};

const PER_PAGE = 20;

type PromoTab = "codes" | "discounts" | "referrals" | "free";
const TABS: PromoTab[] = ["codes", "discounts", "referrals", "free"];

export default function PromosPage() {
  return (
    <Suspense>
      <PromosHub />
    </Suspense>
  );
}

function PromosHub() {
  const { can } = useAuth();
  const router = useRouter();
  const params = useSearchParams();
  // **وإنشاءُ الأكواد وتحريرُها بقدرة العروض** (قرارُ المالك ٢٠٢٦-١٠-٠٥) — لا المحتوى.
  const isAdmin = can("offers.manage");
  const isFinance = can("finance.read");
  // **والتبويبُ في الرابط** (`?tab=`) — كالخزينة: رابطٌ من الرئيسيّة يفتح بابَه، والتحديثُ لا يُرجع لأوّله.
  const fromUrl = params.get("tab") as PromoTab | null;
  const [tab, setTabState] = useState<PromoTab>(fromUrl && TABS.includes(fromUrl) ? fromUrl : "codes");
  useEffect(() => {
    if (fromUrl && TABS.includes(fromUrl) && fromUrl !== tab) setTabState(fromUrl);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fromUrl]);
  const setTab = useCallback(
    (k: PromoTab) => {
      setTabState(k);
      router.replace(`/dashboard/promos?tab=${k}`);
    },
    [router],
  );
  const [summary, setSummary] = useState<Summary | null>(null);

  const loadSummary = useCallback(() => {
    api<Summary>("/api/v1/admin/promos/summary")
      .then(setSummary)
      // @empty-ok — **الملخّصُ زيادةٌ فوق القوائم**: إن سقط بقيت الصفحةُ تعمل.
      .catch(() => setSummary(null));
  }, []);
  useEffect(loadSummary, [loadSummary]);
  // **والمالكُ والمديرُ لا ينتظران الماليّة** (قرارُ المالك ٢٠٢٦-١٠-٢٧) — **والحكمُ من الخادم**
  // (`auto_approve` من `approval.AutoApplies`) لا من اسم دور، فلا تقول لهما النافذةُ «بيروح لموافقة المالية».
  const autoApprove = summary?.auto_approve ?? false;

  const tabs: HeroTab<PromoTab>[] = [
    { key: "codes", label: m.admin.promos.tabCodes, icon: IconPromos },
    { key: "discounts", label: m.admin.promos.tabDiscounts, icon: IconStore },
    { key: "referrals", label: O.tabReferrals, icon: IconUsers },
    { key: "free", label: m.admin.promos.tabFreeDelivery, icon: IconMoto },
  ];

  // تصميمٌ احترافيّ (المالك ٢٠٢٦-١٠-٢٦) — رأسٌ موحّدٌ بالتبويبات، أربعةُ أرقامٍ بمربّعات،
  // وكرتٌ لكلّ كودٍ بدل الجدول.
  return (
    <PageContainer>
      <PageHero<PromoTab>
        icon={IconPromos}
        title={m.admin.promos.title}
        subtitle={O.pageHint}
        tabs={tabs}
        value={tab}
        onChange={setTab}
      />

      {summary && (
        <div className="space-y-2">
          <div className="grid grid-cols-2 gap-2 lg:grid-cols-4">
            <Tile
              icon={IconPromos}
              label={O.sumActiveCodes}
              value={fmtNum(summary.active_codes)}
              tone={summary.active_codes > 0 ? "success" : "muted"}
              onClick={() => setTab("codes")}
            />
            <Tile
              icon={IconStore}
              label={O.sumActiveDiscounts}
              value={fmtNum(summary.active_discounts)}
              tone={summary.active_discounts > 0 ? "success" : "muted"}
              onClick={() => setTab("discounts")}
            />
            <Tile
              icon={IconWallet}
              label={O.sumPlatformCost}
              value={fmtMoney(summary.platform_cost)}
              tone={summary.platform_cost > 0 ? "accent" : "muted"}
            />
            <Tile
              icon={IconStore}
              label={O.sumStoresCost}
              value={fmtMoney(summary.stores_cost)}
              tone={summary.stores_cost > 0 ? "default" : "muted"}
              onClick={() => setTab("discounts")}
            />
          </div>
          {/* **ومن وين إجت كلفة المنصة** — أربعُ خاناتٍ بأرقامها، وكلُّ خانةٍ تفتح تبويبها
              (كانت سطراً تحت الرقم يُقطع على الجوّال فلا يُقرأ). */}
          <section className="surface p-3">
            <p className="mb-2 text-xs font-bold text-ink-muted">{O.costBreakdown}</p>
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
              {(
                [
                  { k: "codes", label: O.costCodes, v: summary.codes },
                  { k: "free", label: O.costFree, v: summary.free_delivery },
                  { k: "discounts", label: O.costItems, v: summary.platform_items },
                  { k: "referrals", label: O.costRefs, v: summary.referrals },
                ] as { k: PromoTab; label: string; v: number }[]
              ).map((c) => (
                <button
                  key={c.k}
                  type="button"
                  onClick={() => setTab(c.k)}
                  className={`rounded-control border px-3 py-2 text-start transition-colors hover:bg-row-hover ${
                    tab === c.k ? "border-accent" : "border-line-soft"
                  }`}
                >
                  <span className="block text-2xs text-ink-muted">{c.label}</span>
                  <span className={`block font-bold tabular-nums ${c.v > 0 ? "" : "text-ink-muted"}`}>{fmtMoney(c.v)}</span>
                </button>
              ))}
            </div>
          </section>
          {/* **وموافقاتُ المالية في صفحة الموافقات الموحّدة** (الخزينة) — لا تبويبَ هنا. */}
          {summary.pending_approvals > 0 && (
            <p className="flex items-center gap-2 rounded-control bg-field px-3 py-2 text-sm">
              <IconHourglass size={16} className="shrink-0 text-warning" />
              {isFinance ? (
                <Link href="/dashboard/treasury?tab=approvals" className="text-primary underline">
                  {O.sumPending.replace("{n}", fmtNum(summary.pending_approvals))}
                </Link>
              ) : (
                O.sumPending.replace("{n}", fmtNum(summary.pending_approvals))
              )}
            </p>
          )}
        </div>
      )}

      {tab === "codes" && <CodesTab isAdmin={isAdmin} autoApprove={autoApprove} onChanged={loadSummary} />}
      {tab === "discounts" && <DiscountsTab onChanged={loadSummary} autoApprove={autoApprove} />}
      {tab === "referrals" && <PromoReferralsTab />}
      {tab === "free" && <FreeDeliveryTab />}
    </PageContainer>
  );
}

function valueText(p: Promo): string {
  if (p.kind === "percent") {
    const cap = p.max_discount ? ` · ${O.capShort.replace("{amount}", fmtMoney(p.max_discount))}` : "";
    return `${p.value}%${cap}`;
  }
  if (p.kind === "fixed") return fmtMoney(p.value);
  return "";
}

/** **أيُّ خانةٍ تُفتح عليها النافذة** — «مدّد التاريخ» على التاريخ، و«زيد السقف» على السقف. */
type ModalFocus = "expiry" | "uses" | null;

function CodesTab({
  isAdmin,
  autoApprove,
  onChanged,
}: {
  isAdmin: boolean;
  autoApprove: boolean;
  onChanged: () => void;
}) {
  const [promos, setPromos] = useState<Promo[]>([]);
  // **ولا «لا عروض» قبل أن يصل الردّ** (تدقيقُ اللوحة ٢٠٢٦-١٠-٠٣).
  const [loaded, setLoaded] = useState(false);
  const [error, setError] = useState("");
  const [loadError, setLoadError] = useState("");
  const [editing, setEditing] = useState<Promo | "new" | null>(null);
  const [focus, setFocus] = useState<ModalFocus>(null);
  const [pausing, setPausing] = useState<Promo | null>(null);
  const [q, setQ] = useState("");
  const [status, setStatus] = useState<PromoStatus | "all">("all");
  const [page, setPage] = useState(1);
  const [busyId, setBusyId] = useState("");

  const load = useCallback(async () => {
    try {
      setPromos(await api<Promo[]>("/api/v1/admin/promos"));
      setLoaded(true);
      setLoadError("");
    } catch (err) {
      // **ولا «لا أكواد بعد» تحت خطأ** — الأكوادُ لم تصل، لا أنّها فارغة.
      setLoadError(errorText(err));
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const filtered = useMemo(() => {
    const needle = q.trim().toUpperCase();
    return promos.filter(
      (p) => (status === "all" || p.status === status) && (!needle || p.code.includes(needle)),
    );
  }, [promos, q, status]);
  const shown = filtered.slice((page - 1) * PER_PAGE, page * PER_PAGE);
  const counts = useMemo(() => {
    const c: Partial<Record<PromoStatus, number>> = {};
    for (const p of promos) c[p.status] = (c[p.status] ?? 0) + 1;
    return c;
  }, [promos]);

  function openEdit(p: Promo, f: ModalFocus = null) {
    setFocus(f);
    setEditing(p);
  }

  async function toggleActive(p: Promo) {
    if (busyId) return;
    setPausing(null);
    setBusyId(p.id);
    setError("");
    try {
      await api(`/api/v1/admin/promos/${p.id}`, {
        method: "PATCH",
        body: JSON.stringify({ active: !p.active }),
      });
      await load();
      onChanged();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusyId("");
    }
  }

  return (
    <div className="space-y-4">
      {/* ── الفلاتر ببطاقةٍ واحدة: البحث والحالة وزرُّ الكود الجديد ── */}
      <section className="surface space-y-3 p-3">
        <div className="flex flex-wrap items-end gap-2">
          <Input
            id="promo-search"
            label={O.search}
            dir="ltr"
            icon={<IconSearch size={16} />}
            placeholder={O.searchCodes}
            value={q}
            onChange={(e) => {
              setQ(e.target.value);
              setPage(1);
            }}
            wrapperClassName="min-w-0 flex-1"
          />
          {isAdmin && (
            <Button onClick={() => setEditing("new")} className="flex items-center gap-1.5">
              <IconAdd size={16} />
              {m.admin.promos.create}
            </Button>
          )}
        </div>
        <Chips
          wrap
          items={[
            { id: "all" as const, label: O.fAll, count: promos.length },
            // **والعددُ جنب كلّ حالة، والفارغةُ لا تُعرض** (إلّا المختارة) — كالتذاكر.
            ...(Object.keys(STATUS) as PromoStatus[])
              .filter((k) => (counts[k] ?? 0) > 0 || k === status)
              .map((k) => ({ id: k, label: STATUS[k].label, count: counts[k] ?? 0 })),
          ]}
          value={status}
          onChange={(k) => {
            setStatus(k);
            setPage(1);
          }}
        />
      </section>

      {error && <Alert>{error}</Alert>}

      {loadError && !loaded ? (
        <section className="surface flex flex-col items-center gap-3 p-8 text-center">
          <span className="inline-flex h-16 w-16 items-center justify-center rounded-full bg-danger-solid text-on-solid">
            <IconPromos size={28} />
          </span>
          <h2 className="heading-section">{O.loadFailed}</h2>
          <Alert>{loadError}</Alert>
          <Button variant="secondary" onClick={() => void load()}>
            {O.retry}
          </Button>
        </section>
      ) : !loaded ? (
        <LoadingState variant="list" />
      ) : shown.length === 0 ? (
        <section className="surface flex flex-col items-center gap-3 p-8 text-center">
          <span className="inline-flex h-16 w-16 items-center justify-center rounded-full bg-field text-ink-muted">
            <IconPromos size={28} />
          </span>
          <h2 className="heading-section">{promos.length === 0 ? m.admin.promos.empty : O.noMatch}</h2>
          <p className="max-w-md text-sm text-ink-muted">{promos.length === 0 ? O.codesEmptyHint : O.noMatchHint}</p>
        </section>
      ) : (
        <ul className="grid grid-cols-1 gap-3 lg:grid-cols-2 2xl:grid-cols-3">
          {shown.map((p) => (
            <PromoCard
              key={p.id}
              p={p}
              isAdmin={isAdmin}
              busy={busyId === p.id}
              onEdit={(f) => openEdit(p, f)}
              onToggle={() => (p.active ? setPausing(p) : void toggleActive(p))}
            />
          ))}
        </ul>
      )}

      {filtered.length > PER_PAGE && (
        <div className="flex justify-center">
          <Pagination page={page} total={filtered.length} perPage={PER_PAGE} onChange={setPage} />
        </div>
      )}
      {loadError && loaded && <Alert>{loadError}</Alert>}

      <Confirm
        open={pausing !== null}
        title={O.pauseCodeTitle.replace("{code}", pausing?.code ?? "")}
        body={O.pauseCodeBody}
        confirmLabel={O.pauseCode}
        busy={busyId !== ""}
        onConfirm={() => pausing && void toggleActive(pausing)}
        onCancel={() => setPausing(null)}
      />

      {editing && (
        <PromoModal
          promo={editing === "new" ? null : editing}
          focus={editing === "new" ? null : focus}
          autoApprove={autoApprove}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            void load();
            onChanged();
          }}
        />
      )}
    </div>
  );
}

const STATUS_TINT: Record<PromoStatus, string> = {
  active: "border-success text-success",
  expired: "border-line text-ink-muted",
  exhausted: "border-line text-ink-muted",
  paused: "border-warning text-warning",
  pending_approval: "border-accent text-ink",
  rejected: "border-danger text-danger",
};

/**
 * **كرتُ الكود** — تصميمٌ احترافيّ (المالك ٢٠٢٦-١٠-٢٦): الكودُ كبيرٌ بقسيمةٍ منقّطة، وقيمةُ الخصم،
 * والحالةُ ملوّنة، والانتهاء، والاستخداماتُ بشريط، ومين عمله وقديش كلّف — والأفعال.
 */
function PromoCard({
  p,
  isAdmin,
  busy,
  onEdit,
  onToggle,
}: {
  p: Promo;
  isAdmin: boolean;
  busy: boolean;
  onEdit: (focus?: ModalFocus) => void;
  onToggle: () => void;
}) {
  const st = STATUS[p.status];
  const usedPct = p.max_uses ? Math.min(100, Math.round((p.used_count / p.max_uses) * 100)) : 0;
  // **والمنتهي وخالصُ السقف لا يُفعَّلان ولا يُوقفان** (تدقيقُ العروض ٢٠٢٦-١٠-٢٧): الخادمُ يردّ
  // `promo_expired_activate`، و«إيقاف» كودٍ لا يعمل أصلاً يُلخبط — **فالفعلُ ما يُرجعه للعمل.**
  const fix: ModalFocus = p.status === "expired" ? "expiry" : p.status === "exhausted" ? "uses" : null;
  const canToggle = !fix && p.status !== "pending_approval" && p.status !== "rejected";
  return (
    <li className="surface flex flex-col gap-3 p-4">
      <div className="flex items-start justify-between gap-3">
        <span
          dir="ltr"
          className={`inline-flex min-w-0 items-center gap-2 rounded-control border-2 border-dashed bg-field px-3 py-1.5 ${STATUS_TINT[p.status]}`}
        >
          <IconPromos size={16} className="shrink-0" />
          <span className="figure truncate font-mono tracking-wider">{p.code}</span>
        </span>
        <Badge variant={st.tone} className="shrink-0">
          {st.label}
        </Badge>
      </div>

      <div className="flex items-end justify-between gap-2 rounded-control bg-field px-3 py-2.5">
        <span className="min-w-0">
          <span className="block text-2xs text-ink-muted">{p.kind === "free_delivery" ? O.kindLabel : KIND_LABEL[p.kind]}</span>
          <span className="figure block tabular-nums">{valueText(p) || KIND_LABEL[p.kind]}</span>
        </span>
        <span className="text-end">
          <span className="block text-2xs text-ink-muted">{m.admin.promos.minOrder}</span>
          <span className="block text-sm tabular-nums">{p.min_order > 0 ? fmtMoney(p.min_order) : O.noMinOrder}</span>
        </span>
      </div>

      <div>
        <div className="flex items-baseline justify-between gap-2 text-sm">
          <span className="text-2xs text-ink-muted">{m.admin.promos.usedCount}</span>
          <span className="tabular-nums">
            {p.max_uses
              ? O.usageOf.replace("{used}", fmtNum(p.used_count)).replace("{max}", fmtNum(p.max_uses))
              : O.usageUnlimited.replace("{used}", fmtNum(p.used_count))}
          </span>
        </div>
        {p.max_uses ? (
          <div className="mt-1.5 h-1.5 overflow-hidden rounded-full bg-field" aria-hidden>
            <div
              className={`h-full rounded-full ${usedPct >= 100 ? "bg-danger-solid" : "bg-accent"}`}
              style={{ width: `${usedPct}%` }}
            />
          </div>
        ) : null}
      </div>

      {(p.first_order_only || p.once_per_user) && (
        <div className="flex flex-wrap gap-1.5">
          {p.first_order_only && <Badge variant="neutral">{m.admin.promos.firstOrderOnly}</Badge>}
          {p.once_per_user && <Badge variant="neutral">{m.admin.promos.oncePerUser}</Badge>}
        </div>
      )}

      <div className="grid grid-cols-1 gap-x-3 gap-y-1 text-2xs text-ink-muted sm:grid-cols-2">
        <span className="flex items-center gap-1">
          <IconDate size={12} className="shrink-0" />
          {O.endsCol}: {p.expires_at ? fmtDate(p.expires_at) : m.admin.promos.noExpiry}
        </span>
        <span>
          {O.cost}: <span className="tabular-nums text-ink">{fmtMoney(p.cost)}</span>
        </span>
        <span className="sm:col-span-2">
          {O.createdBy}: {p.created_by_name || "—"}
        </span>
      </div>

      {fix && <p className="text-xs font-bold text-warning">{fix === "expiry" ? O.expiredNote : O.exhaustedNote}</p>}

      {isAdmin && (
        <div className={`mt-auto grid gap-2 border-t border-line-soft pt-3 ${canToggle || fix ? "grid-cols-2" : "grid-cols-1"}`}>
          <Button variant="secondary" className="w-full" onClick={() => onEdit()}>
            {O.edit}
          </Button>
          {fix && (
            <Button className="w-full" onClick={() => onEdit(fix)}>
              {fix === "expiry" ? O.extendDate : O.raiseCap}
            </Button>
          )}
          {canToggle && (
            <Button variant={p.active ? "danger" : "primary"} className="w-full" disabled={busy} onClick={onToggle}>
              {busy ? O.busy : p.active ? O.pauseCode : O.activateCode}
            </Button>
          )}
        </div>
      )}
    </li>
  );
}

/** **حدّا الموافقة** — بحرف الخادم (`offers.FreePercent` و`offers.FreeUses`). */
const FREE_PERCENT = 20;
const FREE_USES = 50;

/**
 * ══════════════════════════════════════════════════════════════════════
 * **لماذا يحتاج الكودُ موافقةَ الماليّة — بأسبابه** (قرارُ المالك ٢٠٢٦-١٠-١٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كان سطرٌ ثابتٌ يقول «لحد 20% و50 استخدام»** — فكودٌ ثابتٌ بلا حدٍّ أدنى يذهب
 * للماليّة ولا يُقال لماذا. **والأسبابُ بحرف `needsApproval` في الخادم**: بلا سقف ·
 * فوق ٥٠ استخداماً · نسبةٌ فوق ٢٠ · مبلغٌ ثابتٌ فوق ٢٠٪ من الحدّ الأدنى (والصفرُ منه).
 */
function approvalReasons(kind: Promo["kind"], value: number, minOrder: number, uses: number | null): string[] {
  const out: string[] = [];
  if (uses === null) out.push(O.whyNoCap);
  else if (uses > FREE_USES) out.push(O.whyManyUses.replace("{n}", fmtNum(FREE_USES)));
  if (kind === "percent" && value > FREE_PERCENT) out.push(O.whyPercent.replace("{n}", fmtNum(FREE_PERCENT)));
  if (kind === "fixed" && value * 100 > minOrder * FREE_PERCENT) {
    out.push(
      minOrder <= 0
        ? O.whyFixedNoMin
        : O.whyFixedOverMin
            .replace("{n}", fmtNum(FREE_PERCENT))
            .replace("{max}", fmtMoney(Math.floor((minOrder * FREE_PERCENT) / 100))),
    );
  }
  return out;
}

/** **يومُ الانتهاء كما يُعرض في الحقل** — بتوقيت دمشق. */
function damascusDay(iso: string | null): string {
  if (!iso) return "";
  const d = new Date(new Date(iso).getTime() + 3 * 3600 * 1000);
  return d.toISOString().slice(0, 10);
}

function PromoModal({
  promo,
  focus,
  autoApprove,
  onClose,
  onSaved,
}: {
  promo: Promo | null;
  focus: ModalFocus;
  autoApprove: boolean;
  onClose: () => void;
  onSaved: () => void;
}) {
  const editing = promo !== null;
  const [code, setCode] = useState(promo?.code ?? "");
  const [kind, setKind] = useState<Promo["kind"]>(promo?.kind ?? "percent");
  const [value, setValue] = useState(promo ? String(promo.value) : "");
  const [cap, setCap] = useState(promo?.max_discount ? String(promo.max_discount) : "");
  const [minOrder, setMinOrder] = useState(promo ? String(promo.min_order) : "0");
  const [maxUses, setMaxUses] = useState(promo?.max_uses ? String(promo.max_uses) : "");
  const [expiresOn, setExpiresOn] = useState(damascusDay(promo?.expires_at ?? null));
  const [firstOnly, setFirstOnly] = useState(promo?.first_order_only ?? false);
  const [oncePerUser, setOncePerUser] = useState(promo?.once_per_user ?? true);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false);

  // **«مدّد التاريخ» و«زيد السقف» يفتحان النافذةَ على خانتهما.**
  useEffect(() => {
    if (!focus) return;
    const t = setTimeout(() => document.getElementById(focus === "expiry" ? "p-exp" : "p-max")?.focus(), 50);
    return () => clearTimeout(t);
  }, [focus]);

  const uses = maxUses === "" ? null : Number(maxUses);
  const reasons = approvalReasons(kind, Number(value) || 0, Number(minOrder) || 0, uses);
  // **ورفعُ قيد الجمهور يعيد الكودَ للماليّة** — إن كان فوق الحدّ (٢٠٢٦-١٠-١٥).
  const broadening =
    editing && ((promo.first_order_only && !firstOnly) || (promo.once_per_user && !oncePerUser));
  if (broadening && reasons.length > 0) reasons.push(O.whyBroaden);
  const originalDay = damascusDay(promo?.expires_at ?? null);
  const perUse = kind === "percent" ? Number(cap) || 0 : kind === "fixed" ? Number(value) || 0 : 0;
  const preview =
    uses === null
      ? O.costPreviewUnlimited
      : perUse > 0
        ? O.costPreview.replace("{amount}", fmtMoney(perUse * uses))
        : "";

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    // **والضغطُ الثاني لا يُرسل ثانيةً** — والزرُّ مُعطَّلٌ وهو يُرسل.
    if (busy) return;
    setBusy(true);
    setError("");
    try {
      const common = {
        value: kind === "free_delivery" ? 0 : Number(value) || 0,
        max_discount: kind === "percent" && cap !== "" ? Number(cap) : null,
        min_order: Number(minOrder) || 0,
        first_order_only: firstOnly,
        once_per_user: oncePerUser,
      };
      let saved: Promo;
      if (editing) {
        saved = await api<Promo>(`/api/v1/admin/promos/${promo.id}`, {
          method: "PATCH",
          body: JSON.stringify({
            ...common,
            ...(uses === null ? { clear_max_uses: true } : { max_uses: uses }),
            // **والتاريخُ يُرسَل إن تبدّل وحدَه** (قرارُ المالك ٢٠٢٦-١٠-١٥) — كودٌ منتهٍ كان
            // لا يُعدَّل فيه حرف: النافذةُ تعيد تاريخَه فيُردّ «تاريخُ الانتهاء مضى».
            ...(expiresOn === originalDay
              ? {}
              : expiresOn === ""
                ? { clear_expiry: true }
                : { expires_on: expiresOn }),
          }),
        });
      } else {
        saved = await api<Promo>("/api/v1/admin/promos", {
          method: "POST",
          body: JSON.stringify({
            ...common,
            code: code.trim(),
            kind,
            max_uses: uses,
            expires_on: expiresOn === "" ? null : expiresOn,
          }),
        });
      }
      if (saved.approval_state === "pending") {
        setNotice(O.pendingSaved);
        setTimeout(onSaved, 1500);
        return;
      }
      onSaved();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={editing ? O.editTitle : m.admin.promos.createTitle}>
      <form onSubmit={submit} className="space-y-4">
        <Input
          id="p-code"
          label={m.admin.promos.code}
          dir="ltr"
          required
          disabled={editing}
          value={code}
          onChange={(e) => setCode(e.target.value.toUpperCase())}
          className="font-mono uppercase"
          placeholder="WELCOME20"
        />
        {/* **النوعُ بأزرار لا قائمةٍ منسدلة** — ثلاثةُ خياراتٍ تُرى كلُّها (كـ«طريقة الخصم» في الخصومات).
            **ولا يتبدّل بعد الإنشاء** — يُعرض شارةً. */}
        <div className="space-y-1.5">
          <p className="text-sm font-medium">{O.kindLabel}</p>
          {editing ? (
            <Badge variant="accent">{KIND_LABEL[kind]}</Badge>
          ) : (
            <Chips
              wrap
              items={[
                { id: "percent" as const, label: m.admin.promos.kindPercent },
                { id: "fixed" as const, label: m.admin.promos.kindFixed },
                { id: "free_delivery" as const, label: m.admin.promos.kindFreeDelivery },
              ]}
              value={kind}
              onChange={setKind}
            />
          )}
        </div>
        {kind !== "free_delivery" && (
          <Input
            id="p-value"
            label={kind === "percent" ? `${m.admin.promos.value} (${O.percentHint})` : O.fAmount}
            type="number"
            min="1"
            max={kind === "percent" ? "90" : undefined}
            required
            value={value}
            onChange={(e) => setValue(e.target.value)}
          />
        )}
        {kind === "percent" && (
          <Input
            id="p-cap"
            label={O.maxDiscount}
            type="number"
            min="1"
            required
            value={cap}
            onChange={(e) => setCap(e.target.value)}
          />
        )}
        {kind === "percent" && <p className="-mt-2 text-xs text-ink-muted">{O.maxDiscountHint}</p>}
        <div className="grid grid-cols-2 gap-3">
          <Input
            id="p-min"
            label={m.admin.promos.minOrder}
            type="number"
            min="0"
            value={minOrder}
            onChange={(e) => setMinOrder(e.target.value)}
          />
          <Input
            id="p-max"
            label={m.admin.promos.maxUses}
            type="number"
            min="1"
            value={maxUses}
            onChange={(e) => setMaxUses(e.target.value)}
          />
        </div>
        <Input
          id="p-exp"
          label={O.expiresOn}
          type="date"
          value={expiresOn}
          onChange={(e) => setExpiresOn(e.target.value)}
        />
        <div className="flex flex-wrap gap-4">
          <Checkbox
            id="promo-first-only"
            checked={firstOnly}
            onChange={(e) => setFirstOnly(e.target.checked)}
            label={m.admin.promos.firstOrderOnly}
          />
          <Checkbox
            id="promo-once-per-user"
            checked={oncePerUser}
            onChange={(e) => setOncePerUser(e.target.checked)}
            label={m.admin.promos.oncePerUser}
          />
        </div>
        {preview && <p className="text-sm font-medium">{preview}</p>}
        <p className="text-xs text-ink-muted">
          {reasons.length === 0
            ? O.approvalNone
            : (autoApprove ? O.approvalWhyAuto : O.approvalWhy).replace("{reasons}", reasons.join(m.common.listSep))}
        </p>
        {notice && <Alert tone="info">{notice}</Alert>}
        {error && <Alert>{error}</Alert>}
        <FormActions submit onCancel={onClose} busy={busy} />
      </form>
    </Modal>
  );
}
