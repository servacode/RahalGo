"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الإسنادُ اليدويّ — بالقرب، وبتأكيد، وبسبب** (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ١٠)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كان ضغطةً واحدةً على قائمةٍ بلا ترتيبٍ ولا مسافة**، وملاحظتُه فارغةٌ في
 * السجلّ دائماً، **وفشلُ الجلب يُقرأ «لا سائق في الدوام».**
 *
 * **والقائمةُ من الخادم** (`GET /orders/{id}/assign-candidates`): من في الدوام
 * وحدَه، مرتّبون بالقرب من نقطة الاستلام. **ومن ترك الطلبَ يأتي آخرَها بعلامة** —
 * والإسنادُ اليدويُّ يُرجعه إليه (قرارُ المالك ٢٠٢٦-١١-٠٨).
 */

import { useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  FormActions,
  LoadingState,
  Modal,
  Money,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const A = m.admin.ordersPage.board.assign;
const OP = m.admin.ordersPage;

interface Candidate {
  id: string;
  full_name: string;
  phone: string;
  distance_m: number | null;
  cash_held: number;
  open_orders: number;
  location_at: string | null;
  left_before?: boolean;
}

function distanceText(d: number): string {
  if (d < 1000) return OP.transferMeters.replace("{n}", fmtNum(Math.round(d)));
  return OP.transferKm.replace("{n}", fmtNum(Math.round(d / 100) / 10));
}

export function AssignDialog({
  orderId,
  orderNumber,
  onClose,
  onDone,
}: {
  orderId: string;
  orderNumber: number;
  onClose: () => void;
  onDone: () => void;
}) {
  const [list, setList] = useState<Candidate[] | null>(null);
  const [staleMin, setStaleMin] = useState(15);
  const [loadErr, setLoadErr] = useState("");
  const [pick, setPick] = useState<Candidate | null>(null);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let live = true;
    setList(null);
    setLoadErr("");
    api<{ drivers: Candidate[]; stale_location_minutes?: number }>(
      `/api/v1/admin/orders/${orderId}/assign-candidates`,
    )
      .then((r) => {
        if (!live) return;
        setList(r.drivers);
        if (typeof r.stale_location_minutes === "number") setStaleMin(r.stale_location_minutes);
      })
      .catch((e) => {
        // **والفشلُ يُقال فشلاً** (المشكلة ٣١) — لا «لا سائق في الدوام».
        if (live) setLoadErr(errorText(e) || A.failed);
      });
    return () => {
      live = false;
    };
  }, [orderId, attempt]);

  async function send() {
    if (!pick) return;
    setBusy(true);
    setErr("");
    try {
      await api(`/api/v1/admin/orders/${orderId}/assign`, {
        method: "POST",
        // **ولا سببَ** (قرارُ المالك ٢٠٢٦-١١-٠٦): «الإدارة تسند الطلب مباشر بدون سبب».
        body: JSON.stringify({ driver_id: pick.id }),
      });
      onDone();
    } catch (e) {
      setErr(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  const stale = (iso: string | null) =>
    !iso || Date.now() - Date.parse(iso) > staleMin * 60_000;

  return (
    <Modal open onClose={onClose} title={A.title.replace("{n}", `#${orderNumber}`)}>
      {pick ? (
        <div className="space-y-3">
          <p className="text-sm font-medium text-ink">{A.confirmTitle}</p>
          <p className="text-sm text-ink-muted">
            {A.confirmBody
              .replace("{n}", `#${orderNumber}`)
              .replace("{name}", pick.full_name || pick.phone)}
          </p>
          {err && <Alert>{err}</Alert>}
          <FormActions
            busy={busy}
            onSave={() => void send()}
            onCancel={() => {
              setPick(null);
              setErr("");
            }}
            saveLabel={A.send}
            cancelLabel={A.back}
          />
        </div>
      ) : (
        <div className="space-y-3">
          <p className="text-xs text-ink-muted">{A.hint}</p>
          {loadErr ? (
            <Alert>
              <span className="flex flex-wrap items-center gap-2">
                {loadErr}
                <Button variant="secondary" onClick={() => setAttempt((n) => n + 1)}>
                  {m.admin.ordersPage.board.retry}
                </Button>
              </span>
            </Alert>
          ) : list === null ? (
            <LoadingState variant="inline" />
          ) : list.length === 0 ? (
            <p className="text-sm text-ink-muted">{A.none}</p>
          ) : (
            <ul className="max-h-96 space-y-2 overflow-y-auto">
              {list.map((d) => (
                <li
                  key={d.id}
                  className="flex flex-wrap items-center gap-2 rounded-control border border-line-soft p-2"
                >
                  <span className="min-w-0 flex-1">
                    <span className="block truncate font-medium text-ink">
                      {d.full_name || d.phone}
                    </span>
                    <span className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-ink-muted">
                      <span>
                        {d.distance_m == null
                          ? A.noDistance
                          : A.distance.replace("{d}", distanceText(d.distance_m))}
                      </span>
                      <span className="inline-flex items-center gap-1">
                        {A.cash}
                        <Money value={d.cash_held} small />
                      </span>
                      <span>{A.orders.replace("{n}", fmtNum(d.open_orders))}</span>
                      {stale(d.location_at) && <Badge variant="danger">{A.stale}</Badge>}
                      {d.left_before && <Badge variant="warning">{A.leftBefore}</Badge>}
                    </span>
                  </span>
                  <Button
                    variant="secondary"
                    onClick={() => {
                      setPick(d);
                      setErr("");
                    }}
                  >
                    {A.choose}
                  </Button>
                </li>
              ))}
            </ul>
          )}
          <div className="flex justify-end">
            <Button variant="secondary" onClick={onClose}>
              {m.common.cancel}
            </Button>
          </div>
        </div>
      )}
    </Modal>
  );
}
