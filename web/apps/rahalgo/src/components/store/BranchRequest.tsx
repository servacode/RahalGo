"use client";

/**
 * **أضف فرعاً** — طلبُ فرعٍ جديدٍ لمتجري (مرآةُ شاشة «أضف فرع» في تطبيق المتجر).
 *
 * الاسمُ والموقعُ على الخريطة مطلوبان، والهاتفُ اختياريّ. **والطلبُ ينتظر موافقةَ المكتب** —
 * فتحته الإدارةُ فرعاً بعمولةٍ مقرّرة. وتحت النموذج طلباتي بحالتها (وسببُ الرفض إن رُفض).
 */

import { useState } from "react";
import { getMessages, defaultLocale, errorText, fmtDateTime } from "@rahalgo/i18n";
import { Badge, Button, Input, LoadingState, ReloadState, useLiveData } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import { useShopToast } from "@/components/customer/toast";
import { StorePointMap } from "./SetupParts";

const m = getMessages(defaultLocale);
const B = m.store.branch;

type Status = "pending" | "approved" | "rejected";

interface BranchReq {
  id: string;
  parent_merchant_id: string;
  parent_name: string;
  name: string;
  phone: string | null;
  address_text: string | null;
  lat: number;
  lng: number;
  status: Status;
  reject_reason: string | null;
  created_at: string;
}

const VARIANT: Record<Status, "warning" | "success" | "danger"> = {
  pending: "warning",
  approved: "success",
  rejected: "danger",
};

export default function BranchRequest() {
  const { store } = useStore();
  const toast = useShopToast();
  const [name, setName] = useState("");
  const [phone, setPhone] = useState("");
  const [point, setPoint] = useState<{ lat: number; lng: number } | null>(null);
  const [busy, setBusy] = useState(false);
  const list = useLiveData<{ requests: BranchReq[] }>(
    () => api<{ requests: BranchReq[] }>("/api/v1/merchant/branch-requests"),
    ["branch_request"],
  );
  if (!store) return null;
  const suspended = store.status === "suspended";
  const ready = name.trim().length > 0 && point != null;

  async function submit() {
    if (!store || !point) return;
    setBusy(true);
    try {
      const p = phone.trim();
      await api(`/api/v1/merchant/stores/${store.id}/branch-requests`, {
        method: "POST",
        body: JSON.stringify({ name: name.trim(), lat: point.lat, lng: point.lng, ...(p ? { phone: p } : {}) }),
      });
      setName("");
      setPhone("");
      setPoint(null);
      toast(B.sent, "success");
      list.reload();
    } catch (e) {
      toast(errorText(e), "error");
    } finally {
      setBusy(false);
    }
  }

  const rows = list.data?.requests ?? [];
  return (
    <div className="flex flex-col gap-rg-lg" data-testid="store-branch">
      <section className="flex flex-col gap-rg-md" aria-label={B.formTitle}>
        <p className="text-body text-ink-muted">{B.hint}</p>
        <Input
          id="branch-name"
          label={B.name}
          value={name}
          disabled={suspended}
          maxLength={80}
          onChange={(e) => setName(e.target.value)}
        />
        <Input
          id="branch-phone"
          type="tel"
          inputMode="tel"
          dir="ltr"
          label={B.phone}
          value={phone}
          disabled={suspended}
          onChange={(e) => setPhone(e.target.value)}
        />
        <p className="text-body font-bold">{B.point}</p>
        <StorePointMap
          lat={point?.lat ?? null}
          lng={point?.lng ?? null}
          disabled={suspended}
          onPick={(lat, lng) => setPoint({ lat, lng })}
        />
        <Button size="lg" className="w-full font-bold" onClick={() => void submit()} disabled={busy || suspended || !ready}>
          {B.submit}
        </Button>
      </section>

      <section className="flex flex-col gap-rg-md" aria-label={B.listTitle}>
        <h2 className="console-h2">{B.listTitle}</h2>
        {list.loading && !list.data ? (
          <LoadingState variant="list" rows={2} />
        ) : list.error && !list.data ? (
          <ReloadState label={B.loadFailed} onRetry={list.reload} />
        ) : rows.length === 0 ? (
          <p className="text-body text-ink-muted">{B.empty}</p>
        ) : (
          <ul className="flex flex-col gap-rg-sm">
            {rows.map((r) => (
              <li key={r.id} className="panel flex flex-col gap-rg-xs p-rg-md">
                <div className="flex items-center justify-between gap-rg-sm">
                  <span className="min-w-0 truncate text-body-lg font-bold">{r.name}</span>
                  <Badge variant={VARIANT[r.status]}>{B.status[r.status]}</Badge>
                </div>
                <span className="text-body text-ink-muted">
                  {r.parent_name} · {fmtDateTime(r.created_at)}
                </span>
                {r.status === "rejected" && r.reject_reason && (
                  <span className="text-body text-danger">
                    {B.rejectReason} {r.reject_reason}
                  </span>
                )}
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
