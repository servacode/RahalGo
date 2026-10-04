"use client";

/**
 * **موافقاتُ الماليّة على العروض** — كودٌ أو خصمُ صنفٍ فوق حدّ المحتوى
 * (٢٠٪ أو ٥٠ استخداماً) ينتظر هنا (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ١).
 *
 * ومن اقترح لا يوافق — يردّها الخادم (`approval.Check`).
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtDate, fmtMoney, errorText } from "@rahalgo/i18n";
import { Alert, Button, DataView, type DataColumn } from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const O = m.admin.promosOwner;

interface Row {
  id: string;
  amount: number;
  note: string;
  target_kind: "promo" | "offer";
  proposed_by_name: string;
  created_at: string;
}

export default function PromoApprovalsTab({
  canDecide,
  onChanged,
}: {
  canDecide: boolean;
  onChanged: () => void;
}) {
  const [rows, setRows] = useState<Row[] | null>(null);
  const [error, setError] = useState("");
  const [busyId, setBusyId] = useState("");

  const load = useCallback(() => {
    api<{ approvals: Row[] }>("/api/v1/admin/promo-approvals?status=pending")
      .then((r) => {
        setRows(r.approvals ?? []);
        setError("");
      })
      .catch((e) => {
        setRows([]);
        setError(errorText(e));
      });
  }, []);
  useEffect(load, [load]);

  async function decide(r: Row, approve: boolean) {
    if (busyId) return;
    setBusyId(r.id);
    setError("");
    try {
      await api(`/api/v1/admin/promo-approvals/${r.id}/${approve ? "approve" : "reject"}`, {
        method: "POST",
        body: JSON.stringify({}),
      });
      load();
      onChanged();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusyId("");
    }
  }

  const columns: DataColumn<Row>[] = [
    {
      id: "what",
      header: O.apWhat,
      primary: true,
      cell: (r) => `${r.target_kind === "promo" ? O.kindPromo : O.kindOffer} — ${r.note}`,
    },
    { id: "amount", header: O.apAmount, cell: (r) => (r.amount > 0 ? fmtMoney(r.amount) : O.apUnknown) },
    { id: "by", header: O.apBy, cell: (r) => r.proposed_by_name || "—" },
    { id: "when", header: O.apWhen, cell: (r) => fmtDate(r.created_at) },
  ];

  return (
    <div className="space-y-4">
      {error && <Alert>{error}</Alert>}
      <DataView
        items={rows ?? []}
        loading={rows === null}
        getKey={(r) => r.id}
        columns={columns}
        view="table"
        empty={O.apEmpty}
        actions={
          canDecide
            ? (r) => (
                <div className="flex gap-2">
                  <Button disabled={busyId === r.id} onClick={() => void decide(r, true)}>
                    {busyId === r.id ? O.busy : O.apApprove}
                  </Button>
                  <Button
                    variant="danger"
                    disabled={busyId === r.id}
                    onClick={() => void decide(r, false)}
                  >
                    {O.apReject}
                  </Button>
                </div>
              )
            : undefined
        }
      />
    </div>
  );
}
