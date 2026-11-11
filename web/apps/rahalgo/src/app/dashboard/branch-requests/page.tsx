"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **طلباتُ الفروع** — طابورُ المكتب لطلباتِ «أضف فرع» من أصحاب المتاجر والمندوبين
 * ══════════════════════════════════════════════════════════════════════
 *
 * الافتراضُ «بانتظار الموافقة». **والموافقةُ بعمولةٍ** تبدأ من العمولة الفعليّة للمتجر الأمّ
 * (`default_commission_percent`) ويعدّلها الموظّف قبل الضغط، **والرفضُ بسببٍ اختياريّ** يصل صاحبَ الطلب.
 * **والمحرّكُ هو الحَكَم**: صلاحيّةُ الاثنين `merchants.verify`، وقرارٌ بُتّ فيه لا يُبتّ مرّتين.
 */

import { useState } from "react";
import Link from "next/link";
import { getMessages, defaultLocale, fmtNum, fmtDateTime, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Chips,
  DataView,
  FormActions,
  Input,
  Modal,
  PageContainer,
  ReloadState,
  Textarea,
  ViewToggle,
  useLiveData,
  useViewMode,
  type DataColumn,
  IconCheck,
  IconClose,
  IconLocation,
  IconStore,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCanCall } from "@/lib/policy";
import { PageHero } from "@/components/admin/PageHero";

const T = getMessages(defaultLocale).admin.branchRequests;

type Status = "pending" | "approved" | "rejected";
type Filter = Status | "all";
const FILTERS: Filter[] = ["pending", "approved", "rejected", "all"];

interface Req {
  id: string;
  parent_merchant_id: string;
  parent_name: string;
  name: string;
  phone: string | null;
  address_text: string | null;
  lat: number;
  lng: number;
  status: Status;
  commission_percent: number | null;
  reject_reason: string | null;
  requested_by_role: "merchant" | "sales";
  requested_by_name: string | null;
  owner_name: string | null;
  owner_phone: string | null;
  branch_merchant_id: string | null;
  created_at: string;
  decided_at: string | null;
  default_commission_percent: number | null;
}
interface Reply {
  requests: Req[];
  general_commission_percent: number | null;
}

/** **لونُ الحالة** — بانتظار برتقاليّ · مقبولٌ أخضر · مرفوضٌ أحمر. */
const variantOf = (s: Status) => (s === "pending" ? "warning" : s === "approved" ? "success" : "danger");

/** **رابطُ الموقع** — خريطةُ العمليّات على النقطة، لا خريطةٌ ثانية. */
const mapHref = (r: Req) => `/dashboard/opsmap?focus=${r.lat},${r.lng}&layer=none`;

export default function BranchRequestsPage() {
  const canCall = useCanCall();
  const canDecide = canCall("POST", "/branch-requests/{id}/approve");
  const [filter, setFilter] = useState<Filter>("pending");
  const [view, setView] = useViewMode("branch-requests", "cards");
  const [approving, setApproving] = useState<Req | null>(null);
  const [rejecting, setRejecting] = useState<Req | null>(null);
  const [percent, setPercent] = useState("");
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [msg, setMsg] = useState<{ tone: "success" | "error"; text: string } | null>(null);

  const data = useLiveData<Reply>(
    () => api<Reply>(`/api/v1/admin/branch-requests?status=${filter}`),
    ["branch_request"],
    [filter],
  );
  const count = useLiveData<{ pending: number }>(
    () => api<{ pending: number }>("/api/v1/admin/branch-requests/count"),
    ["branch_request"],
  );
  const rows = data.data?.requests ?? [];

  const chips = FILTERS.map((f) => ({
    id: f,
    label: T.filter[f],
    count: f === "pending" ? count.data?.pending : undefined,
  }));

  function openApprove(r: Req) {
    setPercent(r.default_commission_percent != null ? String(r.default_commission_percent) : "");
    setApproving(r);
  }
  function openReject(r: Req) {
    setReason("");
    setRejecting(r);
  }

  async function approve() {
    if (!approving) return;
    const raw = percent.trim().replace(",", ".");
    const n = Number(raw);
    if (raw !== "" && (!Number.isFinite(n) || n < 0 || n > 100)) {
      setMsg({ tone: "error", text: T.percentBad });
      return;
    }
    setBusy(true);
    try {
      await api(`/api/v1/admin/branch-requests/${approving.id}/approve`, {
        method: "POST",
        body: JSON.stringify(raw === "" ? {} : { commission_percent: n }),
      });
      setMsg({ tone: "success", text: T.approved });
      setApproving(null);
      data.reload();
      count.reload();
    } catch (e) {
      setMsg({ tone: "error", text: errorText(e) });
      setApproving(null);
      data.reload();
    } finally {
      setBusy(false);
    }
  }

  async function reject() {
    if (!rejecting) return;
    setBusy(true);
    try {
      const why = reason.trim();
      await api(`/api/v1/admin/branch-requests/${rejecting.id}/reject`, {
        method: "POST",
        body: JSON.stringify(why ? { reason: why } : {}),
      });
      setMsg({ tone: "success", text: T.rejected });
      setRejecting(null);
      data.reload();
      count.reload();
    } catch (e) {
      setMsg({ tone: "error", text: errorText(e) });
      setRejecting(null);
      data.reload();
    } finally {
      setBusy(false);
    }
  }

  const cols: DataColumn<Req>[] = [
    {
      id: "name",
      header: T.cols.branch,
      primary: true,
      icon: <IconStore size={14} />,
      cell: (r) => (
        <span className="font-bold">
          {r.name}
          <Badge variant={variantOf(r.status)} className="ms-2">
            {T.status[r.status]}
          </Badge>
        </span>
      ),
    },
    { id: "parent", header: T.cols.parent, cell: (r) => r.parent_name },
    {
      id: "owner",
      header: T.cols.owner,
      cell: (r) =>
        r.owner_name || r.owner_phone ? (
          <span>
            {r.owner_name}
            {r.owner_phone && (
              <span dir="ltr" className="ms-2 text-ink-muted">
                {r.owner_phone}
              </span>
            )}
          </span>
        ) : (
          "—"
        ),
    },
    {
      id: "by",
      header: T.cols.by,
      cell: (r) => (
        <span>
          {T.role[r.requested_by_role]}
          {r.requested_by_name ? ` · ${r.requested_by_name}` : ""}
        </span>
      ),
    },
    {
      id: "phone",
      header: T.cols.phone,
      hide: (r) => !r.phone,
      cell: (r) => (
        <span dir="ltr">{r.phone}</span>
      ),
    },
    { id: "address", header: T.cols.address, hide: (r) => !r.address_text, cell: (r) => r.address_text },
    { id: "at", header: T.cols.at, cell: (r) => fmtDateTime(r.created_at) },
    {
      id: "commission",
      header: T.cols.commission,
      hide: (r) => r.status !== "approved" || r.commission_percent == null,
      cell: (r) => `${fmtNum(r.commission_percent ?? 0)}%`,
    },
    {
      id: "reason",
      header: T.cols.reason,
      hide: (r) => r.status !== "rejected" || !r.reject_reason,
      cell: (r) => r.reject_reason,
    },
  ];

  return (
    <PageContainer>
      <PageHero icon={IconStore} title={T.title} />
      {msg && (
        <Alert tone={msg.tone} className="mb-4" onDismiss={() => setMsg(null)}>
          {msg.text}
        </Alert>
      )}
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <Chips items={chips} value={filter} onChange={setFilter} wrap />
        <ViewToggle view={view} onChange={setView} tableLabel={T.viewTable} cardsLabel={T.viewCards} />
      </div>
      {data.error && !data.data ? (
        <ReloadState label={T.loadFailed} onRetry={data.reload} />
      ) : (
        <DataView
          items={rows}
          loading={data.loading && !data.data}
          getKey={(r) => r.id}
          columns={cols}
          view={view}
          empty={T.empty}
          actions={(r) => (
            <div className="flex flex-wrap gap-2">
              {r.status === "pending" && canDecide && (
                <>
                  <Button size="sm" onClick={() => openApprove(r)}>
                    <IconCheck size={14} /> {T.approve}
                  </Button>
                  <Button size="sm" variant="secondary" onClick={() => openReject(r)}>
                    <IconClose size={14} /> {T.reject}
                  </Button>
                </>
              )}
              <Link className="btn-secondary text-sm" href={mapHref(r)}>
                <IconLocation size={14} /> {T.onMap}
              </Link>
            </div>
          )}
        />
      )}

      <Modal open={approving != null} onClose={() => setApproving(null)} title={T.approveTitle}>
        <div className="flex flex-col gap-4">
          <p className="text-sm text-ink-muted">
            {approving?.name} · {approving?.parent_name}
          </p>
          <Input
            id="branch-commission"
            type="text"
            inputMode="decimal"
            dir="ltr"
            label={T.commission}
            value={percent}
            onChange={(e) => setPercent(e.target.value)}
          />
          <p className="text-sm text-ink-muted">
            {approving?.default_commission_percent != null
              ? T.commissionHint.replace("{n}", fmtNum(approving.default_commission_percent))
              : T.commissionHintGeneral.replace("{n}", fmtNum(data.data?.general_commission_percent ?? 0))}
          </p>
          <FormActions onSave={() => void approve()} onCancel={() => setApproving(null)} busy={busy} saveLabel={T.approve} />
        </div>
      </Modal>

      <Modal open={rejecting != null} onClose={() => setRejecting(null)} title={T.rejectTitle}>
        <div className="flex flex-col gap-4">
          <p className="text-sm text-ink-muted">
            {rejecting?.name} · {rejecting?.parent_name}
          </p>
          <Textarea id="branch-reject-reason" label={T.reasonLabel} rows={3} value={reason} onChange={(e) => setReason(e.target.value)} />
          <FormActions onSave={() => void reject()} onCancel={() => setRejecting(null)} busy={busy} saveLabel={T.reject} tone="danger" />
        </div>
      </Modal>
    </PageContainer>
  );
}
