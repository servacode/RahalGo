"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDate, fmtMoney, errorText } from "@rahalgo/i18n";
import {
  Tabs,
  Chips,
  Alert,
  PageHeader,
  Button,
  Input,
  Select,
  Badge,
  Modal,
  DataView,
  ViewToggle,
  useViewMode,
  Pagination,
  StatGrid,
  StatCard,
  type DataColumn,
  IconPromos,
  IconAdd,
  IconStatus,
  IconWallet,
  IconDate,
  Checkbox,
  FormActions,
} from "@rahalgo/ui";
import Link from "next/link";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import DiscountsTab from "@/components/admin/DiscountsTab";
import PromoReferralsTab from "@/components/admin/PromoReferralsTab";

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

export default function PromosPage() {
  const { can } = useAuth();
  // **وإنشاءُ الأكواد وتحريرُها بقدرة العروض** (قرارُ المالك ٢٠٢٦-١٠-٠٥) — لا المحتوى.
  const isAdmin = can("offers.manage");
  const isFinance = can("finance.read");
  const [tab, setTab] = useState<"codes" | "discounts" | "referrals">("codes");
  const [summary, setSummary] = useState<Summary | null>(null);

  const loadSummary = useCallback(() => {
    api<Summary>("/api/v1/admin/promos/summary")
      .then(setSummary)
      // @empty-ok — **الملخّصُ زيادةٌ فوق الجداول**: إن سقط بقيت الصفحةُ تعمل.
      .catch(() => setSummary(null));
  }, []);
  useEffect(loadSummary, [loadSummary]);

  const tabs = [
    { key: "codes" as const, label: m.admin.promos.tabCodes },
    { key: "discounts" as const, label: m.admin.promos.tabDiscounts },
    { key: "referrals" as const, label: O.tabReferrals },
  ];

  return (
    <div>
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <PageHeader icon={IconPromos} title={m.admin.promos.title} />
        <Tabs items={tabs} value={tab} onChange={setTab} />
      </div>

      {summary && (
        <div className="mb-6">
          <StatGrid>
            <StatCard icon={IconPromos} label={O.sumActiveCodes} value={fmtNum(summary.active_codes)} />
            <StatCard icon={IconPromos} label={O.sumActiveDiscounts} value={fmtNum(summary.active_discounts)} />
            <StatCard
              icon={IconWallet}
              label={O.sumPlatformCost}
              value={fmtMoney(summary.platform_cost)}
              sub={O.sumPlatformCostSub
                .replace("{codes}", fmtMoney(summary.codes))
                .replace("{free}", fmtMoney(summary.free_delivery))
                .replace("{items}", fmtMoney(summary.platform_items))
                .replace("{refs}", fmtMoney(summary.referrals))}
              tone={summary.platform_cost > 0 ? "accent" : "muted"}
            />
            <StatCard
              icon={IconWallet}
              label={O.sumStoresCost}
              value={fmtMoney(summary.stores_cost)}
              tone={summary.stores_cost > 0 ? "default" : "muted"}
            />
          </StatGrid>
          {/* **وموافقاتُ المالية في صفحة الموافقات الموحّدة** (الخزينة) — لا تبويبَ هنا. */}
          {summary.pending_approvals > 0 && (
            <p className="mt-2 text-sm text-ink-muted">
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

      {tab === "codes" && <CodesTab isAdmin={isAdmin} onChanged={loadSummary} />}
      {tab === "discounts" && <DiscountsTab onChanged={loadSummary} />}
      {tab === "referrals" && <PromoReferralsTab />}
    </div>
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

function CodesTab({ isAdmin, onChanged }: { isAdmin: boolean; onChanged: () => void }) {
  const [promos, setPromos] = useState<Promo[]>([]);
  // **ولا «لا عروض» قبل أن يصل الردّ** (تدقيقُ اللوحة ٢٠٢٦-١٠-٠٣).
  const [loaded, setLoaded] = useState(false);
  const [error, setError] = useState("");
  const [editing, setEditing] = useState<Promo | "new" | null>(null);
  const [view, setView] = useViewMode("promos");
  const [q, setQ] = useState("");
  const [status, setStatus] = useState<PromoStatus | "all">("all");
  const [page, setPage] = useState(1);
  const [busyId, setBusyId] = useState("");

  const load = useCallback(async () => {
    try {
      setPromos(await api<Promo[]>("/api/v1/admin/promos"));
      setLoaded(true);
      setError("");
    } catch (err) {
      setError(errorText(err));
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

  async function toggleActive(p: Promo) {
    if (busyId) return;
    setBusyId(p.id);
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

  const columns: DataColumn<Promo>[] = [
    {
      id: "code",
      header: m.admin.promos.code,
      icon: <IconPromos />,
      primary: true,
      cell: (p) => (
        <span dir="ltr" className="font-mono font-bold tracking-wider">
          {p.code}
        </span>
      ),
    },
    {
      id: "kind",
      header: m.admin.promos.kind,
      cell: (p) => (
        <Badge variant="primary">
          {KIND_LABEL[p.kind]}
          {valueText(p) && ` ${valueText(p)}`}
        </Badge>
      ),
    },
    {
      id: "min",
      header: m.admin.promos.minOrder,
      icon: <IconWallet />,
      cell: (p) => fmtMoney(p.min_order),
    },
    {
      id: "uses",
      header: m.admin.promos.usedCount,
      cell: (p) => `${fmtNum(p.used_count)}${p.max_uses ? ` / ${fmtNum(p.max_uses)}` : ""}`,
    },
    {
      id: "expiry",
      header: m.admin.promos.expiresAt,
      icon: <IconDate />,
      cell: (p) =>
        p.expires_at ? (
          fmtDate(p.expires_at)
        ) : (
          <span className="text-ink-muted">{m.admin.promos.noExpiry}</span>
        ),
    },
    {
      id: "status",
      header: O.statusCol,
      icon: <IconStatus />,
      cell: (p) => <Badge variant={STATUS[p.status].tone}>{STATUS[p.status].label}</Badge>,
    },
    { id: "cost", header: O.cost, cell: (p) => fmtMoney(p.cost) },
    { id: "by", header: O.createdBy, cell: (p) => p.created_by_name || "—" },
  ];

  return (
    <div>
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        {isAdmin ? (
          <Button onClick={() => setEditing("new")} className="flex items-center gap-1.5">
            <IconAdd size={16} />
            {m.admin.promos.create}
          </Button>
        ) : (
          <span />
        )}
        <ViewToggle
          view={view}
          onChange={setView}
          tableLabel={m.common.viewTable}
          cardsLabel={m.common.viewCards}
        />
      </div>
      <div className="mb-4 space-y-3">
        <Input
          id="promo-search"
          label={O.search}
          dir="ltr"
          placeholder={O.searchCodes}
          value={q}
          onChange={(e) => {
            setQ(e.target.value);
            setPage(1);
          }}
        />
        <Chips
          wrap
          items={[
            { id: "all" as const, label: O.fAll },
            ...(Object.keys(STATUS) as PromoStatus[]).map((k) => ({ id: k, label: STATUS[k].label })),
          ]}
          value={status}
          onChange={(k) => {
            setStatus(k);
            setPage(1);
          }}
        />
      </div>
      {error && <Alert className="mb-4">{error}</Alert>}
      <DataView
        items={shown}
        loading={!loaded && !error}
        getKey={(p) => p.id}
        columns={columns}
        view={view}
        empty={m.admin.promos.empty}
        actions={
          isAdmin
            ? (p) => (
                <div className="flex gap-2">
                  <Button variant="secondary" onClick={() => setEditing(p)}>
                    {O.edit}
                  </Button>
                  {p.status !== "pending_approval" && p.status !== "rejected" && (
                    <Button
                      variant={p.active ? "danger" : "secondary"}
                      disabled={busyId === p.id}
                      onClick={() => void toggleActive(p)}
                    >
                      {busyId === p.id
                        ? O.busy
                        : p.active
                          ? m.admin.merchants.deactivate
                          : m.admin.merchants.activate}
                    </Button>
                  )}
                </div>
              )
            : undefined
        }
      />
      <Pagination page={page} total={filtered.length} perPage={PER_PAGE} onChange={setPage} />
      {editing && (
        <PromoModal
          promo={editing === "new" ? null : editing}
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

/** **يومُ الانتهاء كما يُعرض في الحقل** — بتوقيت دمشق. */
function damascusDay(iso: string | null): string {
  if (!iso) return "";
  const d = new Date(new Date(iso).getTime() + 3 * 3600 * 1000);
  return d.toISOString().slice(0, 10);
}

function PromoModal({
  promo,
  onClose,
  onSaved,
}: {
  promo: Promo | null;
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

  const uses = maxUses === "" ? null : Number(maxUses);
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
            ...(expiresOn === "" ? { clear_expiry: true } : { expires_on: expiresOn }),
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
        <div className="grid grid-cols-2 gap-3">
          <Select
            id="p-kind"
            label={m.admin.promos.kind}
            value={kind}
            disabled={editing}
            onChange={(e) => setKind(e.target.value as Promo["kind"])}
          >
            <option value="percent">{m.admin.promos.kindPercent}</option>
            <option value="fixed">{m.admin.promos.kindFixed}</option>
            <option value="free_delivery">{m.admin.promos.kindFreeDelivery}</option>
          </Select>
          <Input
            id="p-value"
            label={kind === "percent" ? `${m.admin.promos.value} (${O.percentHint})` : m.admin.promos.value}
            type="number"
            min="1"
            max={kind === "percent" ? "90" : undefined}
            disabled={kind === "free_delivery"}
            required={kind !== "free_delivery"}
            value={kind === "free_delivery" ? "" : value}
            onChange={(e) => setValue(e.target.value)}
          />
        </div>
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
        <p className="text-xs text-ink-muted">{O.approvalHint}</p>
        {notice && <Alert tone="info">{notice}</Alert>}
        {error && <Alert>{error}</Alert>}
        <FormActions submit onCancel={onClose} busy={busy} />
      </form>
    </Modal>
  );
}
