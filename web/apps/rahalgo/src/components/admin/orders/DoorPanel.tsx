"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عند باب الزبون — السائقُ يُبلّغ، والمكتبُ يقرّر**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك مساءَ ٢٠٢٦-١٠-٠٢، البند ١: «دائماً إذا في مشكلة بين السائق
 *  والزبون يكون الردّ: انتظر، الإدارة تقوم بالتواصل مع الزبون، ويبقى الطلبُ
 *  مع السائق إلى أن تُحلّ القصّة… وقتها الإدارةُ هي تُنهي الطلبَ من عندها».)
 *
 * **ما يُقرأ**: آخرُ بلاغٍ وكم ينتظر السائقُ وهاتفُ الزبون — **كلُّها من
 * المحرّك** (`door` في ردّ الإدارة، `orders/door_view.go`).
 *
 * **وما يُفعل**: أمران لا ثالثَ لهما — `POST /admin/orders/{id}/door-resolution`
 * بقدرة `orders.intervene`.
 *
 *	سلّم الآن       الطلبُ يبقى مع السائق ويصله أمرٌ عاجل
 *	عُد إلى المكتب  الطلبُ يُنهى فشلاً **بذنبٍ يكتبه المكتب**
 */

import { useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtTime, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  FormActions,
  IconPhone,
  Modal,
  Radio,
  Select,
  Textarea,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);
const D = m.admin.ordersPage.door;
const REPORTS: Record<string, string> = D.reports;
const FAULTS = ["customer", "driver", "merchant", "platform"] as const;
type Fault = (typeof FAULTS)[number];

/** **حالُ الباب كما يرسله المحرّك** — مرآةُ `orders.DoorView`. */
export interface DoorView {
  report_code: string;
  report_note: string;
  report_at: string | null;
  suggested_fault: string;
  arrived_at: string | null;
  /** **وسالبٌ إن لم يُعرف** — لا يُعرض صفراً. */
  waited_min: number;
  instruction: string;
  instruction_note: string;
  instruction_at: string | null;
}

export function DoorPanel({
  orderId,
  orderNumber,
  customerPhone,
  door,
  cashBanDays,
  onChanged,
  onReturned,
}: {
  orderId: string;
  orderNumber: number;
  customerPhone: string;
  door: DoorView | undefined;
  /** **مدّةُ منع النقد من الإعدادات** — لا رقمٌ مكتوبٌ هنا. */
  cashBanDays: number;
  onChanged: () => void;
  /**
   * **بعد «عُد إلى المكتب» يخرج الطلبُ من شاشة العمل** — فالتذكيرُ بتسوية
   * البضاعة يُرفع إلى الشاشة، **ولا يضيع مع البطاقة.**
   */
  onReturned: (orderNumber: number) => void;
}) {
  const { can } = useAuth();
  const canResolve = can("orders.intervene");
  const [dialog, setDialog] = useState<"" | "deliver" | "return">("");
  const [note, setNote] = useState("");
  const [fault, setFault] = useState<Fault | "">("");
  const [reason, setReason] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState("");

  function open(kind: "deliver" | "return") {
    setNote("");
    setReason("");
    setFault("");
    setErr("");
    setNotice("");
    setDialog(kind);
  }

  async function send() {
    const action = dialog === "deliver" ? "deliver_now" : "return_to_office";
    if (action === "return_to_office") {
      if (!fault) {
        setErr(D.faultMissing);
        return;
      }
      if (note.trim() === "") {
        setErr(D.noteMissing);
        return;
      }
    }
    setBusy(true);
    setErr("");
    try {
      await api(`/api/v1/admin/orders/${orderId}/door-resolution`, {
        method: "POST",
        body: JSON.stringify(
          action === "deliver_now"
            ? { action, note: note.trim() }
            : { action, fault, reason, note: note.trim() },
        ),
      });
      setDialog("");
      if (action === "return_to_office") onReturned(orderNumber);
      else setNotice(D.sent);
      onChanged();
    } catch (e) {
      setErr(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  const reportLabel = door?.report_code
    ? (REPORTS[door.report_code] ?? door.report_code)
    : "";
  const suggested = door?.suggested_fault
    ? (D.faults as Record<string, string>)[door.suggested_fault]
    : "";

  return (
    // **وضغطُ اللوحة لا يصل البطاقةَ تحتها** — ولا ما في نوافذها.
    <div
      className="w-full space-y-3 rounded-control border border-warning-edge bg-warning-tint p-3"
      onClick={(e) => e.stopPropagation()}
    >
      {/* **والعنوانُ تسميةُ الحقل فوق اللوحة** (`header` في البطاقة) —
          **فأوّلُ سطرٍ فيها كم ينتظر رجلٌ في الشارع.** */}
      <p className="text-sm font-semibold text-warning">
        {door && door.waited_min >= 0
          ? D.waited.replace("{n}", fmtNum(door.waited_min))
          : D.waitedUnknown}
      </p>
      <p className="text-xs text-ink-muted">{D.hint}</p>

      <div className="space-y-1 text-sm">
        <p className="text-xs text-ink-muted">{D.lastReport}</p>
        {reportLabel ? (
          <p className="flex flex-wrap items-center gap-2">
            <span className="font-medium text-ink">{reportLabel}</span>
            {door?.report_at && (
              <span className="text-xs text-ink-muted">
                {D.reportAt.replace("{t}", fmtTime(door.report_at))}
              </span>
            )}
          </p>
        ) : (
          <p className="text-ink-muted">{D.noReport}</p>
        )}
        {door?.report_note && (
          <p className="text-xs text-ink">
            {D.driverNote}: {door.report_note}
          </p>
        )}
        {suggested && (
          <p className="text-xs text-ink-muted">
            {D.suggested.replace("{f}", suggested)}
          </p>
        )}
      </div>

      {customerPhone && (
        <div className="flex flex-wrap items-center gap-2 text-sm">
          <span className="text-xs text-ink-muted">{D.customerPhone}</span>
          <a
            href={`tel:${customerPhone}`}
            className="inline-flex items-center gap-1.5 rounded-control border border-line px-3 py-1.5 text-sm text-ink"
          >
            <IconPhone size={15} />
            <span dir="ltr">{customerPhone}</span>
            <span className="text-xs text-ink-muted">{D.call}</span>
          </a>
        </div>
      )}

      {door?.instruction === "deliver_now" && (
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant="success">{D.instructionSent}</Badge>
          {door.instruction_at && (
            <span className="text-xs text-ink-muted">
              {D.instructionAt.replace("{t}", fmtTime(door.instruction_at))}
            </span>
          )}
          {door.instruction_note && (
            <span className="text-xs text-ink">{door.instruction_note}</span>
          )}
        </div>
      )}

      {notice && <Alert tone="success">{notice}</Alert>}

      {canResolve && (
        <div className="flex flex-wrap gap-2">
          <Button disabled={busy} onClick={() => open("deliver")}>
            {D.deliverNow}
          </Button>
          <Button variant="danger" disabled={busy} onClick={() => open("return")}>
            {D.returnToOffice}
          </Button>
        </div>
      )}

      <Modal
        open={dialog === "deliver"}
        onClose={() => setDialog("")}
        title={D.deliverTitle}
      >
        <div className="space-y-3">
          <p className="text-sm text-ink-muted">{D.deliverHint}</p>
          <Textarea
            id={`door-deliver-${orderId}`}
            label={D.deliverNote}
            value={note}
            maxLength={300}
            autoGrow
            onChange={(e) => setNote(e.target.value)}
          />
          {err && <Alert>{err}</Alert>}
          <FormActions
            busy={busy}
            onSave={() => void send()}
            onCancel={() => setDialog("")}
            saveLabel={D.deliverSend}
          />
        </div>
      </Modal>

      <Modal
        open={dialog === "return"}
        onClose={() => setDialog("")}
        title={D.returnTitle}
      >
        <div className="space-y-4">
          <p className="text-sm text-ink-muted">{D.returnHint}</p>
          <fieldset className="space-y-2">
            <legend className="mb-1.5 text-sm font-medium text-ink">
              {D.faultLabel}
            </legend>
            {FAULTS.map((f) => (
              <div key={f} className="space-y-0.5">
                <Radio
                  id={`door-fault-${orderId}-${f}`}
                  name={`door-fault-${orderId}`}
                  checked={fault === f}
                  onChange={() => {
                    setFault(f);
                    setErr("");
                  }}
                  label={D.faults[f]}
                />
                <p className="ps-7 text-xs text-ink-muted">
                  {D.faultEffects[f].replace("{days}", fmtNum(cashBanDays))}
                </p>
              </div>
            ))}
          </fieldset>
          <div>
            <label
              htmlFor={`door-reason-${orderId}`}
              className="mb-1.5 block text-sm font-medium text-ink"
            >
              {D.reasonLabel}
            </label>
            <Select
              id={`door-reason-${orderId}`}
              value={reason}
              onChange={(e) => setReason(e.target.value)}
            >
              <option value="">
                {reportLabel ? `${D.reasonLast} — ${reportLabel}` : D.reasonLast}
              </option>
              {Object.entries(REPORTS).map(([code, label]) => (
                <option key={code} value={code}>
                  {label}
                </option>
              ))}
            </Select>
          </div>
          <Textarea
            id={`door-note-${orderId}`}
            label={D.returnNote}
            value={note}
            maxLength={300}
            autoGrow
            onChange={(e) => {
              setNote(e.target.value);
              setErr("");
            }}
          />
          {err && <Alert>{err}</Alert>}
          <FormActions
            busy={busy}
            onSave={() => void send()}
            onCancel={() => setDialog("")}
            saveLabel={D.returnSend}
            tone="danger"
          />
        </div>
      </Modal>
    </div>
  );
}
