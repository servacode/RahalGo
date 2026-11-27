"use client";

/**
 * **طلباتُ تسجيل الكباتن** — ما يكتبه المتقدّمون في قائمة بوت الواتساب.
 * الافتراضُ «جديد»، وتغييرُ الحالة (جديد · تم التواصل · مرفوض) بقدرة `drivers.manage`.
 */

import { useState } from "react";
import { getMessages, defaultLocale, fmtDateTime, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Chips,
  DataView,
  PageContainer,
  ReloadState,
  Select,
  ViewToggle,
  useLiveData,
  useViewMode,
  type DataColumn,
  IconSignup,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCanCall } from "@/lib/policy";
import { PageHero } from "@/components/admin/PageHero";

const T = getMessages(defaultLocale).admin.captainApplications;

type Status = "new" | "contacted" | "rejected";
type Filter = Status | "all";
const FILTERS: Filter[] = ["new", "contacted", "rejected", "all"];
const STATUSES: Status[] = ["new", "contacted", "rejected"];

interface App {
  id: string;
  phone: string;
  full_name: string;
  age: string;
  area: string;
  has_ebike: boolean;
  has_motorbike: boolean;
  has_license: boolean;
  has_android: boolean;
  experience: string;
  hours: string;
  status: Status;
  created_at: string;
  updated_at: string;
}

const variantOf = (s: Status) => (s === "new" ? "warning" : s === "contacted" ? "success" : "danger");
const yn = (b: boolean) => (b ? T.yes : T.no);
const waHref = (phone: string) => `https://wa.me/${phone.replace(/\D/g, "")}`;

export default function CaptainApplicationsPage() {
  const canCall = useCanCall();
  const canManage = canCall("POST", "/captain-applications/{id}/status");
  const [filter, setFilter] = useState<Filter>("new");
  const [view, setView] = useViewMode("captain-applications", "cards");
  const [busyId, setBusyId] = useState("");
  const [msg, setMsg] = useState<{ tone: "success" | "error"; text: string } | null>(null);

  const data = useLiveData<{ applications: App[] }>(
    () => api<{ applications: App[] }>(`/api/v1/admin/captain-applications?status=${filter === "all" ? "" : filter}`),
    ["captain_application"],
    [filter],
  );
  const count = useLiveData<{ new: number }>(
    () => api<{ new: number }>("/api/v1/admin/captain-applications/count"),
    ["captain_application"],
  );
  const rows = data.data?.applications ?? [];

  const chips = FILTERS.map((f) => ({
    id: f,
    label: T.filter[f],
    count: f === "new" ? count.data?.new : undefined,
  }));

  async function setStatus(a: App, status: Status) {
    if (status === a.status || busyId) return;
    setBusyId(a.id);
    try {
      await api(`/api/v1/admin/captain-applications/${a.id}/status`, {
        method: "POST",
        body: JSON.stringify({ status }),
      });
      setMsg({ tone: "success", text: T.updated });
      data.reload();
      count.reload();
    } catch (e) {
      setMsg({ tone: "error", text: errorText(e) });
    } finally {
      setBusyId("");
    }
  }

  const cols: DataColumn<App>[] = [
    {
      id: "name",
      header: T.cols.name,
      primary: true,
      icon: <IconSignup size={14} />,
      cell: (a) => (
        <span className="font-bold">
          {a.full_name}
          <Badge variant={variantOf(a.status)} className="ms-2">
            {T.status[a.status]}
          </Badge>
        </span>
      ),
    },
    {
      id: "phone",
      header: T.cols.phone,
      cell: (a) => (
        <span className="flex flex-wrap items-center gap-2">
          <a dir="ltr" className="underline" href={`tel:${a.phone}`}>
            {a.phone}
          </a>
          <a className="underline" href={waHref(a.phone)} target="_blank" rel="noreferrer">
            {T.whatsapp}
          </a>
        </span>
      ),
    },
    { id: "age", header: T.cols.age, cell: (a) => a.age || "—" },
    { id: "area", header: T.cols.area, cell: (a) => a.area || "—" },
    { id: "ebike", header: T.cols.ebike, cell: (a) => yn(a.has_ebike) },
    { id: "motorbike", header: T.cols.motorbike, cell: (a) => yn(a.has_motorbike) },
    { id: "license", header: T.cols.license, cell: (a) => yn(a.has_license) },
    { id: "android", header: T.cols.android, cell: (a) => yn(a.has_android) },
    { id: "experience", header: T.cols.experience, hide: (a) => !a.experience, cell: (a) => a.experience },
    { id: "hours", header: T.cols.hours, hide: (a) => !a.hours, cell: (a) => a.hours },
    { id: "at", header: T.cols.at, cell: (a) => fmtDateTime(a.created_at) },
  ];

  return (
    <PageContainer>
      <PageHero icon={IconSignup} title={T.title} />
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
          getKey={(a) => a.id}
          columns={cols}
          view={view}
          empty={T.empty}
          actions={(a) =>
            canManage ? (
              <Select
                label={T.setStatus}
                value={a.status}
                disabled={busyId === a.id}
                onChange={(e) => void setStatus(a, e.target.value as Status)}
              >
                {STATUSES.map((s) => (
                  <option key={s} value={s}>
                    {T.status[s]}
                  </option>
                ))}
              </Select>
            ) : null
          }
        />
      )}
    </PageContainer>
  );
}
