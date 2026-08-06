"use client";

/**
 * **سجلُّ مخالفات المتجر — أيُّ طلباتٍ هي.**
 *
 * # الثغرة
 *
 * كانت العملياتُ ترى «٤ مخالفات» على متجر **ولا تملك أن ترى أيَّ طلباتٍ هي**،
 * ثمّ تُقرّر الحظرَ أو العفوَ على رقمٍ مجرّد. **وقرارٌ يُبنى على عددٍ بلا وقائعَ
 * قرارٌ لا يُراجَع** — ولا يُحتجّ به أمام متجرٍ يسأل «متى؟».
 *
 * وهي الثغرةُ `G-01` المسجَّلةُ في `CYCLE-PLATFORM-MANAGED.md` بلفظها: «فتراه
 * العملياتُ ٤ مخالفات **ولا تملك أن ترى أيَّ طلباتٍ هي**».
 *
 * # والإنذارُ يُصدَر من هنا
 *
 * **ولا كلَّ إساءةٍ طلبٌ فاشل**: متجرٌ رفع أسعارَه عن المتّفق، أو أساء إلى
 * سائق، أو تكرّر تأخيرُه بلا فشلٍ مسجَّل. `POST /merchants/{id}/warnings`
 * **مبنيّةٌ منذ البداية ولا زرَّ لها** — **ومن لا مكانَ لإنذاره لا يُنذَر إلّا
 * بمكالمةٍ لا أثر لها**، فلا تُعدّ ولا يُحتجّ بها يوم الحظر.
 *
 * **وموضعُه هنا لا في مكانٍ آخر**: من يقرأ سجلَّ المخالفات هو من يقرّر أن
 * يُنذر.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtRef, fmtDateTime } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Input,
  Modal,
  EmptyState,
  IconWarning,
  IconOrder,
  IconAdd,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);
const V = m.admin.merchants.violationsLog;
const STATUS_LABELS: Record<string, string> = m.orders.status;
const FAIL_REASONS: Record<string, string> = m.common.failReasons;

interface Row {
  order_number: number | null;
  status: string;
  reason: string;
  note: string;
  manual: boolean;
  closed_at: string;
}

function errText(err: unknown): string {
  if (!(err instanceof ApiError)) return m.errors.internal;
  const key = err.body.message_key.split(".").pop() ?? "";
  return (m.errors as Record<string, string>)[key] ?? m.errors.internal;
}

export default function ViolationsModal({
  merchant,
  onClose,
  onChanged,
}: {
  merchant: { id: string; name: string };
  onClose: () => void;
  onChanged: () => Promise<void> | void;
}) {
  const [rows, setRows] = useState<Row[] | null>(null);
  const [count, setCount] = useState(0);
  const [limit, setLimit] = useState(0);
  const [issuing, setIssuing] = useState(false);
  const [reason, setReason] = useState("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      const res = await api<{ violations: number; limit: number; items: Row[] | null }>(
        `/api/v1/admin/merchants/${merchant.id}/violations`,
      );
      setCount(res.violations);
      setLimit(res.limit);
      setRows(res.items ?? []);
    } catch (err) {
      setError(errText(err));
      setRows([]);
    }
  }, [merchant.id]);

  useEffect(() => {
    void load();
  }, [load]);

  async function issue() {
    if (!reason.trim()) return;
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/merchants/${merchant.id}/warnings`, {
        method: "POST",
        body: JSON.stringify({ reason: reason.trim(), note: note.trim() }),
      });
      setReason("");
      setNote("");
      setIssuing(false);
      await load();
      await onChanged();
    } catch (err) {
      setError(errText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={`${V.title}: ${merchant.name}`}>
      {/* **الرقمُ من الحدّ** — «٤ من ٥» تقول ما لا يقوله «٤». */}
      <p
        className={`mb-3 rounded-control px-3 py-2 text-sm ${
          limit > 0 && count >= limit
            ? "bg-danger-tint font-bold text-danger"
            : "bg-page text-ink-muted"
        }`}
      >
        <span dir="ltr" className="font-bold tabular-nums">
          {fmtNum(count)} / {fmtNum(limit)}
        </span>{" "}
        · {V.ofLimit}
      </p>

      {!rows ? (
        <p className="py-6 text-center text-ink-muted">{m.common.loading}</p>
      ) : rows.length === 0 ? (
        <EmptyState icon={IconWarning} title={V.empty} />
      ) : (
        <ul className="max-h-72 divide-y divide-line overflow-y-auto">
          {rows.map((r, i) => (
            <li key={i} className="flex items-start gap-3 py-2 text-sm">
              <span className="min-w-0 flex-1">
                <span className="flex flex-wrap items-center gap-2">
                  {r.order_number != null ? (
                    <span dir="ltr" className="flex items-center gap-1 font-bold tabular-nums">
                      <IconOrder size={13} />#{fmtRef(r.order_number)}
                    </span>
                  ) : (
                    <Badge variant="warning">{V.manual}</Badge>
                  )}
                  {!r.manual && (
                    <Badge variant="danger">{STATUS_LABELS[r.status] ?? r.status}</Badge>
                  )}
                </span>
                <span className="block text-ink-muted">
                  {FAIL_REASONS[r.reason] ?? r.reason}
                  {r.note && ` · ${r.note}`}
                </span>
              </span>
              <span dir="ltr" className="shrink-0 text-xs text-ink-muted">
                {fmtDateTime(r.closed_at)}
              </span>
            </li>
          ))}
        </ul>
      )}

      {error && (
        <Alert className="mt-3">{error}</Alert>
      )}

      {/* **الإنذارُ اليدويّ** — لما لا طلبَ يشهد عليه. */}
      {issuing ? (
        <div className="mt-4 space-y-2 rounded-control border border-line p-3">
          <p className="text-xs text-ink-muted">{V.issueHint}</p>
          <Input
            id="warn-reason"
            label={V.reason}
            required
            value={reason}
            onChange={(e) => setReason(e.target.value)}
          />
          <Input
            id="warn-note"
            label={V.note}
            value={note}
            onChange={(e) => setNote(e.target.value)}
          />
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setIssuing(false)}>
              {m.common.cancel}
            </Button>
            <Button variant="danger" disabled={busy || !reason.trim()} onClick={() => void issue()}>
              {V.issue}
            </Button>
          </div>
        </div>
      ) : (
        <div className="mt-4 flex justify-end">
          <Button variant="secondary" onClick={() => setIssuing(true)}>
            <span className="flex items-center gap-1.5">
              <IconAdd size={15} />
              {V.issue}
            </span>
          </Button>
        </div>
      )}
    </Modal>
  );
}
