"use client";

/**
 * **غرفةُ الطوارئ** — كلُّ الطوارئ في صندوقٍ واحد (قراراتُ المالك ٢٠٢٦-١٠-٠٤).
 *
 * # لماذا شاشةٌ لا إشعارٌ وحدَه
 *
 * الإشعارُ يمرّ في الشريط ويُقرأ مرّةً. **وطارئٌ يُقرأ ولا يُغلق طارئٌ نُسي**:
 * من سأل عنه بعد يومين لم يجد من يقول ماذا جرى.
 *
 * # وما صار
 *
 * **النوعُ مخزَّنٌ ويُرى** (حادث · عطل · ظرفٌ قاهر · إغلاقٌ طارئٌ لمتجر · توقّفُ المنصّة)،
 * **والحادثُ والعطلُ قبل الاستلام هنا أيضاً** — لم يكونا. **وطارئٌ بلا مستلِمٍ بعد مهلته
 * يحمرّ**، ولكلّ طارئٍ صفحتُه بخطوات الحلّ (`/dashboard/emergencies/{id}`).
 */

import { useState } from "react";
import Link from "next/link";
import { getMessages, defaultLocale, fmtRef, fmtDateTime, fmtSpan, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Tabs,
  PageContainer,
  Pagination,
  PageHeader,
  EmptyState,
  LoadingState,
  Card,
  useLiveData,
  IconWarning,
  IconPhone,
  IconStatus,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { kindVariant } from "@/components/admin/emergency/kinds";

const m = getMessages(defaultLocale);
const E = m.admin.emergencies;
const R = m.admin.emergencyRoom;

interface EmergencyListRow {
  id: string;
  driver_name: string | null;
  driver_phone: string;
  order_number: number | null;
  note: string;
  created_at: string;
  resolution: string;
  resolved_at: string | null;
  resolved_by: string;
  kind: keyof typeof R.kinds;
  stage: "" | keyof typeof R.stages;
  merchant_name: string;
  acknowledged_at: string | null;
  acknowledged_by: string;
  stale: boolean;
}

const since = (iso: string) =>
  fmtSpan(Math.max(0, (Date.now() - Date.parse(iso)) / 1000));

export default function EmergenciesPage() {
  const [page, setPage] = useState(1);
  const [tab, setTab] = useState<"open" | "resolved">("open");
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  const { data, reload } = useLiveData<{
    emergencies: EmergencyListRow[];
    total: number;
    per_page: number;
  }>(
    () => api(`/api/v1/admin/emergencies?page=${page}&status=${tab}`),
    ["driver", "order", "emergency"],
    [page, tab],
  );
  if (!data) return <LoadingState />;
  const list = data.emergencies ?? [];

  async function ack(id: string) {
    setBusy(id);
    setError("");
    try {
      await api(`/api/v1/admin/emergencies/${id}/ack`, { method: "POST", body: "{}" });
      reload();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy("");
    }
  }

  return (
    <PageContainer>
      <PageHeader icon={IconWarning} title={E.title} subtitle={R.hint} />

      <Tabs
        items={[
          { key: "open" as const, label: E.tabOpen, icon: IconWarning },
          { key: "resolved" as const, label: E.tabResolved, icon: IconStatus },
        ]}
        value={tab}
        onChange={(k) => {
          setTab(k);
          setPage(1);
        }}
        className="mb-4"
      />
      {error && <Alert className="mb-3">{error}</Alert>}

      {list.length === 0 ? (
        <EmptyState icon={IconStatus} title={E.empty} />
      ) : (
        <div className="space-y-3">
          {list.map((x) => (
            <Card key={x.id} tone={x.stale ? "danger" : "default"}>
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="min-w-0 space-y-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <Badge variant={kindVariant(x.kind)}>{R.kinds[x.kind] ?? x.kind}</Badge>
                    {x.stage && <Badge>{R.stages[x.stage]}</Badge>}
                    <span className="font-bold">
                      {x.driver_name || x.driver_phone || x.merchant_name || R.kinds[x.kind]}
                    </span>
                    {x.order_number !== null && (
                      <span className="text-sm text-ink-muted" dir="ltr">
                        #{fmtRef(x.order_number)}
                      </span>
                    )}
                  </div>
                  {x.note && <p className="text-sm">{x.note}</p>}
                  <p className="text-xs text-ink-muted" dir="ltr">
                    {fmtDateTime(x.created_at)}
                  </p>
                  {tab === "open" &&
                    (x.acknowledged_at ? (
                      <p className="text-xs text-success">{R.ackedBy.replace("{name}", x.acknowledged_by || "—")}</p>
                    ) : (
                      <p className={`text-xs ${x.stale ? "font-bold text-danger" : "text-warning"}`}>
                        {x.stale ? R.stale.replace("{t}", since(x.created_at)) : R.unacked}
                      </p>
                    ))}
                  {x.resolved_at && (
                    <p className="text-xs text-success">
                      {x.resolution || E.noResolution}
                      <span className="ms-2 text-ink-muted">
                        {E.by}: {x.resolved_by || "—"} · {fmtDateTime(x.resolved_at)}
                      </span>
                    </p>
                  )}
                </div>
                <div className="flex flex-wrap items-center gap-2">
                  {x.driver_phone && (
                    <a
                      href={`tel:${x.driver_phone}`}
                      className="inline-flex items-center gap-1.5 rounded-control border border-line px-3 py-1.5 text-sm"
                    >
                      <IconPhone size={15} />
                      <span dir="ltr">{x.driver_phone}</span>
                    </a>
                  )}
                  {tab === "open" && !x.acknowledged_at && (
                    <Button variant="danger" disabled={busy === x.id} onClick={() => void ack(x.id)}>
                      {R.ack}
                    </Button>
                  )}
                  <Link
                    href={`/dashboard/emergencies/${x.id}`}
                    className="inline-flex items-center rounded-control border border-line px-3 py-1.5 text-sm"
                  >
                    {R.open}
                  </Link>
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}

      {data.total > data.per_page && (
        <div className="mt-4 flex justify-center">
          <Pagination page={page} total={data.total} perPage={data.per_page} onChange={setPage} />
        </div>
      )}
    </PageContainer>
  );
}
