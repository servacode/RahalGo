"use client";

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtTime } from "@rahalgo/i18n";
import {
  Alert,
  useLiveRefresh,
  PageHeader,
  Button,
  Input,
  Badge,
  Modal,
  DataView,
  ViewToggle,
  useViewMode,
  type DataColumn,
  IconDriver,
  IconPhone,
  IconUser,
  IconWallet,
  IconOrder,
  IconStatus,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);

interface Driver {
  id: string;
  phone: string;
  full_name: string;
  status: string;
  on_shift: boolean;
  shift_started_at: string | null;
  cash_held: number;
  open_orders: number;
  delivered_today: number;
}

interface CashEntry {
  id: number;
  amount: number;
  kind: string;
  ref: string;
  note: string;
  created_at: string;
}

function translateKey(key: string): string {
  let node: unknown = m;
  for (const part of key.split(".")) {
    if (typeof node !== "object" || node === null) return m.errors.internal;
    node = (node as Record<string, unknown>)[part];
  }
  return typeof node === "string" ? node : m.errors.internal;
}
function errText(err: unknown): string {
  return err instanceof ApiError ? translateKey(err.body.message_key) : m.errors.internal;
}

export default function DriversTable() {
  const { user: me } = useAuth();
  const canSettle = !!me?.roles.some((r) => r === "admin" || r === "finance");

  const [drivers, setDrivers] = useState<Driver[]>([]);
  const [cashLimit, setCashLimit] = useState(0);
  const [error, setError] = useState("");
  const [boxFor, setBoxFor] = useState<Driver | null>(null);
  /** **السائقُ الذي يُغلَق دوامُه** — بكلمةٍ تصله، لا بصمت. */
  const [endShiftFor, setEndShiftFor] = useState<Driver | null>(null);
  const [view, setView] = useViewMode("drivers");

  const load = useCallback(async () => {
    try {
      const res = await api<{ drivers: Driver[]; cash_limit: number }>("/api/v1/admin/drivers");
      setDrivers(res.drivers);
      setCashLimit(res.cash_limit);
      setError("");
    } catch (err) {
      setError(errText(err));
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  useLiveRefresh(["wallet", "order"], load);

  useLiveRefresh(["order", "account", "wallet"], load);

  function cashBadge(d: Driver) {
    const ratio = cashLimit > 0 ? d.cash_held / cashLimit : 0;
    const variant = ratio >= 1 ? "danger" : ratio >= 0.7 ? "warning" : "success";
    return (
      <span className="flex flex-col items-end gap-0.5 sm:items-start">
        <Badge variant={variant}>
          {fmtNum(d.cash_held)} {m.common.currency}
        </Badge>
        {ratio >= 1 && <span className="text-xs text-danger">{m.admin.drivers.overLimit}</span>}
        {ratio >= 0.7 && ratio < 1 && (
          <span className="text-xs text-warning">{m.admin.drivers.nearLimit}</span>
        )}
      </span>
    );
  }

  const columns: DataColumn<Driver>[] = [
    {
      id: "name",
      header: m.admin.users.table.name,
      icon: <IconUser />,
      primary: true,
      cell: (d) => d.full_name || "—",
    },
    {
      id: "phone",
      header: m.admin.users.table.phone,
      icon: <IconPhone />,
      primary: true,
      cell: (d) => (
        <span dir="ltr" className="font-medium">
          {d.phone}
        </span>
      ),
    },
    {
      // الدوام أوّل ما تسأل عنه العمليات: «من يعمل الآن؟». وكانت تسأله بالهاتف
      // بينما الجواب في قاعدتها منذ أن بُني علَم السائق.
      id: "shift",
      header: m.terms.onShift,
      icon: <IconDriver />,
      primary: true,
      cell: (d) =>
        d.on_shift ? (
          <Badge variant="success">
            {d.shift_started_at ? fmtTime(d.shift_started_at) : m.terms.onShift}
          </Badge>
        ) : (
          <Badge variant="neutral">{m.terms.offShift}</Badge>
        ),
    },
    {
      id: "cash",
      header: m.admin.drivers.cashHeld,
      icon: <IconWallet />,
      cell: cashBadge,
    },
    {
      id: "open",
      header: m.admin.drivers.openOrders,
      icon: <IconOrder />,
      cell: (d) => (d.open_orders > 0 ? <Badge variant="primary">{d.open_orders}</Badge> : "—"),
    },
    {
      id: "today",
      header: m.admin.drivers.deliveredToday,
      cell: (d) => fmtNum(d.delivered_today),
    },
    {
      id: "status",
      header: m.admin.users.table.status,
      icon: <IconStatus />,
      cell: (d) => (
        <Badge variant={d.status === "active" ? "success" : "danger"}>
          {d.status === "active" ? m.admin.users.active : m.admin.users.blocked}
        </Badge>
      ),
    },
  ];

  return (
    <div>
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <PageHeader icon={IconDriver} title={m.admin.drivers.title} />
        <div className="flex items-center gap-3">
          <Badge variant="neutral">
            {m.admin.drivers.cashLimit}: {fmtNum(cashLimit)} {m.common.currency}
          </Badge>
          <ViewToggle
            view={view}
            onChange={setView}
            tableLabel={m.common.viewTable}
            cardsLabel={m.common.viewCards}
          />
        </div>
      </div>

      {error && (
        <Alert className="mb-4">{error}</Alert>
      )}

      <DataView
        items={drivers}
        getKey={(d) => d.id}
        columns={columns}
        view={view}
        empty={m.admin.drivers.empty}
        actions={(d) => (
          <>
            <Button variant="secondary" onClick={() => setBoxFor(d)} className="flex items-center gap-1.5">
              <IconWallet size={15} />
              {m.admin.drivers.cashBox}
            </Button>
            {/* **إغلاقُ دوامٍ نُسي.**

                علَمُ الدوام بيد السائق وحدَه، **ومن ذهب إلى بيته ونسي أن
                يُطفئه يبقى في الدور**: يُعرض عليه كلُّ طلبٍ خمساً وأربعين
                ثانيةً ثمّ يمضي — **فكلُّ طلبٍ يتأخّر بمقدار غيابه.**

                **ولا يُفتَح من هنا**: فتحُ الدوام إقرارٌ من إنسانٍ بأنّه جاهز،
                والمنصةُ تعلم أنّه لا يستجيب ولا تعلم أنّه جاهز. */}
            {d.on_shift && d.open_orders === 0 && (
              <Button variant="ghost" onClick={() => setEndShiftFor(d)}>
                {m.admin.drivers.endShift}
              </Button>
            )}
          </>
        )}
      />

      {boxFor && (
        <CashBoxModal
          driver={boxFor}
          canSettle={canSettle}
          onClose={() => setBoxFor(null)}
          onChanged={load}
        />
      )}
      {endShiftFor && (
        <EndShiftModal
          driver={endShiftFor}
          onClose={() => setEndShiftFor(null)}
          onDone={() => {
            setEndShiftFor(null);
            void load();
          }}
        />
      )}
    </div>
  );
}

/**
 * إغلاقُ الدوام — **بكلمةٍ تصل صاحبَه.**
 *
 * **ومن أُغلق دوامُه بلا علمه يظنّ أنّ النظام أعطبه** — فيشتكي، أو يظنّ أن لا
 * طلباتِ اليوم فيمضي إلى بيته. **والكلمةُ ليست تجميلاً: هي الفرقُ بين إجراءٍ
 * وبين عطبٍ يبدو عشوائياً.**
 */
function EndShiftModal({
  driver,
  onClose,
  onDone,
}: {
  driver: Driver;
  onClose: () => void;
  onDone: () => void;
}) {
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");

  async function submit() {
    setBusy(true);
    setErr("");
    try {
      await api(`/api/v1/admin/drivers/${driver.id}/end-shift`, {
        method: "POST",
        body: JSON.stringify({ note: note.trim() }),
      });
      onDone();
    } catch (e) {
      setErr(errText(e));
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={`${m.admin.drivers.endShift}: ${driver.full_name}`}>
      <div className="space-y-3">
        <p className="text-sm text-ink-muted">{m.admin.drivers.endShiftHint}</p>
        <Input
          id="end-shift-note"
          label={m.admin.drivers.endShiftNote}
          value={note}
          onChange={(e) => setNote(e.target.value)}
        />
        {err && <p className="text-sm text-danger">{err}</p>}
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={onClose}>
            {m.common.cancel}
          </Button>
          <Button variant="danger" disabled={busy} onClick={() => void submit()}>
            {m.admin.drivers.endShift}
          </Button>
        </div>
      </div>
    </Modal>
  );
}

function CashBoxModal({
  driver,
  canSettle,
  onClose,
  onChanged,
}: {
  driver: Driver;
  canSettle: boolean;
  onClose: () => void;
  onChanged: () => Promise<void> | void;
}) {
  const KINDS: Record<string, string> = m.admin.drivers.entryKinds;
  const [held, setHeld] = useState<number | null>(null);
  const [limit, setLimit] = useState(0);
  const [entries, setEntries] = useState<CashEntry[]>([]);
  const [amount, setAmount] = useState("");
  const [note, setNote] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      const st = await api<{ held: number; limit: number; entries: CashEntry[] }>(
        `/api/v1/admin/drivers/${driver.id}/cash`,
      );
      setHeld(st.held);
      setLimit(st.limit);
      setEntries(st.entries);
      setError("");
    } catch (err) {
      setError(errText(err));
    }
  }, [driver.id]);

  useEffect(() => {
    void load();
  }, [load]);

  async function settle(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/drivers/${driver.id}/settle`, {
        method: "POST",
        body: JSON.stringify({ amount: Number(amount) || 0, note }),
      });
      setAmount("");
      setNote("");
      await load();
      await onChanged();
    } catch (err) {
      setError(errText(err));
    } finally {
      setBusy(false);
    }
  }

  const ratio = limit > 0 && held !== null ? Math.min(held / limit, 1) : 0;

  return (
    <Modal
      open
      onClose={onClose}
      size="lg"
      title={`${m.admin.drivers.boxTitle}: ${driver.full_name || driver.phone}`}
    >
      <div className="mb-4 rounded-card bg-primary-tint p-4">
        <div className="flex items-center justify-between">
          <span className="flex items-center gap-2 font-medium text-primary-dark">
            <IconWallet size={18} />
            {m.admin.drivers.cashHeld}
          </span>
          <span className="figure text-primary-dark">
            {held === null ? "…" : `${fmtNum(held)} ${m.common.currency}`}
          </span>
        </div>
        {/* شريط السقف */}
        <div className="mt-3 h-2 overflow-hidden rounded-badge bg-surface">
          <div
            className={`h-full transition-all ${ratio >= 1 ? "bg-danger" : ratio >= 0.7 ? "bg-warning" : "bg-success"}`}
            style={{ width: `${ratio * 100}%` }}
          />
        </div>
        <p className="mt-1 text-end text-xs text-ink-muted">
          {m.admin.drivers.cashLimit}: {fmtNum(limit)}
        </p>
      </div>

      {canSettle && (
        <form onSubmit={settle} className="mb-4 flex items-end gap-2 rounded-card border border-line p-4">
          <div className="flex-1">
            <Input
              id="s-amount"
              label={`${m.admin.drivers.settleAmount} (${m.common.currency})`}
              type="number"
              min="1"
              required
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
            />
          </div>
          <Button
            type="button"
            variant="ghost"
            onClick={() => setAmount(String(held ?? 0))}
            disabled={!held}
          >
            {m.admin.drivers.settleAll}
          </Button>
          <div className="flex-1">
            <Input
              id="s-note"
              label={m.admin.users.noteField}
              value={note}
              onChange={(e) => setNote(e.target.value)}
            />
          </div>
          <Button type="submit" disabled={busy || !amount}>
            {m.admin.drivers.settle}
          </Button>
        </form>
      )}

      {error && (
        <Alert className="mb-3">{error}</Alert>
      )}

      <h3 className="mb-2 text-sm font-bold">{m.admin.drivers.entriesHistory}</h3>
      {entries.length === 0 ? (
        <p className="rounded-control bg-field p-4 text-center text-sm text-ink-muted">
          {m.admin.drivers.noEntries}
        </p>
      ) : (
        <ul className="max-h-60 space-y-1.5 overflow-y-auto">
          {entries.map((e) => (
            <li
              key={e.id}
              className="flex items-center justify-between rounded-control border border-line px-3 py-2 text-sm"
            >
              <span className="flex items-center gap-2">
                <Badge variant={e.amount > 0 ? "primary" : "success"}>{KINDS[e.kind] ?? e.kind}</Badge>
                {e.note && <span className="text-xs text-ink-muted">{e.note}</span>}
              </span>
              <span className={`font-bold ${e.amount > 0 ? "text-primary-dark" : "text-success"}`}>
                {e.amount > 0 ? "+" : ""}
                {fmtNum(e.amount)}
              </span>
            </li>
          ))}
        </ul>
      )}
    </Modal>
  );
}
