"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أجزاءُ ملفّ الحساب — قراراتُ المالك ٢٠٢٦-١٠-٠٤**
 * ══════════════════════════════════════════════════════════════════════
 *
 *	NotesLog          الملاحظاتُ الداخليّةُ سجلٌّ لا يُمحى بكاتبه وتاريخه
 *	CashBanCard       منعُ النقد بسببه وانتهائه، ورفعُه لمديرِ المنصّة بسبب
 *	DriverPanel       المركبة · النقدُ «من أصل السقف» · قفلُ ما بعد الحادث
 *	TempPasswordBadge «لم يدخل بعد · تنتهي بعد …» و«إعادة إرسال»
 *	PendingPhone      طلبُ تغيير رقمٍ ينتظر شخصاً ثانياً
 *	ResetPasswordModal · ChangePhoneModal
 *
 * **وكلُّ زرٍّ بقدرة بابه من جدول المحرّك** (`useCanCall`).
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtMoney, fmtDate, fmtDateTime, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Confirm,
  FormActions,
  FormSection,
  Input,
  Modal,
  Radio,
  Textarea,
  LoadingState,
  IconEdit,
  IconNote,
  IconBalance,
  IconMoto,
  IconWarning,
  IconCheck,
  IconPhone,
  IconLock,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCanCall } from "@/lib/policy";

const m = getMessages(defaultLocale);
const A = m.admin.acc;

/** **غيرُ مسموح** — لا «لا يوجد اتصال». */
export function NotAllowed() {
  return <Alert tone="warning">{A.notAllowed}</Alert>;
}

function hoursLeft(iso: string): number {
  return Math.max(0, Math.ceil((new Date(iso).getTime() - Date.now()) / 3_600_000));
}

// ── الملاحظاتُ الداخليّة ────────────────────────────────────────────

interface Note {
  id: number;
  body: string;
  author: string | null;
  created_at: string;
}

export function NotesLog({ userID }: { userID: string }) {
  const canCall = useCanCall();
  const canAdd = canCall("POST", "/users/{id}/notes");
  const [rows, setRows] = useState<Note[] | null>(null);
  const [body, setBody] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      const r = await api<{ notes: Note[] }>(`/api/v1/admin/users/${userID}/notes`);
      setRows(r?.notes ?? []);
      setError("");
    } catch (err) {
      setRows([]);
      setError(errorText(err));
    }
  }, [userID]);

  useEffect(() => {
    void load();
  }, [load]);

  async function add(e: React.FormEvent) {
    e.preventDefault();
    if (!body.trim()) return;
    setBusy(true);
    try {
      await api(`/api/v1/admin/users/${userID}/notes`, {
        method: "POST",
        body: JSON.stringify({ body: body.trim() }),
      });
      setBody("");
      await load();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <FormSection title={A.notes} icon={<IconNote />}>
      <p className="mb-2 text-xs text-ink-muted">{A.notesHint}</p>
      {error && <Alert className="mb-2">{error}</Alert>}
      {canAdd && (
        <form onSubmit={add} className="mb-3 space-y-2">
          <Textarea
            id="note-new"
            label=""
            rows={2}
            placeholder={A.notesPlaceholder}
            value={body}
            onChange={(e) => setBody(e.target.value)}
          />
          <div className="flex justify-end">
            <Button type="submit" disabled={busy || !body.trim()}>
              {A.notesAdd}
            </Button>
          </div>
        </form>
      )}
      {rows === null ? (
        <LoadingState variant="text" />
      ) : rows.length === 0 ? (
        <p className="py-3 text-center text-sm text-ink-muted">{A.notesEmpty}</p>
      ) : (
        <ul className="space-y-1.5">
          {rows.map((n) => (
            <li key={n.id} className="rounded-control border border-line px-3 py-2 text-sm">
              <p className="whitespace-pre-wrap">{n.body}</p>
              <p className="mt-1 text-xs text-ink-muted">
                {n.author ?? A.notesUnknown} · <span dir="ltr">{fmtDateTime(n.created_at)}</span>
              </p>
            </li>
          ))}
        </ul>
      )}
    </FormSection>
  );
}

// ── منعُ النقد ──────────────────────────────────────────────────────

interface CashBan {
  blocked: boolean;
  failures: number;
  limit: number;
  days: number;
  until: string | null;
  orders: number[];
  lifted_at: string | null;
}

export function CashBanCard({ userID }: { userID: string }) {
  const canCall = useCanCall();
  const canLift = canCall("POST", "/users/{id}/cash-ban/lift");
  const [ban, setBan] = useState<CashBan | null>(null);
  const [error, setError] = useState("");
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      setBan(await api<CashBan>(`/api/v1/admin/users/${userID}/cash-ban`));
      setError("");
    } catch (err) {
      setError(errorText(err));
    }
  }, [userID]);

  useEffect(() => {
    void load();
  }, [load]);

  async function lift(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      await api(`/api/v1/admin/users/${userID}/cash-ban/lift`, {
        method: "POST",
        body: JSON.stringify({ reason: reason.trim() }),
      });
      setOpen(false);
      setReason("");
      await load();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  if (error && !ban) return <Alert className="mb-3">{error}</Alert>;
  if (!ban) return null;
  return (
    <div className={`mb-3 surface p-3 ${ban.blocked ? "border-danger" : ""}`}>
      <div className="flex flex-wrap items-center justify-between gap-2">
        <span className="flex items-center gap-2 text-sm font-bold">
          <IconBalance size={15} className={ban.blocked ? "text-danger" : "text-ink-muted"} />
          {A.cashBan}
          {ban.blocked ? (
            <Badge variant="danger">
              {A.cashBanActive.replace("{until}", ban.until ? fmtDate(ban.until) : "—")}
            </Badge>
          ) : (
            <Badge variant="success">{A.cashBanNone}</Badge>
          )}
        </span>
        {ban.blocked && canLift && (
          <Button variant="secondary" className="!px-2.5" onClick={() => setOpen(true)}>
            {A.cashBanLift}
          </Button>
        )}
      </div>
      {ban.blocked && (
        <p className="mt-1 text-xs text-ink-muted">
          {A.cashBanWhy
            .replace("{n}", fmtNum(ban.failures))
            .replace("{days}", fmtNum(ban.days))
            .replace("{limit}", fmtNum(ban.limit))
            .replace("{orders}", ban.orders.map((o) => `#${o}`).join(" · "))}
        </p>
      )}
      {ban.lifted_at && (
        <p className="mt-1 text-xs text-ink-muted">{A.cashBanLifted.replace("{at}", fmtDateTime(ban.lifted_at))}</p>
      )}
      {error && <Alert className="mt-2">{error}</Alert>}
      {open && (
        <Modal open onClose={() => setOpen(false)} title={A.cashBanLiftTitle}>
          <form onSubmit={lift} className="space-y-3">
            <p className="text-xs leading-relaxed text-ink-muted">{A.cashBanLiftHint}</p>
            <Textarea id="cb-reason" label={A.reason} required rows={3} value={reason} onChange={(e) => setReason(e.target.value)} />
            <FormActions submit onCancel={() => setOpen(false)} busy={busy} saveLabel={A.cashBanLift} />
          </form>
        </Modal>
      )}
    </div>
  );
}

// ── السائق ──────────────────────────────────────────────────────────

export interface DriverInfo {
  id: string;
  on_shift: boolean;
  driver_cash: number;
  cash_limit: number;
  cash_limit_general: number;
  cash_limit_override: number | null;
  vehicle_type: string;
  vehicle_plate: string;
  vehicle_color: string;
  accident_locked: boolean;
  accident_lock_at: string | null;
  accident_cleared_at: string | null;
  accident_cleared_by: string | null;
}

export function DriverPanel({ d, onChanged, endShift }: { d: DriverInfo; onChanged: () => void; endShift?: React.ReactNode }) {
  const canCall = useCanCall();
  const canVehicle = canCall("PATCH", "/users/{id}/vehicle");
  const canCap = canCall("PATCH", "/users/{id}/cash-limit");
  const canOk = canCall("POST", "/users/{id}/driver-ok");
  const [vehOpen, setVehOpen] = useState(false);
  const [capOpen, setCapOpen] = useState(false);
  const [okOpen, setOkOpen] = useState(false);
  const pct = d.cash_limit > 0 ? Math.min(100, Math.round((d.driver_cash / d.cash_limit) * 100)) : 0;

  return (
    <div className="mb-3 grid grid-cols-1 gap-3 lg:grid-cols-3">
      {/* **شريطُ الحال**: الدوامُ وقفلُ ما بعد الحادث. */}
      <div className="surface p-3 lg:col-span-3">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant={d.on_shift ? "success" : "neutral"}>{d.on_shift ? A.onShiftNow : A.offShiftNow}</Badge>
          {d.accident_locked ? (
            <Badge variant="danger">
              <span className="inline-flex items-center gap-1">
                <IconWarning size={12} />
                {A.accidentLocked}
              </span>
            </Badge>
          ) : (
            d.accident_cleared_at && (
              <span className="text-xs text-ink-muted">
                {A.accidentCleared
                  .replace("{by}", d.accident_cleared_by ?? "—")
                  .replace("{at}", fmtDateTime(d.accident_cleared_at))}
              </span>
            )
          )}
          <span className="ms-auto flex flex-wrap gap-1.5">
            {d.accident_locked && canOk && (
              <Button className="!px-2.5" onClick={() => setOkOpen(true)}>
                <span className="flex items-center gap-1.5">
                  <IconCheck size={14} />
                  {A.driverOk}
                </span>
              </Button>
            )}
            {endShift}
          </span>
        </div>
      </div>

      {/* **النقدُ من أصل السقف** — بشريطٍ يُقرأ بلمحة. */}
      <div className="surface p-3">
        <p className="text-xs text-ink-muted">{m.admin.users.profile.driverCash}</p>
        <p className="figure">{fmtMoney(d.driver_cash)}</p>
        <p className="text-xs text-ink-muted">{A.cashOfCap.replace("{cap}", fmtMoney(d.cash_limit))}</p>
        <div className="mt-2 h-1.5 overflow-hidden rounded-badge bg-field">
          <div
            className={`h-full ${pct >= 90 ? "bg-danger-solid" : pct >= 60 ? "bg-warning-fill" : "bg-success-fill"}`}
            style={{ width: `${pct}%` }}
          />
        </div>
        <div className="mt-2 flex flex-wrap items-center justify-between gap-2 text-xs text-ink-muted">
          <span>
            {d.cash_limit_override == null
              ? A.cashCapUseGeneral.replace("{cap}", fmtMoney(d.cash_limit_general))
              : A.cashCapCustom}
          </span>
          {canCap && (
            <button type="button" className="font-medium text-primary hover:underline" onClick={() => setCapOpen(true)}>
              {A.cashCapEdit}
            </button>
          )}
        </div>
      </div>

      {/* **المركبة** — نوعٌ ولوحةٌ ولون، اختياريّةٌ كلُّها. */}
      <div className="surface p-3 lg:col-span-2">
        <div className="mb-1 flex items-center justify-between gap-2">
          <span className="flex items-center gap-1.5 text-sm font-bold">
            <IconMoto size={15} className="text-primary" />
            {A.vehicle}
          </span>
          {canVehicle && (
            <button type="button" className="text-xs font-medium text-primary hover:underline" onClick={() => setVehOpen(true)}>
              <span className="inline-flex items-center gap-1">
                <IconEdit size={12} />
                {A.vehicleEdit}
              </span>
            </button>
          )}
        </div>
        {d.vehicle_type || d.vehicle_plate || d.vehicle_color ? (
          <dl className="grid grid-cols-1 gap-2 text-sm sm:grid-cols-3">
            <div>
              <dt className="text-xs text-ink-muted">{A.vehicleType}</dt>
              <dd>{d.vehicle_type || "—"}</dd>
            </div>
            <div>
              <dt className="text-xs text-ink-muted">{A.vehiclePlate}</dt>
              <dd dir="ltr" className="text-end font-mono">
                {d.vehicle_plate || "—"}
              </dd>
            </div>
            <div>
              <dt className="text-xs text-ink-muted">{A.vehicleColor}</dt>
              <dd>{d.vehicle_color || "—"}</dd>
            </div>
          </dl>
        ) : (
          <p className="text-sm text-ink-muted">{A.vehicleNone}</p>
        )}
      </div>

      {vehOpen && <VehicleModal d={d} onClose={() => setVehOpen(false)} onSaved={onChanged} />}
      {capOpen && <CashCapModal d={d} onClose={() => setCapOpen(false)} onSaved={onChanged} />}
      {okOpen && <DriverOkModal userID={d.id} onClose={() => setOkOpen(false)} onSaved={onChanged} />}
    </div>
  );
}

function VehicleModal({ d, onClose, onSaved }: { d: DriverInfo; onClose: () => void; onSaved: () => void }) {
  const [type, setType] = useState(d.vehicle_type);
  const [plate, setPlate] = useState(d.vehicle_plate);
  const [color, setColor] = useState(d.vehicle_color);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  return (
    <Modal open onClose={onClose} title={A.vehicle}>
      <form
        className="space-y-3"
        onSubmit={async (e) => {
          e.preventDefault();
          setBusy(true);
          try {
            await api(`/api/v1/admin/users/${d.id}/vehicle`, {
              method: "PATCH",
              body: JSON.stringify({ vehicle_type: type, vehicle_plate: plate, vehicle_color: color }),
            });
            onClose();
            onSaved();
          } catch (err) {
            setError(errorText(err));
          } finally {
            setBusy(false);
          }
        }}
      >
        <Input id="veh-type" label={A.vehicleType} value={type} onChange={(e) => setType(e.target.value)} />
        <Input id="veh-plate" label={A.vehiclePlate} dir="ltr" value={plate} onChange={(e) => setPlate(e.target.value)} />
        <Input id="veh-color" label={A.vehicleColor} value={color} onChange={(e) => setColor(e.target.value)} />
        {error && <Alert>{error}</Alert>}
        <FormActions submit onCancel={onClose} busy={busy} />
      </form>
    </Modal>
  );
}

function CashCapModal({ d, onClose, onSaved }: { d: DriverInfo; onClose: () => void; onSaved: () => void }) {
  const [mode, setMode] = useState<"general" | "custom">(d.cash_limit_override == null ? "general" : "custom");
  const [value, setValue] = useState(String(d.cash_limit_override ?? d.cash_limit_general));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  return (
    <Modal open onClose={onClose} title={A.cashCap}>
      <form
        className="space-y-3"
        onSubmit={async (e) => {
          e.preventDefault();
          setBusy(true);
          try {
            await api(`/api/v1/admin/users/${d.id}/cash-limit`, {
              method: "PATCH",
              body: JSON.stringify({ limit: mode === "general" ? null : Number(value) || 0 }),
            });
            onClose();
            onSaved();
          } catch (err) {
            setError(errorText(err));
          } finally {
            setBusy(false);
          }
        }}
      >
        <Radio
          id="cap-general"
          name="cap-mode"
          checked={mode === "general"}
          onChange={() => setMode("general")}
          label={A.cashCapUseGeneral.replace("{cap}", fmtMoney(d.cash_limit_general))}
        />
        <Radio id="cap-custom" name="cap-mode" checked={mode === "custom"} onChange={() => setMode("custom")} label={A.cashCapUseCustom} />
        {mode === "custom" && (
          <Input
            id="cap-value"
            label={`${A.cashCapCustom} (${m.common.currency})`}
            type="number"
            min="0"
            required
            value={value}
            onChange={(e) => setValue(e.target.value)}
          />
        )}
        {error && <Alert>{error}</Alert>}
        <FormActions submit onCancel={onClose} busy={busy} />
      </form>
    </Modal>
  );
}

function DriverOkModal({ userID, onClose, onSaved }: { userID: string; onClose: () => void; onSaved: () => void }) {
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  return (
    <Modal open onClose={onClose} title={A.driverOkTitle}>
      <form
        className="space-y-3"
        onSubmit={async (e) => {
          e.preventDefault();
          setBusy(true);
          try {
            await api(`/api/v1/admin/users/${userID}/driver-ok`, {
              method: "POST",
              body: JSON.stringify({ note: note.trim() }),
            });
            onClose();
            onSaved();
          } catch (err) {
            setError(errorText(err));
          } finally {
            setBusy(false);
          }
        }}
      >
        <p className="text-sm text-ink-muted">{A.driverOkHint}</p>
        <Input id="ok-note" label={A.driverOkNote} value={note} onChange={(e) => setNote(e.target.value)} />
        {error && <Alert>{error}</Alert>}
        <FormActions submit onCancel={onClose} busy={busy} saveLabel={A.driverOk} />
      </form>
    </Modal>
  );
}

// ── كلمةُ الدخول المؤقّتة ─────────────────────────────────────────────

export function TempPasswordBadge({
  userID,
  expiresAt,
  canResend,
  onDone,
}: {
  userID: string;
  expiresAt: string;
  canResend: boolean;
  onDone: () => void;
}) {
  const [confirm, setConfirm] = useState(false);
  const [busy, setBusy] = useState(false);
  const [msg, setMsg] = useState<{ tone: "success" | "warning" | "danger"; text: string } | null>(null);
  const left = hoursLeft(expiresAt);

  async function resend() {
    setBusy(true);
    try {
      const r = await api<{ sent: boolean; expires_at: string }>(`/api/v1/admin/users/${userID}/resend-welcome`, {
        method: "POST",
        body: "{}",
      });
      setMsg(
        r.sent
          ? { tone: "success", text: A.resendDone.replace("{h}", fmtNum(hoursLeft(r.expires_at))) }
          : { tone: "warning", text: A.resendNotSent },
      );
      onDone();
    } catch (err) {
      setMsg({ tone: "danger", text: errorText(err) });
    } finally {
      setBusy(false);
      setConfirm(false);
    }
  }

  return (
    <span className="inline-flex flex-wrap items-center gap-1.5">
      <Badge variant={left > 0 ? "warning" : "danger"}>
        <span className="inline-flex items-center gap-1">
          <IconLock size={11} />
          {left > 0 ? A.tempPending.replace("{h}", fmtNum(left)) : A.tempExpired}
        </span>
      </Badge>
      {canResend && (
        <button type="button" className="text-xs font-medium text-primary hover:underline" onClick={() => setConfirm(true)}>
          {A.resend}
        </button>
      )}
      {msg && <span className={`text-xs ${msg.tone === "success" ? "text-success" : "text-danger"}`}>{msg.text}</span>}
      <Confirm
        open={confirm}
        tone="primary"
        title={A.resendTitle}
        body={A.resendBody}
        confirmLabel={A.resend}
        busy={busy}
        onConfirm={() => void resend()}
        onCancel={() => setConfirm(false)}
      />
    </span>
  );
}

// ── تغييرُ الرقم ─────────────────────────────────────────────────────

export interface PendingPhoneChange {
  id: string;
  new_phone: string;
  proposed_by: string;
  proposer_name: string;
}

export function PendingPhoneBanner({ req, onDone }: { req: PendingPhoneChange; onDone: () => void }) {
  const { user: me } = useAuth();
  const canCall = useCanCall();
  const canDecide = canCall("POST", "/phone-requests/{id}/approve");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function decide(ok: boolean) {
    setBusy(true);
    try {
      await api(`/api/v1/admin/phone-requests/${req.id}/${ok ? "approve" : "reject"}`, { method: "POST", body: "{}" });
      onDone();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }
  return (
    <Alert tone="warning" className="mb-3">
      <span className="flex flex-wrap items-center gap-2">
        <IconPhone size={14} />
        <span>
          {A.phonePendingBanner.replace("{phone}", req.new_phone).replace("{by}", req.proposer_name)}
        </span>
        {canDecide && (
          <span className="ms-auto flex gap-1.5">
            {me?.id !== req.proposed_by && (
              <Button className="!px-2.5" disabled={busy} onClick={() => void decide(true)}>
                {A.approve}
              </Button>
            )}
            <Button variant="secondary" className="!px-2.5" disabled={busy} onClick={() => void decide(false)}>
              {A.reject}
            </Button>
          </span>
        )}
      </span>
      {error && <span className="mt-1 block text-xs">{error}</span>}
    </Alert>
  );
}

export function ChangePhoneModal({
  userID,
  current,
  onClose,
  onDone,
}: {
  userID: string;
  current: string;
  onClose: () => void;
  onDone: (msg: string) => void;
}) {
  const [phone, setPhone] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      // **ولا ترويسةَ هنا** — فيسأل العميلُ كلمةَ سرِّ الموظّف تلقائيّاً (خطوةُ التحقّق).
      const r = await api<{ phone_change?: { pending: boolean } }>(`/api/v1/admin/users/${userID}`, {
        method: "PATCH",
        body: JSON.stringify({ phone }),
      });
      onDone(r?.phone_change?.pending ? A.phonePendingSent : A.phoneChanged);
    } catch (err) {
      setError(errorText(err));
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={A.phoneTitle}>
      <form onSubmit={submit} className="space-y-4">
        <p className="text-sm text-ink-muted" dir="ltr">
          {current}
        </p>
        <Input
          id="new-phone"
          label={m.admin.users.profile.newPhone}
          icon={<IconPhone />}
          dir="ltr"
          required
          autoFocus
          value={phone}
          onChange={(e) => setPhone(e.target.value)}
          className="text-end"
          placeholder="09xxxxxxxx"
        />
        <p className="rounded-control bg-field px-3 py-2 text-xs leading-relaxed text-ink-muted">{A.phoneBody}</p>
        {error && <Alert>{error}</Alert>}
        <FormActions submit onCancel={onClose} busy={busy} />
      </form>
    </Modal>
  );
}

// ── إعادةُ كلمة المرور ───────────────────────────────────────────────

export function ResetPasswordModal({
  userID,
  hours,
  onClose,
  onDone,
}: {
  userID: string;
  /** **مهلةُ الكلمة المؤقّتة** — من الإعدادات عبر المحرّك. */
  hours: number;
  onClose: () => void;
  onDone: () => void;
}) {
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<{ sent: boolean; expires_at: string } | null>(null);
  const [error, setError] = useState("");

  async function go() {
    setBusy(true);
    setError("");
    try {
      setResult(
        await api<{ sent: boolean; expires_at: string }>(`/api/v1/admin/users/${userID}/password`, {
          method: "POST",
          body: "{}",
        }),
      );
      onDone();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={A.resetTitle}>
      {result ? (
        <div className="space-y-4">
          <Alert tone={result.sent ? "success" : "warning"}>
            {result.sent ? A.resendDone.replace("{h}", fmtNum(hoursLeft(result.expires_at))) : A.resendNotSent}
          </Alert>
          <FormActions onSave={onClose} saveLabel={m.common.confirm} />
        </div>
      ) : (
        <div className="space-y-4">
          <p className="text-sm leading-relaxed text-ink-muted">{A.resetBody.replace("{h}", fmtNum(hours))}</p>
          {error && <Alert>{error}</Alert>}
          <FormActions onSave={() => void go()} onCancel={onClose} busy={busy} saveLabel={A.resetConfirm} />
        </div>
      )}
    </Modal>
  );
}
