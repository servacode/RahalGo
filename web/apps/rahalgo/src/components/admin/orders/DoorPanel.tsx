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
import {
  getMessages,
  defaultLocale,
  fmtNum,
  fmtTime,
  errorText,
} from "@rahalgo/i18n";
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
  /** **هاتفُ المتجر** — للّوحة عند المتجر (تدقيقُ اللوحة ٢٠٢٦-١٠-٠٣). */
  store_phone?: string;
}

/** **رقمٌ يُتّصل به** — زرٌّ واحدُ الشكل للزبون والمتجر والسائق. */
function PhoneLink({ label, phone }: { label: string; phone: string }) {
  return (
    <div className="flex flex-wrap items-center gap-2 text-sm">
      <span className="text-xs text-ink-muted">{label}</span>
      <a
        href={`tel:${phone}`}
        className="inline-flex items-center gap-1.5 rounded-control border border-line px-3 py-1.5 text-sm text-ink"
      >
        <IconPhone size={15} />
        <span dir="ltr">{phone}</span>
        <span className="text-xs text-ink-muted">{D.call}</span>
      </a>
    </div>
  );
}

export function DoorPanel({
  orderId,
  orderNumber,
  atDoor,
  atStore = false,
  beforeStore = false,
  canTransfer = true,
  driverPhone,
  customerPhone,
  door,
  cashBanDays,
  onChanged,
  onReturned,
}: {
  orderId: string;
  orderNumber: number;
  /**
   * **عند الباب أم في الطريق** (٢٠٢٦-١٠-٠٣) — الأمرُ نفسُه، ونصُّه «سلّم الآن» عند
   * الباب و«أكمل التوصيل» قبله.
   */
  atDoor: boolean;
  /**
   * **عند المتجر** (قرارُ المالك ٢٠٢٦-١٠-٠٣: «ينتظر الإدارة تحلّ المشكلة») — «استلم الطلب»
   * أو «حوّل لمتجرٍ آخر» بدل «سلّم» و«عُد إلى المكتب».
   */
  atStore?: boolean;
  /**
   * **قبل المتجر** (قرارُ المالك ٢٠٢٦-١٠-٠٣: «الإلغاءُ قبل المتجر الإدارةُ تقرّره، لأنّ الزبونَ لا
   * يبقى عنده زرُّ إلغاء») — «أكمل الطلب» أو «ألغِ الطلب»، **ولا «عُد إلى المكتب»: لا بضاعةَ معه.**
   */
  beforeStore?: boolean;
  /** **و«لدي توصيلة» لا تُحوَّل** — متجرُها مُنشئُها. */
  canTransfer?: boolean;
  customerPhone: string;
  /** **هاتفُ السائق** — المكتبُ يكلّمه في كلّ مرحلة. */
  driverPhone?: string | null;
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
  const [dialog, setDialog] = useState<"" | "deliver" | "return" | "cancel">(
    "",
  );
  const [note, setNote] = useState("");
  const [fault, setFault] = useState<Fault | "">("");
  const [reason, setReason] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState("");

  function open(kind: "deliver" | "return" | "cancel") {
    setNote("");
    setReason("");
    setFault("");
    setErr("");
    setNotice("");
    setDialog(kind);
  }

  // **والإلغاءُ قبل المتجر انتقالٌ عامّ** — لا أمرَ باب: الطلبُ يُغلق ويُخبَر السائق.
  async function cancelOrder() {
    if (note.trim() === "") {
      setErr(D.noteMissing);
      return;
    }
    setBusy(true);
    setErr("");
    try {
      await api(`/api/v1/admin/orders/${orderId}/transition`, {
        method: "POST",
        body: JSON.stringify({ to: "cancelled", note: note.trim() }),
      });
      setDialog("");
      onChanged();
    } catch (e) {
      setErr(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  async function send() {
    const action = dialog === "deliver" ? "deliver_now" : "return_to_office";
    // **وعند المتجر الذنبُ ذنبُه** — والمكتبُ اتّصل به فعرف.
    const effFault = atStore ? "merchant" : fault;
    if (action === "return_to_office") {
      if (!effFault) {
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
            : { action, fault: effFault, reason, note: note.trim() },
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
        {beforeStore
          ? D.beforeStore
          : atStore
            ? D.atStore.replace(
                "{n}",
                fmtNum(Math.max(door?.waited_min ?? 0, 0)),
              )
            : !atDoor
              ? D.onTheWay
              : door && door.waited_min >= 0
                ? D.waited.replace("{n}", fmtNum(door.waited_min))
                : D.waitedUnknown}
      </p>
      <p className="text-xs text-ink-muted">
        {beforeStore
          ? D.hintBefore
          : atStore
            ? D.hintStore
            : atDoor
              ? D.hint
              : D.hintTrip}
      </p>

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

      {/* **ومن يُكلَّم يتبع الموضع** (تدقيقُ اللوحة ٢٠٢٦-١٠-٠٣): عند المتجر المتجرُ، وفي الطريق وعند
          الباب الزبون — **والسائقُ في كلّ مرّة.** */}
      {atStore
        ? door?.store_phone && (
            <PhoneLink label={D.storePhone} phone={door.store_phone} />
          )
        : customerPhone && (
            <PhoneLink label={D.customerPhone} phone={customerPhone} />
          )}
      {driverPhone && <PhoneLink label={D.driverPhone} phone={driverPhone} />}

      {door?.instruction === "deliver_now" && (
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant="success">
            {beforeStore
              ? D.instructionSentKeep
              : atStore
                ? D.instructionSentCollect
                : atDoor
                  ? D.instructionSent
                  : D.instructionSentContinue}
          </Badge>
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
            {beforeStore
              ? D.keep
              : atStore
                ? D.collect
                : atDoor
                  ? D.deliverNow
                  : D.continue}
          </Button>
          {beforeStore && (
            <Button
              variant="danger"
              disabled={busy}
              onClick={() => open("cancel")}
            >
              {D.cancelOrder}
            </Button>
          )}
          {!beforeStore && (!atStore || canTransfer) && (
            <Button
              variant="danger"
              disabled={busy}
              onClick={() => open("return")}
            >
              {atStore ? D.transfer : D.returnToOffice}
            </Button>
          )}
        </div>
      )}

      <Modal
        open={dialog === "deliver"}
        onClose={() => setDialog("")}
        title={
          beforeStore
            ? D.keepTitle
            : atStore
              ? D.collectTitle
              : atDoor
                ? D.deliverTitle
                : D.continueTitle
        }
      >
        <div className="space-y-3">
          <p className="text-sm text-ink-muted">
            {beforeStore
              ? D.keepHint
              : atStore
                ? D.collectHint
                : atDoor
                  ? D.deliverHint
                  : D.continueHint}
          </p>
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
        open={dialog === "cancel"}
        onClose={() => setDialog("")}
        title={D.cancelTitle}
      >
        <div className="space-y-3">
          <p className="text-sm text-ink-muted">{D.cancelHint}</p>
          <Textarea
            id={`door-cancel-${orderId}`}
            label={D.cancelNote}
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
            onSave={() => void cancelOrder()}
            onCancel={() => setDialog("")}
            saveLabel={D.cancelSend}
            tone="danger"
          />
        </div>
      </Modal>

      <Modal
        open={dialog === "return"}
        onClose={() => setDialog("")}
        title={atStore ? D.transferTitle : D.returnTitle}
      >
        <div className="space-y-4">
          <p className="text-sm text-ink-muted">
            {atStore ? D.transferHint : D.returnHint}
          </p>
          {!atStore && (
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
          )}
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
                {reportLabel
                  ? `${D.reasonLast} — ${reportLabel}`
                  : D.reasonLast}
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
