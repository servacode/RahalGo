"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **موظّفو المتجر ورقمُ الطلبات بحسب الدوام** (قرارُ المالك ٢٠٢٦-١٠-٢٠)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **الموظّفُ يدخل تطبيقَ المتجر برقمه ويعمل باسم المتجر** — بلا أن يُخرج صاحبَه
 * (`identity/staff.go`). **ورقمُ الطلبات** إمّا واحدٌ أو صباحيٌّ ومسائيّ، ويذهب إليه
 * تحويلُ الطلب على واتساب بحسب الساعة.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtDateTime } from "@rahalgo/i18n";
import { Alert, Button, Card, Confirm, Input } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { fill } from "@/lib/customer/texts";

const m = getMessages(defaultLocale);
const S = m.admin.storeStaff;
const P = m.admin.orderPhones;

interface StaffRow {
  id: string;
  phone: string;
  name: string;
  last_login_at: string | null;
}

export function StoreStaffAdmin({ merchantId, canManage }: { merchantId: string; canManage: boolean }) {
  return <StoreStaffCard base={`/api/v1/admin/merchants/${merchantId}/staff`} canManage={canManage} />;
}

/** **البطاقةُ نفسُها للإدارة ولصاحب المتجر** — والفرقُ بابُ الشبكة وحدَه. */
export function StoreStaffCard({ base, canManage }: { base: string; canManage: boolean }) {
  const [list, setList] = useState<StaffRow[] | null>(null);
  const [phone, setPhone] = useState("");
  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [note, setNote] = useState("");
  const [removing, setRemoving] = useState<StaffRow | null>(null);

  const load = useCallback(() => {
    api<{ staff: StaffRow[] }>(base)
      .then((r) => setList(r.staff ?? []))
      .catch((e) => setError(errorText(e)));
  }, [base]);
  useEffect(load, [load]);

  async function add() {
    setBusy(true);
    setError("");
    setNote("");
    try {
      const r = await api<{ password: string; sent: boolean }>(base, {
        method: "POST",
        body: JSON.stringify({ phone, name }),
      });
      setNote(`${fill(S.added, { password: r.password })} — ${r.sent ? S.sent : S.notSent}`);
      setPhone("");
      setName("");
      load();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  async function remove(row: StaffRow) {
    setBusy(true);
    setError("");
    try {
      await api(`${base}/${row.id}`, { method: "DELETE" });
      setRemoving(null);
      load();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Card title={S.title}>
      <div className="flex flex-col gap-3">
        <p className="text-xs text-ink-muted">{S.hint}</p>
        {list && list.length === 0 && <p className="text-sm text-ink-muted">{S.none}</p>}
        {list?.map((row) => (
          <div key={row.id} className="flex flex-wrap items-center gap-2 border-b border-line-soft py-2 text-sm">
            <div className="min-w-0 flex-1">
              <p className="font-bold">{row.name || row.phone}</p>
              <p className="text-xs text-ink-muted" dir="ltr">
                {row.phone}
              </p>
              <p className="text-xs text-ink-muted">
                {row.last_login_at ? fill(S.lastLogin, { at: fmtDateTime(row.last_login_at) }) : S.neverLogin}
              </p>
            </div>
            {canManage && (
              <Button variant="secondary" onClick={() => setRemoving(row)} disabled={busy}>
                {S.remove}
              </Button>
            )}
          </div>
        ))}
        {canManage && (
          <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
            <Input id="staff-phone" label={S.phone} dir="ltr" value={phone} onChange={(e) => setPhone(e.target.value)} />
            <Input id="staff-name" label={S.name} value={name} onChange={(e) => setName(e.target.value)} />
            <Button onClick={() => void add()} disabled={busy || !phone.trim()} className="w-fit">
              {S.add}
            </Button>
          </div>
        )}
        {note && <p className="text-sm font-bold text-success">{note}</p>}
        {error && <Alert>{error}</Alert>}
      </div>
      {removing && (
        <Confirm
          open
          title={fill(S.removeConfirm, { name: removing.name || removing.phone })}
          confirmLabel={S.remove}
          tone="danger"
          busy={busy}
          onConfirm={() => void remove(removing)}
          onCancel={() => setRemoving(null)}
        />
      )}
    </Card>
  );
}

interface Phones {
  mode: "full" | "split";
  morning_phone: string;
  evening_phone: string;
  evening_from: string;
}

export function OrderPhonesAdmin({ merchantId, canManage }: { merchantId: string; canManage: boolean }) {
  return <OrderPhonesCard url={`/api/v1/admin/merchants/${merchantId}/order-phones`} canManage={canManage} />;
}

export function OrderPhonesCard({ url, canManage }: { url: string; canManage: boolean }) {
  const [p, setP] = useState<Phones | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [note, setNote] = useState("");

  useEffect(() => {
    api<Phones>(url)
      .then(setP)
      .catch((e) => setError(errorText(e)));
  }, [url]);

  async function save() {
    if (!p) return;
    setBusy(true);
    setError("");
    setNote("");
    try {
      setP(await api<Phones>(url, { method: "PUT", body: JSON.stringify(p) }));
      setNote(P.saved);
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  if (!p) return error ? <Alert>{error}</Alert> : null;
  return (
    <Card title={P.title}>
      <div className="flex flex-col gap-3">
        <p className="text-xs text-ink-muted">{P.hint}</p>
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          {(["full", "split"] as const).map((mode) => (
            <button
              key={mode}
              type="button"
              aria-pressed={p.mode === mode}
              disabled={!canManage}
              onClick={() => setP({ ...p, mode })}
              className={`rounded-control border px-3 py-3 text-sm font-bold ${
                p.mode === mode ? "border-2 border-accent bg-accent-tint" : "border-line bg-field hover:bg-row-hover"
              }`}
            >
              {mode === "full" ? P.full : P.split}
            </button>
          ))}
        </div>
        {p.mode === "split" && (
          <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
            <Input id="op-morning" label={P.morning} dir="ltr" value={p.morning_phone} disabled={!canManage}
              onChange={(e) => setP({ ...p, morning_phone: e.target.value })} />
            <Input id="op-evening" label={P.evening} dir="ltr" value={p.evening_phone} disabled={!canManage}
              onChange={(e) => setP({ ...p, evening_phone: e.target.value })} />
            <Input id="op-from" label={P.eveningFrom} type="time" dir="ltr" value={p.evening_from} disabled={!canManage}
              onChange={(e) => setP({ ...p, evening_from: e.target.value })} />
          </div>
        )}
        {canManage && (
          <Button onClick={() => void save()} disabled={busy} className="w-fit">
            {P.save}
          </Button>
        )}
        {note && <p className="text-sm text-success">{note}</p>}
        {error && <Alert>{error}</Alert>}
      </div>
    </Card>
  );
}

/**
 * **«حذف المتجر نهائياً» — لمتجرٍ بلا طلب** (قرارُ المالك ٢٠٢٦-١٠-٢٠) — والخادمُ يردّ
 * متجراً له طلبٌ أو تسوية (`merchant_has_orders`)، فالزرُّ لا يكذب ولو ظهر.
 */
export function DeleteMerchantButton({
  merchantId,
  name,
  onDeleted,
}: {
  merchantId: string;
  name: string;
  onDeleted: () => void;
}) {
  const D = m.admin.merchantDelete;
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function run() {
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/merchants/${merchantId}`, { method: "DELETE" });
      setOpen(false);
      onDeleted();
    } catch (e) {
      setError(errorText(e));
      setOpen(false);
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="flex flex-col gap-2">
      <p className="text-xs text-ink-muted">{D.hint}</p>
      <Button variant="danger" onClick={() => setOpen(true)} disabled={busy} className="w-fit">
        {D.button}
      </Button>
      {error && <Alert>{error}</Alert>}
      <Confirm
        open={open}
        title={fill(D.confirm, { name })}
        confirmLabel={D.button}
        tone="danger"
        busy={busy}
        onConfirm={() => void run()}
        onCancel={() => setOpen(false)}
      />
    </div>
  );
}
