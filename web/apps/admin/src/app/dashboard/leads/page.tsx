"use client";

/** طلبات انضمام المتاجر — واردة عبر روابط المندوبين؛ مراجعة ووسم الحالة. */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Button,
  Badge,
  DataView,
  ViewToggle,
  useViewMode,
  type DataColumn,
  IconStore,
  IconPhone,
  IconUser,
  IconPromos,
  IconStatus,
  IconLink,
  IconSuccess,
  IconBlock,
  IconUnblock,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);

interface Lead {
  id: string;
  store_name: string;
  owner_name: string;
  phone: string;
  area: string;
  note: string;
  rep_name: string | null;
  rep_code: string | null;
  status: "new" | "converted" | "rejected";
  created_at: string;
}

const STATUS_VARIANT: Record<Lead["status"], "warning" | "success" | "danger"> = {
  new: "warning",
  converted: "success",
  rejected: "danger",
};

const FILTERS: { key: string; label: string }[] = [
  { key: "", label: m.admin.leads.filterAll },
  { key: "new", label: m.admin.leads.st.new },
  { key: "converted", label: m.admin.leads.st.converted },
  { key: "rejected", label: m.admin.leads.st.rejected },
];

export default function LeadsPage() {
  const [leads, setLeads] = useState<Lead[]>([]);
  const [filter, setFilter] = useState("");
  const [view, setView] = useViewMode("leads", "cards");

  const load = useCallback(async () => {
    const q = filter ? `?status=${filter}` : "";
    setLeads(await api<Lead[]>(`/api/v1/admin/leads${q}`));
  }, [filter]);

  useEffect(() => {
    void load();
  }, [load]);

  async function setStatus(id: string, status: string) {
    await api(`/api/v1/admin/leads/${id}/status`, {
      method: "POST",
      body: JSON.stringify({ status }),
    });
    await load();
  }

  const columns: DataColumn<Lead>[] = [
    {
      id: "store",
      header: m.admin.leads.store,
      icon: <IconStore />,
      primary: true,
      cell: (l) => <span className="font-medium">{l.store_name}</span>,
    },
    {
      id: "owner",
      header: m.admin.leads.owner,
      icon: <IconUser />,
      cell: (l) => l.owner_name || "—",
    },
    {
      id: "phone",
      header: m.admin.leads.phone,
      icon: <IconPhone />,
      primary: true,
      cell: (l) => (
        <a href={`https://wa.me/${l.phone.replace(/^0/, "963")}`} target="_blank" rel="noreferrer" dir="ltr" className="font-medium text-primary hover:underline" onClick={(e) => e.stopPropagation()}>
          {l.phone}
        </a>
      ),
    },
    {
      id: "area",
      header: m.admin.leads.area,
      cell: (l) => l.area || "—",
    },
    {
      id: "rep",
      header: m.admin.leads.rep,
      icon: <IconPromos />,
      cell: (l) =>
        l.rep_name ? (
          <span className="flex items-center gap-1.5">
            {l.rep_name}
            {l.rep_code && (
              <span dir="ltr" className="rounded-badge bg-accent/10 px-1.5 font-mono text-xs text-accent-dark">
                {l.rep_code}
              </span>
            )}
          </span>
        ) : (
          <span className="text-ink-muted">{m.admin.leads.noRep}</span>
        ),
    },
    {
      id: "status",
      header: m.admin.leads.status,
      icon: <IconStatus />,
      cell: (l) => <Badge variant={STATUS_VARIANT[l.status]}>{m.admin.leads.st[l.status]}</Badge>,
    },
    {
      id: "date",
      header: m.admin.leads.date,
      cell: (l) => (
        <span dir="ltr" className="text-xs text-ink-muted">
          {new Date(l.created_at).toLocaleDateString("ar-SY")}
        </span>
      ),
    },
  ];

  return (
    <div>
      <div className="mb-2 flex flex-wrap items-center justify-between gap-3">
        <h1 className="flex items-center gap-2 text-2xl font-bold">
          <IconLink className="text-primary" />
          {m.admin.leads.title}
        </h1>
        <ViewToggle
          view={view}
          onChange={setView}
          tableLabel={m.common.viewTable}
          cardsLabel={m.common.viewCards}
        />
      </div>
      <p className="mb-4 text-sm text-ink-muted">{m.admin.leads.subtitle}</p>

      <div className="mb-4 flex flex-wrap gap-2">
        {FILTERS.map((f) => (
          <button
            key={f.key}
            onClick={() => setFilter(f.key)}
            className={`rounded-control px-3 py-1.5 text-sm transition-colors ${
              filter === f.key
                ? "bg-primary font-medium text-white"
                : "bg-surface text-ink-muted hover:bg-page"
            }`}
          >
            {f.label}
          </button>
        ))}
      </div>

      <DataView
        items={leads}
        getKey={(l) => l.id}
        columns={columns}
        view={view}
        empty={m.admin.leads.empty}
        actions={(l) => (
          <>
            {l.note && (
              <span className="text-xs text-ink-muted" title={l.note}>
                {l.note.length > 40 ? l.note.slice(0, 40) + "…" : l.note}
              </span>
            )}
            {l.status !== "converted" && (
              <Button
                variant="secondary"
                onClick={() => void setStatus(l.id, "converted")}
                className="flex items-center gap-1.5 !text-success"
              >
                <IconSuccess size={15} />
                {m.admin.leads.markConverted}
              </Button>
            )}
            {l.status !== "rejected" ? (
              <Button
                variant="secondary"
                onClick={() => void setStatus(l.id, "rejected")}
                className="flex items-center gap-1.5 !text-danger"
              >
                <IconBlock size={15} />
                {m.admin.leads.markRejected}
              </Button>
            ) : (
              <Button
                variant="secondary"
                onClick={() => void setStatus(l.id, "new")}
                className="flex items-center gap-1.5"
              >
                <IconUnblock size={15} />
                {m.admin.leads.reopen}
              </Button>
            )}
          </>
        )}
      />
    </div>
  );
}
