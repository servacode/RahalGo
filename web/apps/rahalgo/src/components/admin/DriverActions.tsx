"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أفعالُ السائق — صندوقُه وتسويتُه وإنهاءُ ورديّته**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (نُقلت من شاشة السائقين حين حُذفت ٢٠٢٦-٠٨-١٥ — **بنوافذها ونداءاتها
 *  كما هي**: النقلُ لا يُضيّع شيئاً، **وحذفُ ملفٍّ وإعادةُ كتابته من
 *  الذاكرة هو ما يُضيّع.**)
 *
 * # ولماذا في بطاقة الحساب لا في ملفّه
 *
 * **«إنهاءُ الورديّة» و«التسوية» يُفعلان على عجل**: سائقٌ نسي ورديّتَه
 * بعد منتصف الليل، أو بيده مالٌ ينتظر. **ومن فتح ملفَّه ليضغط زرّاً
 * واحداً دفع ثمنَ صفحةٍ كاملة.**
 *
 * # ولا تعرف هذه النوافذُ من السائق إلّا معرّفَه واسمَه
 *
 * **فتُنادى من أيّ شاشة** — ولو حملت نوعَ صفِّ السائق لَما استُعملت
 * إلّا حيث ذاك الصفّ.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtTime, fmtMoney } from "@rahalgo/i18n";
import {
  Alert,
  Button,
  Input,
  Badge,
  Modal,
  Money,
  IconWallet,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);

/** **ومن تُفعل به** — معرّفٌ واسمٌ لا صفُّ جدول. */
export interface DriverRef {
  id: string;
  full_name: string;
  /** **ويُعرض حين لا اسمَ له** — حسابٌ بلا اسمٍ يُعرف برقمه. */
  phone?: string;
}

interface CashEntry {
  id: string;
  kind: string;
  amount: number;
  note: string;
  created_at: string;
}

/** **ورسالةُ الخادم تُترجَم بمفتاحها** — نُقلت مع النوافذ كما هي. */
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

/**
 * إغلاقُ الدوام — **بكلمةٍ تصل صاحبَه.**
 *
 * **ومن أُغلق دوامُه بلا علمه يظنّ أنّ النظام أعطبه** — فيشتكي، أو يظنّ أن لا
 * طلباتِ اليوم فيمضي إلى بيته. **والكلمةُ ليست تجميلاً: هي الفرقُ بين إجراءٍ
 * وبين عطبٍ يبدو عشوائياً.**
 */
export function EndShiftModal({
  driver,
  onClose,
  onDone,
}: {
  driver: DriverRef;
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

export function CashBoxModal({
  driver,
  canSettle,
  onClose,
  onChanged,
}: {
  driver: DriverRef;
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
            {held === null ? "…" : fmtMoney(held)}
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
