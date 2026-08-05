"use client";

/** طلبات انضمام المتاجر — واردة عبر روابط المندوبين؛ مراجعة ووسم الحالة. */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtDate } from "@rahalgo/i18n";
import {
  useLiveRefresh,
  PageHeader,
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
  IconLocation,
  IconSuccess,
  IconBlock,
  IconUnblock,
  Modal,
  Input,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);

interface Lead {
  id: string;
  store_name: string;
  owner_name: string;
  phone: string;
  area: string;
  category_name: string | null;
  category_icon: string | null;
  lat: number | null;
  lng: number | null;
  rep_name: string | null;
  rep_code: string | null;
  status: "new" | "converted" | "rejected";
  /** ما كتبه المندوبُ حين أرسل — **يُقرأ قبل الردّ، فقد يكون فيه جوابُ سؤالك.** */
  note?: string;
  decision_note?: string;
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
  /**
   * **والقسمُ للمعلَّق لا للمنتهي.**
   *
   * كان يفتح على «الكلّ»، **فتبقى الفرصةُ المحوَّلة بينها بعد أن صارت متجراً**
   * — ويُقرأ العددُ عملاً باقياً وهو مُنجَز. **وبابٌ اسمُه «طلبات الانضمام»
   * يعرض من انضمّ فعلاً يفقد معناه.**
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-٠٣: «بعد الموافقة على طلب متجرٍ لا يجب أن يبقى بقسم
   * طلبات الانضمام — فالطلبُ قُبل والمتجرُ تحوّل إلى المتاجر».)
   *
   * **والمرشِّحُ يبقى** — من أراد المحوَّلةَ وجدها بضغطة، **ولا شيءَ يُحذف من
   * القاعدة.**
   */
  const [filter, setFilter] = useState("new");
  const [view, setView] = useViewMode("leads", "cards");
  /** الفرصةُ التي تُردّ الآن — **ولا تُردّ حتى تُكتب كلمة.** */
  const [rejecting, setRejecting] = useState<Lead | null>(null);
  const [note, setNote] = useState("");

  const load = useCallback(async () => {
    const q = filter ? `?status=${filter}` : "";
    setLeads(await api<Lead[]>(`/api/v1/admin/leads${q}`));
  }, [filter]);

  useEffect(() => {
    void load();
  }, [load]);

  useLiveRefresh(["lead"], load);

  async function setStatus(id: string, status: string, note = "") {
    await api(`/api/v1/admin/leads/${id}/status`, {
      method: "POST",
      body: JSON.stringify({ status, note }),
    });
    setRejecting(null);
    setNote("");
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
      id: "category",
      header: m.admin.leads.category,
      icon: <IconStore />,
      cell: (l) =>
        l.category_name ? (
          <span className="inline-flex items-center gap-1">
            <span>{l.category_icon}</span>
            {l.category_name}
          </span>
        ) : (
          "—"
        ),
    },
    {
      id: "area",
      header: m.admin.leads.area,
      cell: (l) => (
        <span className="inline-flex items-center gap-1.5">
          {l.area || "—"}
          {l.lat != null && l.lng != null && (
            <a
              href={`https://www.google.com/maps?q=${l.lat},${l.lng}`}
              target="_blank"
              rel="noreferrer"
              title={m.admin.leads.onMap}
              onClick={(e) => e.stopPropagation()}
              className="text-primary hover:text-primary-dark"
            >
              <IconLocation size={15} />
            </a>
          )}
        </span>
      ),
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
          {fmtDate(l.created_at)}
        </span>
      ),
    },
  ];

  return (
    <div>
      <div className="mb-2 flex flex-wrap items-center justify-between gap-3">
        <PageHeader icon={IconLink} title={m.admin.leads.title} />
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
                ? "bg-primary font-medium text-on-solid"
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
              /* **والردُّ يلزمه كلمة.**

                 كان يُردّ بضغطةٍ صامتة، فيصل المندوبَ «رُدّ طلبُ الانضمام»
                 واسمُ المتجر وحدَه — **فلا يعرف أمكرّرٌ هو أم خارج التغطية أم
                 الرقمُ خطأ.** فيلاحق عميلاً ميتاً، **أو يعيد إرسال الفرصة
                 نفسِها** فتُردّ ثانية.

                 **والقاعدةُ مفروضةٌ في كلّ ردٍّ سواه** — والفرصةُ وحدَها كانت
                 تُردّ صامتة. */
              <Button
                variant="secondary"
                onClick={() => setRejecting(l)}
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

      {rejecting && (
        <Modal
          open
          onClose={() => {
            setRejecting(null);
            setNote("");
          }}
          title={`${m.admin.leads.markRejected}: ${rejecting.store_name}`}
        >
          <div className="space-y-3">
            <p className="text-sm text-ink-muted">{m.admin.leads.rejectHint}</p>
            {/* **وما كتبه المندوبُ يُقرأ قبل الردّ** — قد يكون فيه جوابُ سؤالك. */}
            {rejecting.note && (
              <p className="rounded-control bg-page px-3 py-2 text-sm">
                <span className="text-ink-muted">{m.admin.leads.repNote}: </span>
                {rejecting.note}
              </p>
            )}
            <Input
              id="lead-reject-note"
              label={m.admin.leads.rejectNote}
              required
              value={note}
              onChange={(e) => setNote(e.target.value)}
            />
            <div className="flex justify-end gap-2">
              <Button
                variant="secondary"
                onClick={() => {
                  setRejecting(null);
                  setNote("");
                }}
              >
                {m.common.cancel}
              </Button>
              <Button
                variant="danger"
                disabled={!note.trim()}
                onClick={() => void setStatus(rejecting.id, "rejected", note.trim())}
              >
                {m.admin.leads.markRejected}
              </Button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
}
