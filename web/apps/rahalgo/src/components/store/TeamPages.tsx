"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **موظّفو المتجر ورقمُ الطلبات حسب الدوام — صفحتان** (مرآةُ `TeamScreens.kt`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **بالشكل الجديد لتطبيق المتجر** (٢٠٢٦-١٠-٢٩): رأسٌ بسهم رجوع، بطاقاتٌ
 * بيضاءُ مدوّرة، بطاقتا اختيارٍ متساويتان بعلامة صحّ، **وبلا أسطرِ شرح** —
 * عناوينُ وحقولٌ وأزرارٌ وأخطاءُ الحقول وحدَها.
 *
 * **وفي «دوام كامل» لا خانةَ رقم** — المحرّكُ يحوّل الطلبَ إلى رقم المتجر الأساسيّ.
 *
 * **والموظّفُ لا يرى بابَ الموظّفين** (`staff_not_allowed`)، **والمتجرُ الموقوفُ
 * يقرأ ولا يعدّل.** (وبطاقةُ الإدارة القديمة باقيةٌ في `StoreStaffAdmin.tsx`.)
 */

import Link from "next/link";
import { useCallback, useEffect, useState, type ReactNode } from "react";
import { getMessages, defaultLocale, errorText, fmtDateTime } from "@rahalgo/i18n";
import {
  Alert,
  Button,
  Confirm,
  IconAdd,
  IconCheck,
  IconDelete,
  IconHourglass,
  IconPhone,
  IconPrev,
  IconUser,
  Input,
  LoadingState,
  ReloadState,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { fill } from "@/lib/customer/texts";
import { useStore } from "@/lib/store/context";

const m = getMessages(defaultLocale);
const T = m.store.team;

// ══════════════════════════════════════════════════════════════════════
// **قِطَعُ الصفحتين**
// ══════════════════════════════════════════════════════════════════════

/** **رأسُ الصفحة** — سهمُ رجوعٍ إلى «متجري» وعنوان (`PageHeader` في التطبيق). */
function TeamHeader({ title }: { title: string }) {
  return (
    <div className="flex items-center gap-1">
      <Link
        href="/store"
        aria-label={T.back}
        className="flex h-12 w-12 shrink-0 items-center justify-center rounded-full text-ink hover:bg-row-hover"
      >
        <IconPrev size={24} />
      </Link>
      <h2 className="heading-page">{title}</h2>
    </div>
  );
}

/** **بطاقةٌ بيضاءُ مدوّرةٌ بحدٍّ رفيع** — وعنوانٌ باهتٌ فوقها إن وُجد. */
function WhiteCard({ title, children }: { title?: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-2">
      {title && <h3 className="px-1 text-sm font-bold text-ink-muted">{title}</h3>}
      <div className="surface flex flex-col gap-3 rounded-card border border-line p-4">{children}</div>
    </section>
  );
}

/** **أيصلح الرقم؟** — فحصٌ أوّليٌّ (٩ إلى ١٥ رقماً)، **والفصلُ للمحرّك.** */
function phoneLooksValid(s: string): boolean {
  const n = s.replace(/\D/g, "").length;
  return n >= 9 && n <= 15;
}

function phoneError(value: string, tried: boolean): string | undefined {
  if (!tried) return undefined;
  if (!value.trim()) return T.phoneNeeded;
  if (!phoneLooksValid(value)) return T.phoneBad;
  return undefined;
}

/** **بطاقتا اختيارٍ متساويتان** — أيقونةٌ في دائرة وعنوان، والمختارةُ بحدّ العلامة وعلامةِ صحّ. */
function IconChoiceCard({
  icon,
  title,
  selected,
  disabled,
  onClick,
}: {
  icon: ReactNode;
  title: string;
  selected: boolean;
  disabled: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      role="radio"
      aria-checked={selected}
      disabled={disabled}
      onClick={onClick}
      className={`relative flex min-h-[10.5rem] flex-1 flex-col items-center justify-center gap-3 rounded-card p-4 text-center transition-colors disabled:cursor-default ${
        selected ? "border-2 border-accent bg-accent-tint" : "border border-line bg-surface hover:bg-row-hover"
      }`}
    >
      {selected && (
        <span className="absolute end-3 top-3 flex h-6 w-6 items-center justify-center rounded-full bg-accent text-on-bright">
          <IconCheck size={14} />
        </span>
      )}
      <span
        className={`flex h-14 w-14 items-center justify-center rounded-full ${
          selected ? "bg-accent text-on-bright" : "bg-accent-tint text-accent-text"
        }`}
      >
        {icon}
      </span>
      <span className="text-base font-bold text-ink">{title}</span>
    </button>
  );
}

// ══════════════════════════════════════════════════════════════════════
// **١ · موظّفو المتجر**
// ══════════════════════════════════════════════════════════════════════

interface StaffRow {
  id: string;
  phone: string;
  name: string;
  last_login_at: string | null;
}

const BASE = "/api/v1/merchant/staff";

export function StaffPage() {
  const { store } = useStore();
  const canManage = store?.status !== "suspended";
  const [list, setList] = useState<StaffRow[] | null>(null);
  const [loadError, setLoadError] = useState("");
  const [isStaff, setIsStaff] = useState("");
  const [phone, setPhone] = useState("");
  const [name, setName] = useState("");
  const [tried, setTried] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [added, setAdded] = useState<{ password: string; sent: boolean } | null>(null);
  const [removing, setRemoving] = useState<StaffRow | null>(null);

  const load = useCallback(() => {
    setLoadError("");
    api<{ staff: StaffRow[] }>(BASE)
      .then((r) => setList(r.staff ?? []))
      .catch((e) => {
        if (e instanceof ApiError && e.body.code === "staff_not_allowed") setIsStaff(errorText(e));
        else setLoadError(errorText(e));
      });
  }, []);
  useEffect(load, [load]);

  async function add() {
    setTried(true);
    if (busy || !phoneLooksValid(phone)) return;
    setTried(false);
    setBusy(true);
    setError("");
    setAdded(null);
    try {
      const r = await api<{ password: string; sent: boolean }>(BASE, {
        method: "POST",
        body: JSON.stringify({ phone: phone.trim(), name: name.trim() }),
      });
      setAdded(r);
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
      await api(`${BASE}/${row.id}`, { method: "DELETE" });
      setRemoving(null);
      load();
    } catch (e) {
      setError(errorText(e));
      setRemoving(null);
    } finally {
      setBusy(false);
    }
  }

  let body: ReactNode;
  if (isStaff) body = <Alert>{isStaff}</Alert>;
  else if (list == null)
    body = loadError ? <ReloadState label={loadError} onRetry={load} /> : <LoadingState variant="list" rows={3} />;
  else
    body = (
      <>
        {list.length === 0 ? (
          <div className="surface flex flex-col items-center gap-3 rounded-card border border-line px-4 py-8">
            <span className="flex h-14 w-14 items-center justify-center rounded-full bg-accent-tint text-accent-text">
              <IconUser size={28} />
            </span>
            <p className="font-bold text-ink-muted">{T.none}</p>
          </div>
        ) : (
          <section className="flex flex-col gap-2">
            <h3 className="px-1 text-sm font-bold text-ink-muted">{T.listTitle}</h3>
            <ul className="surface divide-y divide-line-soft rounded-card border border-line">
              {list.map((row) => {
                const initial = row.name.trim().charAt(0) || row.phone.replace(/\D/g, "").slice(-1) || "?";
                return (
                  <li key={row.id} className="flex min-h-[4.5rem] items-center gap-3 px-3 py-2">
                    <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-accent-tint text-base font-bold text-accent-text">
                      {initial}
                    </span>
                    <div className="min-w-0 flex-1">
                      <p className="truncate font-bold">{row.name || <bdi dir="ltr">{row.phone}</bdi>}</p>
                      {row.name && (
                        <p className="text-sm text-ink-muted">
                          <bdi dir="ltr">{row.phone}</bdi>
                        </p>
                      )}
                      <p className="text-xs text-ink-muted">
                        {row.last_login_at ? fill(T.lastLogin, { at: fmtDateTime(row.last_login_at) }) : T.neverLogin}
                      </p>
                    </div>
                    {canManage && (
                      <button
                        type="button"
                        aria-label={T.remove}
                        title={T.remove}
                        disabled={busy}
                        onClick={() => setRemoving(row)}
                        className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-danger-tint text-danger transition-opacity hover:opacity-80 disabled:opacity-60"
                      >
                        <IconDelete size={20} />
                      </button>
                    )}
                  </li>
                );
              })}
            </ul>
          </section>
        )}

        {canManage && (
          <WhiteCard title={T.newTitle}>
            <Input
              id="staff-phone"
              label={T.phone}
              icon={<IconPhone size={18} />}
              type="tel"
              inputMode="tel"
              dir="ltr"
              autoComplete="off"
              value={phone}
              disabled={busy}
              error={phoneError(phone, tried)}
              onChange={(e) => setPhone(e.target.value.replace(/[^\d+]/g, "").slice(0, 16))}
            />
            <Input
              id="staff-name"
              label={T.name}
              icon={<IconUser size={18} />}
              autoComplete="off"
              value={name}
              disabled={busy}
              onChange={(e) => setName(e.target.value.slice(0, 60))}
            />
            <Button size="lg" className="flex w-full items-center justify-center gap-2" disabled={busy} onClick={() => void add()}>
              <IconAdd size={20} />
              {T.add}
            </Button>
          </WhiteCard>
        )}

        {added && (
          <Alert tone="success">
            {fill(T.added, { password: added.password })} — {added.sent ? T.sent : T.notSent}
          </Alert>
        )}
        {error && <Alert>{error}</Alert>}
      </>
    );

  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-4">
      <TeamHeader title={T.staffTitle} />
      {body}
      {removing && (
        <Confirm
          open
          title={fill(T.removeConfirm, { name: removing.name || removing.phone })}
          confirmLabel={T.remove}
          tone="danger"
          busy={busy}
          onConfirm={() => void remove(removing)}
          onCancel={() => setRemoving(null)}
        />
      )}
    </div>
  );
}

// ══════════════════════════════════════════════════════════════════════
// **٢ · رقمُ الطلبات حسب الدوام**
// ══════════════════════════════════════════════════════════════════════

interface Phones {
  mode: "full" | "split";
  morning_phone: string;
  evening_phone: string;
  evening_from: string;
}

export function OrderPhonesPage() {
  const { store } = useStore();
  const id = store?.id ?? "";
  const canManage = store?.status !== "suspended";
  const url = `/api/v1/merchant/stores/${id}/order-phones`;
  const [p, setP] = useState<Phones | null>(null);
  const [loadError, setLoadError] = useState("");
  const [tried, setTried] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [saved, setSaved] = useState(false);

  const load = useCallback(() => {
    if (!id) return;
    setLoadError("");
    api<Phones>(url)
      .then(setP)
      .catch((e) => setLoadError(errorText(e)));
  }, [id, url]);
  useEffect(load, [load]);

  function edit(next: Phones) {
    setP(next);
    setSaved(false);
  }

  async function save() {
    if (!p || busy) return;
    setTried(true);
    if (p.mode === "split" && !(phoneLooksValid(p.morning_phone) && phoneLooksValid(p.evening_phone))) return;
    setBusy(true);
    setError("");
    setSaved(false);
    try {
      setP(await api<Phones>(url, { method: "PUT", body: JSON.stringify(p) }));
      setSaved(true);
      setTried(false);
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  const editable = canManage && !busy;

  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-4">
      <TeamHeader title={T.phonesTitle} />
      {p == null ? (
        loadError ? <ReloadState label={loadError} onRetry={load} /> : <LoadingState variant="list" rows={2} />
      ) : (
        <>
          <div className="flex gap-3" role="radiogroup" aria-label={T.phonesTitle}>
            <IconChoiceCard
              icon={<IconPhone size={28} />}
              title={T.full}
              selected={p.mode === "full"}
              disabled={!editable}
              onClick={() => {
                edit({ ...p, mode: "full" });
                setTried(false);
              }}
            />
            <IconChoiceCard
              icon={<IconHourglass size={28} />}
              title={T.split}
              selected={p.mode === "split"}
              disabled={!editable}
              onClick={() => edit({ ...p, mode: "split" })}
            />
          </div>

          {p.mode === "split" && (
            <WhiteCard title={T.numbersTitle}>
              <Input
                id="op-morning"
                label={T.morning}
                icon={<IconPhone size={18} />}
                type="tel"
                inputMode="tel"
                dir="ltr"
                value={p.morning_phone}
                disabled={!editable}
                error={phoneError(p.morning_phone, tried)}
                onChange={(e) => edit({ ...p, morning_phone: e.target.value.replace(/[^\d+]/g, "").slice(0, 16) })}
              />
              <Input
                id="op-evening"
                label={T.evening}
                icon={<IconPhone size={18} />}
                type="tel"
                inputMode="tel"
                dir="ltr"
                value={p.evening_phone}
                disabled={!editable}
                error={phoneError(p.evening_phone, tried)}
                onChange={(e) => edit({ ...p, evening_phone: e.target.value.replace(/[^\d+]/g, "").slice(0, 16) })}
              />
              <div className="flex items-center gap-2 border-t border-line-soft pt-3">
                <IconHourglass size={20} className="shrink-0 text-accent-text" />
                <label htmlFor="op-from" className="min-w-0 flex-1 text-sm">
                  {T.eveningFrom}
                </label>
                {/* **رقاقةُ الساعة** — بلون العلامة كرقاقات مخطّط الدوام، والقيمةُ `HH:MM`. */}
                <input
                  id="op-from"
                  type="time"
                  dir="ltr"
                  value={(p.evening_from || "16:00").slice(0, 5)}
                  disabled={!editable}
                  onChange={(e) => edit({ ...p, evening_from: e.target.value })}
                  className="rounded-full border border-accent-edge bg-accent-tint px-4 py-2 font-bold text-accent-text disabled:text-ink-muted"
                />
              </div>
            </WhiteCard>
          )}

          {canManage && (
            <Button size="lg" className="flex w-full items-center justify-center gap-2" disabled={busy} onClick={() => void save()}>
              <IconCheck size={20} />
              {T.save}
            </Button>
          )}
          {saved && <Alert tone="success">{T.saved}</Alert>}
          {error && <Alert>{error}</Alert>}
        </>
      )}
    </div>
  );
}
