"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«آخر طلبات الواتساب»** — قرارُ المالك ٢٠٢٦-١١-٠٥
 * ══════════════════════════════════════════════════════════════════════
 *
 * آخرُ عشرين طلباً بحالها وسائقها وأجرتها (`GET /admin/whatsapp-orders`)، **و«إلغاء الطلب»
 * قبل الاستلام بسببٍ إلزاميّ** (`POST /admin/whatsapp-orders/{id}/cancel`) — والمحرّكُ يحكم:
 * بعد الاستلام يُقال «الحلّ من بطاقة الطلب».
 *
 * **و«بدّل السائق» قبل الاستلام** (قرارُ المالك ٢٠٢٦-١١-٠٨) — يُعاد للطابور بسبب، ثمّ يُختار سائقٌ آخر.
 *
 * **ومن هنا إلى لوحة الطلبات وسجلّها** بفلتر «طلبات الواتساب» — الطلبُ طلبٌ نظاميٌّ هناك.
 *
 * **و«تعديل العنوان» قبل التسليم** (ملحقُ المالك ٢٠٢٦-١١-٠٥، د: «نحن رح نعبيه يدوي») — المستلِمُ
 * كتب عنوانَه بالتفصيل على الواتساب فيعبّيه الموظّف (`POST /admin/whatsapp-orders/{id}/address`)،
 * ويصل السائقَ في لحظته. **و«بلا موقع» على الطلب الذي لم يصل موقعُ مستلِمه بعد.**
 *
 * **والجاريةُ أوّلاً وخمسون** (الخادم) — طلبٌ مفتوحٌ يوم العجقة لا يغيب خلف المنتهية. **والقائمةُ
 * حيّة**: كلُّ حركةٍ في طلب واتساب (وصولُ الموقع، سائقٌ أخذه، تسليم) تُعيد قراءتها.
 */

import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtDamascusTime, fmtMoney } from "@rahalgo/i18n";
import {
  AdminCard,
  AdminEmpty,
  AdminLoading,
  AdminReload,
  Alert,
  Badge,
  Button,
  FormActions,
  IconWhatsApp,
  Modal,
  Row,
  RowList,
  StatusBadge,
  Textarea,
  useLiveEvent,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { fill } from "@/lib/customer/texts";
import { AssignDialog } from "../orders/AssignDialog";
import { AmanahPointsSection } from "../orders/CardSections";

const m = getMessages(defaultLocale);
const W = m.admin.whatsappOrders;

/** **طلبٌ في القائمة** — من حمولة المكتب. */
interface WaRow {
  id: string;
  number: number;
  status: string;
  created_at: string;
  merchant_name: string;
  customer_name: string;
  driver_name?: string | null;
  delivery_fee: number;
  address_text?: string;
  dropoff_known?: boolean;
  /** **ثمنُ الطلب كتبه السائق** (قرارُ المالك ٢٠٢٦-١١-٠٦) — ووصل الزبونَ برسالة. */
  fetch_amount?: number | null;
  /** **«أمانة من شخص»، وانتظارُها موقعَ صاحبها** (قرارُ المالك ٢٠٢٦-١١-١١) — غلافُ القائمة. */
  wa_amanah?: boolean;
  pickup_pending?: boolean;
  /** **نقطتا الأمانة وطرفاها** (قرارُ المالك ٢٠٢٦-١١-١٢) — صندوقا «الاستلام من» و«التسليم لـ». */
  merchant_phone?: string;
  customer_phone?: string;
  customer_phone_masked?: string;
  pickup_lat?: number | null;
  pickup_lng?: number | null;
  pickup_address?: string;
  lat?: number | null;
  lng?: number | null;
}

/** **قبل الاستلام وحدَه** — والمحرّكُ يحرسه (`wa_cancel_after_pickup`). و«مقبول» أمانةٌ تنتظر موقعَ صاحبها. */
const CANCELLABLE = new Set(["accepted", "dispatching", "assigned", "at_pickup"]);
/** **«بدّل السائق» قبل الاستلام** (قرارُ المالك ٢٠٢٦-١١-٠٨) — والمحرّكُ يحرسه، ولا يعود لمن تُرك. */
const DRIVER_CHANGEABLE = new Set(["assigned", "at_pickup"]);
/** **قبل التسليم** — والمحرّكُ يحرسه (`wa_address_closed`). */
const ADDRESS_EDITABLE = new Set(["accepted", "dispatching", "assigned", "at_pickup", "picked_up", "on_the_way", "at_dropoff"]);

export function RecentWhatsAppOrders({ refresh }: { refresh: number }) {
  const [rows, setRows] = useState<WaRow[] | null>(null);
  const [failed, setFailed] = useState(false);
  const [target, setTarget] = useState<WaRow | null>(null);
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  const [done, setDone] = useState("");
  const [addrTarget, setAddrTarget] = useState<WaRow | null>(null);
  const [addrText, setAddrText] = useState("");
  const [swapTarget, setSwapTarget] = useState<WaRow | null>(null);
  const [assignFor, setAssignFor] = useState<WaRow | null>(null);
  // **«بدّل السائق» = اختيارُ السائق الجديد بنفس النافذة** (طلبُ المالك ٢٠٢٦-١١-٠٩) — لا رجوعَ للتوزيع.
  const [swapFor, setSwapFor] = useState<WaRow | null>(null);

  const load = useCallback(async () => {
    try {
      const r = await api<{ orders: WaRow[] }>("/api/v1/admin/whatsapp-orders");
      setRows(r.orders ?? []);
      setFailed(false);
    } catch {
      setFailed(true);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load, refresh]);

  // **حيّةٌ بلا تحديث** — حركةُ طلب واتساب أو إشارةُ المكتب (`touch("orders")`) تُعيد القراءة،
  // **مجمّعةً بنصف ثانية** فلا تُقرأ عشرَ مرّاتٍ لعشر حركات.
  const pending = useRef<ReturnType<typeof setTimeout> | null>(null);
  useLiveEvent((event) => {
    const origin = (event.order as { origin?: string } | undefined)?.origin;
    if ((event.type === "order" && origin === "whatsapp") || event.type === "orders") {
      if (pending.current) clearTimeout(pending.current);
      pending.current = setTimeout(() => void load(), 500);
    }
  });
  useEffect(
    () => () => {
      if (pending.current) clearTimeout(pending.current);
    },
    [],
  );

  async function cancel() {
    if (!target || busy) return;
    if (!reason.trim()) {
      setErr(W.cancelReason);
      return;
    }
    setBusy(true);
    setErr("");
    try {
      await api(`/api/v1/admin/whatsapp-orders/${target.id}/cancel`, {
        method: "POST",
        body: JSON.stringify({ reason: reason.trim() }),
      });
      setDone(fill(W.cancelled, { n: target.number }));
      setTarget(null);
      setReason("");
      void load();
    } catch (e) {
      setErr(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  /** **«بدّل السائق»** — يُعاد للطابور بسببٍ يُسجَّل، ثمّ يُختار سائقٌ أو يُترك للتوزيع. */
  async function releaseDriver() {
    if (!swapTarget || busy) return;
    if (!reason.trim()) {
      setErr(W.changeDriverReason);
      return;
    }
    setBusy(true);
    setErr("");
    try {
      await api(`/api/v1/admin/orders/${swapTarget.id}/transition`, {
        method: "POST",
        body: JSON.stringify({ to: "dispatching", note: reason.trim() }),
      });
      setDone(fill(W.driverReleased, { n: swapTarget.number }));
      setAssignFor(swapTarget);
      setSwapTarget(null);
      setReason("");
      void load();
    } catch (e) {
      setErr(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  /** **«تعديل العنوان»** — يُحفظ ويصل السائقَ، والسجلُّ يحمل القديمَ والجديد. */
  async function saveAddress() {
    if (!addrTarget || busy) return;
    if (!addrText.trim()) {
      setErr(fill(W.missing, { fields: W.address }));
      return;
    }
    setBusy(true);
    setErr("");
    try {
      await api(`/api/v1/admin/whatsapp-orders/${addrTarget.id}/address`, {
        method: "POST",
        body: JSON.stringify({ address_text: addrText.trim() }),
      });
      setDone(fill(W.addressSaved, { n: addrTarget.number }));
      setAddrTarget(null);
      void load();
    } catch (e) {
      setErr(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <AdminCard
      title={W.recent}
      padded={false}
      actions={
        <span className="flex flex-wrap gap-3 text-sm">
          <Link className="underline" href="/dashboard/orders?kind=whatsapp">
            {W.allOpen}
          </Link>
          <Link className="underline" href="/dashboard/history?kind=whatsapp">
            {W.history}
          </Link>
        </span>
      }
    >
      {done && (
        <div className="px-4 pb-3 lg:px-5">
          <Alert tone="success">{done}</Alert>
        </div>
      )}
      {rows === null && !failed && <AdminLoading rows={3} />}
      {failed && <AdminReload onRetry={() => void load()} />}
      {rows && rows.length === 0 && <AdminEmpty icon={IconWhatsApp} title={W.recentEmpty} />}
      {rows && rows.length > 0 && (
        <RowList className="rounded-none border-0">
          {rows.map((o) => (
            <Row
              key={o.id}
              title={
                <span className="flex flex-wrap items-center gap-2">
                  <Link className="underline" href={`/dashboard/orders?id=${encodeURIComponent(o.id)}`}>
                    #{o.number}
                  </Link>
                  <StatusBadge domain="order" status={o.status} />
                  {o.wa_amanah && <Badge variant="info">{W.amanahBadge}</Badge>}
                </span>
              }
              meta={
                o.wa_amanah ? (
                  <>
                    {/* **الأمانةُ بصندوقين جنب بعض** (قرارُ المالك ٢٠٢٦-١١-١٢) — كلٌّ بحال موقعه. */}
                    {fmtDamascusTime(o.created_at)} · {o.driver_name || W.noDriver}
                    <span className="mt-2 block text-ink">
                      <AmanahPointsSection
                        o={{ ...o, customer_phone: o.customer_phone ?? "", address_text: o.address_text ?? "" }}
                        live={ADDRESS_EDITABLE.has(o.status)}
                      />
                    </span>
                  </>
                ) : (
                <>
                  {o.merchant_name} ← {o.customer_name} · {fmtDamascusTime(o.created_at)} ·{" "}
                  {o.driver_name || W.noDriver}
                  {o.address_text ? <span className="block">{o.address_text}</span> : null}
                  {o.pickup_pending && <span className="block text-warning">{W.pickupPending}</span>}
                  {o.dropoff_known === false && ADDRESS_EDITABLE.has(o.status) && (
                    <span className="block text-warning">{W.noPoint}</span>
                  )}
                  {o.fetch_amount != null && (
                    <span className="block font-bold">{fill(W.goodsLine, { goods: fmtMoney(o.fetch_amount) })}</span>
                  )}
                </>
                )
              }
              trailing={
                <span className="flex flex-col items-end gap-1">
                  <span className="font-bold">{fmtMoney(o.delivery_fee)}</span>
                  {ADDRESS_EDITABLE.has(o.status) && (
                    <Button
                      size="sm"
                      variant="secondary"
                      onClick={() => {
                        setAddrTarget(o);
                        setAddrText(o.address_text ?? "");
                        setErr("");
                        setDone("");
                      }}
                    >
                      {W.editAddress}
                    </Button>
                  )}
                  {DRIVER_CHANGEABLE.has(o.status) && o.driver_name && (
                    <Button
                      size="sm"
                      variant="secondary"
                      onClick={() => {
                        setSwapFor(o);
                        setReason("");
                        setErr("");
                        setDone("");
                      }}
                    >
                      {W.changeDriver}
                    </Button>
                  )}
                  {CANCELLABLE.has(o.status) && (
                    <Button
                      size="sm"
                      variant="danger-outline"
                      onClick={() => {
                        setTarget(o);
                        setReason("");
                        setErr("");
                        setDone("");
                      }}
                    >
                      {W.cancel}
                    </Button>
                  )}
                </span>
              }
            />
          ))}
        </RowList>
      )}

      <Modal open={target !== null} onClose={() => setTarget(null)} title={target ? fill(W.cancelTitle, { n: target.number }) : W.cancel}>
        <div className="space-y-3">
          <Textarea label={W.cancelReason} value={reason} maxLength={300} autoGrow onChange={(e) => setReason(e.target.value)} />
          {err && <Alert tone="error">{err}</Alert>}
          <FormActions
            onSave={() => void cancel()}
            onCancel={() => setTarget(null)}
            busy={busy}
            tone="danger"
            saveLabel={W.cancelConfirm}
          />
        </div>
      </Modal>

      <Modal
        open={swapTarget !== null}
        onClose={() => setSwapTarget(null)}
        title={swapTarget ? fill(W.changeDriverTitle, { n: swapTarget.number }) : W.changeDriver}
      >
        <div className="space-y-3">
          <p className="text-sm text-ink-muted">{W.changeDriverHint}</p>
          <Textarea label={W.changeDriverReason} value={reason} maxLength={300} autoGrow onChange={(e) => setReason(e.target.value)} />
          {err && <Alert tone="error">{err}</Alert>}
          <FormActions
            onSave={() => void releaseDriver()}
            onCancel={() => setSwapTarget(null)}
            busy={busy}
            saveLabel={W.changeDriverSave}
          />
        </div>
      </Modal>

      {swapFor && (
        <AssignDialog
          orderId={swapFor.id}
          orderNumber={swapFor.number}
          swapFrom={swapFor.driver_name ?? ""}
          onClose={() => setSwapFor(null)}
          onDone={() => {
            setSwapFor(null);
            void load();
          }}
        />
      )}

      {assignFor && (
        <AssignDialog
          orderId={assignFor.id}
          orderNumber={assignFor.number}
          onClose={() => setAssignFor(null)}
          onDone={() => {
            setAssignFor(null);
            void load();
          }}
        />
      )}

      <Modal
        open={addrTarget !== null}
        onClose={() => setAddrTarget(null)}
        title={addrTarget ? fill(W.editAddressTitle, { n: addrTarget.number }) : W.editAddress}
      >
        <div className="space-y-3">
          <Textarea label={W.address} value={addrText} maxLength={300} autoGrow onChange={(e) => setAddrText(e.target.value)} />
          {err && <Alert tone="error">{err}</Alert>}
          <FormActions
            onSave={() => void saveAddress()}
            onCancel={() => setAddrTarget(null)}
            busy={busy}
            saveLabel={W.editAddressSave}
          />
        </div>
      </Modal>
    </AdminCard>
  );
}
